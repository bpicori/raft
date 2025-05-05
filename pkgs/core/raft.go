package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/dto"
)

type Role int

const (
	Follower Role = iota
	Candidate
	Leader
)

type ServerState struct {
	CurrentTerm  int32           `json:"currentTerm"`
	VotedFor     string          `json:"votedFor"`
	LogEntry     []*dto.LogEntry `json:"logEntry"`
	CommitLength int32           `json:"commitLength"`
}

func (s *ServerState) SaveToFile(serverId string, path string) error {
	fileName := fmt.Sprintf("%s.json", serverId)
	filePath := fmt.Sprintf("%s/%s", path, fileName)

	slog.Info("Saving state to file", "path", filePath)

	// Check if file already exists and overwrite
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	return encoder.Encode(s)
}

type Server struct {
	mu sync.RWMutex // lock when changing server state

	/* Fields that need to be persisted */
	currentTerm  int32           // latest term server has seen
	votedFor     string          // in election state, the candidateId that this server voted for
	logEntry     []*dto.LogEntry // log entries
	commitLength int32           // index of highest log entry known to be committed
	/* End of persisted fields */

	/* Volatile state on all servers */
	currentRole      Role             // current role of the server
	currentLeader    string           // the current leader
	votesReceivedMap sync.Map         // map of votes received from other servers
	sentLength       map[string]int32 // length of log entries sent to other servers
	ackedLength      map[string]int32 // length of log entries acknowledged by other servers
	/* End of volatile fields */

	config          config.Config  // cluster configuration
	eventLoop       *EventChannels // event loop
	electionTimeout *time.Timer    // timer for election timeout
	heartbeatTimer  *time.Timer    // timer for heartbeat

	/* Server lifecycle */
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
	/* End of server lifecycle */
}

// NewServer creates a new server with a random election timeout.
func NewServer() *Server {
	config, err := config.LoadConfig(false)
	if err != nil {
		panic(fmt.Sprintf("Error loading config %v", err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	currentTerm, votedFor, logEntry, commitLength := loadPersistedState(config)
	eventLoop := NewEventLoop()

	s := &Server{
		config:           config,
		eventLoop:        eventLoop,
		currentTerm:      currentTerm,  // should be fetched from persistent storage
		votedFor:         votedFor,     // should be fetched from persistent storage
		logEntry:         logEntry,     // should be fetched from persistent storage
		commitLength:     commitLength, // should be fetched from persistent storage
		currentRole:      Follower,
		currentLeader:    "",
		votesReceivedMap: sync.Map{},
		sentLength:       make(map[string]int32),
		ackedLength:      make(map[string]int32),
		electionTimeout:  time.NewTimer(randomTimeout(config.TimeoutMin, config.TimeoutMax)),
		ctx:              ctx,
		cancel:           cancel,
	}
	return s
}

func (s *Server) Start() error {
	s.wg.Add(2)
	go s.RunTcp()
	go s.RunStateMachine()
	return nil
}

func (s *Server) Stop() {
	s.cancel()
	s.PersistState()
	s.eventLoop.Close()
	s.wg.Wait()
}

func (s *Server) PersistState() {
	persistedState := ServerState{
		CurrentTerm:  s.currentTerm,
		VotedFor:     s.votedFor,
		LogEntry:     s.logEntry,
		CommitLength: s.commitLength,
	}

	err := persistedState.SaveToFile(s.config.SelfID, s.config.PersistentFilePath)
	if err != nil {
		slog.Error("Error saving state to file", "error", err)
	}

	slog.Info("State saved to file", "term", s.currentTerm, "votedFor", s.votedFor, "commitLength", s.commitLength)
}

func (s *Server) RunStateMachine() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			slog.Info("[STATE_MACHINE] Gracefully shutting down state machine")
			return
		default:
			switch s.currentRole {
			case Follower:
				slog.Debug("[STATE_MACHINE] Current role is Follower")
				s.runFollower()
			case Candidate:
				slog.Debug("[STATE_MACHINE] Current role is Candidate")
				s.runCandidate()
			case Leader:
				slog.Debug("[STATE_MACHINE] Current role is Leader")
				s.runLeader()
			}
		}
	}
}

func (s *Server) runFollower() {
	for {
		select {
		case <-s.ctx.Done():
			return

		case requestVoteReq := <-s.eventLoop.voteRequestChan:
			slog.Info(
				"[FOLLOWER] Received RequestVoteReq",
				"candidate", requestVoteReq.Data.CandidateId,
				"term", requestVoteReq.Data.Term,
				"lastLogIndex", requestVoteReq.Data.LastLogIndex,
				"lastLogTerm", requestVoteReq.Data.LastLogTerm)
			s.OnVoteRequest(requestVoteReq.Data)
			return

		case requestVoteRes := <-s.eventLoop.voteResponseChan:
			slog.Debug("[FOLLOWER] Received vote response from peer, discarding",
				"peer", requestVoteRes.Data.NodeId,
				"granted", requestVoteRes.Data.VoteGranted,
				"term", requestVoteRes.Data.Term)
			s.OnVoteResponse(requestVoteRes.Data)
			return

		case logRequest := <-s.eventLoop.logRequestChan:
			slog.Debug("[FOLLOWER] Received {LogRequest}", "leader", logRequest.Data.LeaderId)
			s.OnLogRequest(logRequest.Data)
			return

		case logResponse := <-s.eventLoop.logResponseChan:
			slog.Debug("[FOLLOWER] Received log response from peer",
				"peer", logResponse.Data.FollowerId,
				"term", logResponse.Data.Term,
				"ack", logResponse.Data.Ack,
				"success", logResponse.Data.Success)
			s.OnLogResponse(logResponse.Data)
			return

		case <-s.electionTimeout.C:
			slog.Info("[FOLLOWER] Election timeout from Follower state, starting new election")
			s.startElection()
			return
		}
	}
}

func (s *Server) startElection() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.currentRole = Candidate
	s.currentTerm += 1
	s.votedFor = s.config.SelfID
	s.votesReceivedMap.Clear()
	s.votesReceivedMap.Store(s.config.SelfID, true)

	slog.Info("[CANDIDATE] Starting election", "term", s.currentTerm)
}

func (s *Server) runCandidate() {
	votes := 1
	majority := len(s.config.Servers)/2 + 1
	lastTerm := int32(0)
	if len(s.logEntry) > 0 {
		lastTerm = s.logEntry[len(s.logEntry)-1].Term
	}

	slog.Info("[CANDIDATE] Running...",
		"term", s.currentTerm,
		"lastTerm", lastTerm,
		"votes", votes,
		"majority", majority)

	for _, follower := range s.otherServers() {
		go s.sendVoteRequest(follower.Addr, &dto.VoteRequest{
			NodeId:       s.config.SelfID,
			Term:         s.currentTerm,
			CandidateId:  s.config.SelfID,
			LastLogIndex: int32(len(s.logEntry)),
			LastLogTerm:  lastTerm,
		})
	}

	s.electionTimeout.Reset(randomTimeout(s.config.TimeoutMin, s.config.TimeoutMax))

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.electionTimeout.C:
			// Election timeout, start new election
			slog.Info("[CANDIDATE] Election timeout, starting new election")
			s.startElection()
			return

		case requestVoteReq := <-s.eventLoop.voteRequestChan:
			slog.Info("[CANDIDATE] Received RequestVoteReq",
				"candidate", requestVoteReq.Data.CandidateId,
				"term", requestVoteReq.Data.Term,
				"lastLogIndex", requestVoteReq.Data.LastLogIndex,
				"lastLogTerm", requestVoteReq.Data.LastLogTerm)
			s.OnVoteRequest(requestVoteReq.Data)

		case requestVoteResp := <-s.eventLoop.voteResponseChan:
			slog.Info("[CANDIDATE] Received vote response from",
				"peer", requestVoteResp.Data.NodeId,
				"granted", requestVoteResp.Data.VoteGranted,
				"term", requestVoteResp.Data.Term)
			s.OnVoteResponse(requestVoteResp.Data)

		case logRequest := <-s.eventLoop.logRequestChan:
			slog.Info("[CANDIDATE] Received heartbeat from leader", "leader", logRequest.Data.LeaderId)
			go s.OnLogRequest(logRequest.Data)

		case <-s.eventLoop.leaderElectedCh:
			slog.Info("[CANDIDATE] Received leader elected event")
			return
		}
	}
}

func (s *Server) becomeLeader() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.currentRole = Leader
	s.electionTimeout.Stop()
	s.currentLeader = s.config.SelfID

	truePtr := true
	s.eventLoop.leaderElectedCh <- Event[bool]{Data: &truePtr}

	for _, follower := range s.otherServers() {
		s.sentLength[follower.ID] = int32(len(s.logEntry))
		s.ackedLength[follower.ID] = 0

		go s.replicateLog(s.config.SelfID, follower.ID)
	}
}

func (s *Server) OnVoteRequest(requestVoteArgs *dto.VoteRequest) {
	slog.Info("Received vote request from candidate", "candidate", requestVoteArgs.CandidateId)
	cID := requestVoteArgs.CandidateId
	cTerm := requestVoteArgs.Term
	cLastLogIndex := requestVoteArgs.LastLogIndex
	cLastLogTerm := requestVoteArgs.LastLogTerm

	if cTerm > s.currentTerm {
		s.currentTerm = cTerm
		s.currentRole = Follower
		s.votedFor = ""
	}

	lastTerm := int32(0)

	if len(s.logEntry) > 0 {
		lastTerm = s.logEntry[len(s.logEntry)-1].Term
	}

	logOk := cLastLogTerm > lastTerm || (cLastLogTerm == lastTerm && cLastLogIndex >= int32(len(s.logEntry)))

	if cTerm == s.currentTerm && logOk && (s.votedFor == "" || s.votedFor == cID) {
		s.votedFor = cID
		go s.sendVoteResponse(cID, &dto.VoteResponse{
			NodeId:      s.config.SelfID,
			Term:        s.currentTerm,
			VoteGranted: true,
		})
	} else {
		go s.sendVoteResponse(cID, &dto.VoteResponse{
			NodeId:      s.config.SelfID,
			Term:        s.currentTerm,
			VoteGranted: false,
		})
	}
}

func (s *Server) OnVoteResponse(requestVoteReply *dto.VoteResponse) {
	voterId := requestVoteReply.NodeId
	voterTerm := requestVoteReply.Term
	voterVoteGranted := requestVoteReply.VoteGranted

	if s.currentRole == Candidate && voterTerm == s.currentTerm && voterVoteGranted {
		s.votesReceivedMap.Store(voterId, true)
		votesReceived := 0
		s.votesReceivedMap.Range(func(_, _ interface{}) bool {
			votesReceived++
			return true
		})

		majority := len(s.config.Servers)/2 + 1
		if votesReceived >= majority {
			s.becomeLeader()
		}
	} else if voterTerm > s.currentTerm {
		s.currentTerm = voterTerm
		s.currentRole = Follower
		s.votedFor = ""
		s.electionTimeout.Reset(randomTimeout(s.config.TimeoutMin, s.config.TimeoutMax))
	}
}

func (s *Server) otherServers() []config.ServerConfig {
	others := make([]config.ServerConfig, 0, len(s.config.Servers)-1)
	for _, server := range s.config.Servers {
		if server.ID != s.config.SelfID {
			others = append(others, server)
		}
	}
	return others
}

func (s *Server) runLeader() {
	for _, follower := range s.otherServers() {
		go s.replicateLog(s.config.SelfID, follower.ID)
	}

	s.heartbeatTimer = time.NewTimer(time.Duration(s.config.Heartbeat) * time.Millisecond)

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.heartbeatTimer.C:
			return
		}
	}
}

func (s *Server) replicateLog(leaderId string, followerId string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	prefixLength := s.sentLength[followerId]
	suffix := s.logEntry[prefixLength:]

	prefixTerm := int32(0)
	if prefixLength > 0 {
		prefixTerm = s.logEntry[prefixLength-1].Term
	}

	go s.sendLogRequest(followerId, &dto.LogRequest{
		LeaderId:     leaderId,
		Term:         s.currentTerm,
		PrefixLength: int32(prefixLength),
		PrefixTerm:   prefixTerm,
		Suffix:       int32(len(suffix)),
		LeaderCommit: s.commitLength,
	})
}

func (s *Server) OnLogRequest(logRequest *dto.LogRequest) {
	leaderId := logRequest.LeaderId
	term := logRequest.Term
	prefixLength := logRequest.PrefixLength
	prefixTerm := logRequest.PrefixTerm
	suffix := logRequest.Suffix
	leaderCommit := logRequest.LeaderCommit

	if term > s.currentTerm {
		s.currentTerm = term
		s.votedFor = ""
		s.electionTimeout.Reset(randomTimeout(s.config.TimeoutMin, s.config.TimeoutMax))
	}

	if term == s.currentTerm {
		s.currentRole = Follower
		s.currentLeader = leaderId
	}

	logOk := int(prefixLength) >= len(s.logEntry) && (prefixLength == 0 || s.logEntry[prefixLength-1].Term == prefixTerm)

	if term == s.currentTerm && logOk {
		go s.AppendEntries(prefixLength, leaderCommit, suffix)
		ack := prefixLength + suffix
		go s.sendLogResponse(leaderId, &dto.LogResponse{
			FollowerId: s.config.SelfID,
			Term:       s.currentTerm,
			Ack:        ack,
			Success:    true,
		})
	} else {
		go s.sendLogResponse(leaderId, &dto.LogResponse{
			FollowerId: s.config.SelfID,
			Term:       s.currentTerm,
			Ack:        0,
			Success:    false,
		})
	}
}

func (s *Server) OnLogResponse(logResponse *dto.LogResponse) {
	followerId := logResponse.FollowerId
	term := logResponse.Term
	ack := logResponse.Ack
	success := logResponse.Success

	if s.currentTerm == term && s.currentRole == Leader {
		if success && ack >= int32(s.ackedLength[followerId]) {
			s.sentLength[followerId] = ack
			s.ackedLength[followerId] = ack
			s.CommitLogEntries()
		} else if s.sentLength[followerId] > 0 {
			s.sentLength[followerId] -= 1
			s.replicateLog(s.config.SelfID, followerId)
		}
	} else if term > s.currentTerm {
		s.currentTerm = term
		s.currentRole = Follower
		s.votedFor = ""
		s.electionTimeout.Reset(randomTimeout(s.config.TimeoutMin, s.config.TimeoutMax))
	}
}

func (s *Server) AppendEntries(prefixLength int32, leaderCommit int32, suffix int32) {
	// TODO: implement
}

func (s *Server) CommitLogEntries() {
	// TODO: implement
}
