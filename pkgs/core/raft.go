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
	// mu sync.Mutex
	mu sync.RWMutex
	// Persistent state on all servers
	currentTerm  int32           // latest term server has seen
	votedFor     string          // candidateId that received vote in current term
	logEntry     []*dto.LogEntry // log entries
	commitLength int32           // index of highest log entry known to be committed
	// Volatile state on all servers
	currentRole   Role
	currentLeader string
	votesReceived sync.Map
	sentLength    map[string]int
	ackLength     map[string]int
	// cluster configuration
	config config.Config
	// event loop
	eventLoop *EventChannels
	// Channels for communication
	electionTimeout *time.Timer
	heartbeatTimer  *time.Timer
	// Server lifecycle
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

// NewServer creates a new server with a random election timeout.
func NewServer() *Server {
	config, err := config.LoadConfig(false)
	if err != nil {
		panic(fmt.Sprintf("Error loading config %v", err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	currentTerm, votedFor, logEntry, commitLength := LoadPersistedState(config)
	eventLoop := NewEventLoop()

	s := &Server{
		config:          config,
		eventLoop:       eventLoop,
		currentTerm:     currentTerm,  // should be fetched from persistent storage
		votedFor:        votedFor,     // should be fetched from persistent storage
		logEntry:        logEntry,     // should be fetched from persistent storage
		commitLength:    commitLength, // should be fetched from persistent storage
		currentRole:     Follower,
		currentLeader:   "",
		votesReceived:   sync.Map{},
		sentLength:      make(map[string]int),
		ackLength:       make(map[string]int),
		electionTimeout: time.NewTimer(randomTimeout(config.TimeoutMin, config.TimeoutMax)),
		ctx:             ctx,
		cancel:          cancel,
	}
	return s
}

func (s *Server) Start() error {
	s.wg.Add(2)
	go s.RunTcp()
	go s.RunStateMachine()
	// go s.RunHTTPServer()
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

func LoadPersistedState(config config.Config) (currentTerm int32, votedFor string, logEntry []*dto.LogEntry, commitLength int32) {
	fileName := fmt.Sprintf("%s.json", config.SelfID)
	filePath := fmt.Sprintf("%s/%s", config.PersistentFilePath, fileName)

	file, err := os.Open(filePath)
	if err != nil {
		// this is the first time the server is starting
		logEntry = make([]*dto.LogEntry, 0)
		return 0, "", logEntry, 0
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var state ServerState
	err = decoder.Decode(&state)
	if err != nil {
		return 0, "", nil, 0
	}

	if state.LogEntry == nil {
		state.LogEntry = make([]*dto.LogEntry, 0)
	}

	return state.CurrentTerm, state.VotedFor, state.LogEntry, state.CommitLength
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

		case requestVoteReq := <-s.eventLoop.requestVoteReqCh:
			slog.Info(
				"[FOLLOWER] Received RequestVoteReq",
				"candidate", requestVoteReq.Data.CandidateId,
				"term", requestVoteReq.Data.Term,
				"lastLogIndex", requestVoteReq.Data.LastLogIndex,
				"lastLogTerm", requestVoteReq.Data.LastLogTerm)
			s.OnRequestVoteReq(requestVoteReq.Data)

		case requestVoteRes := <-s.eventLoop.requestVoteRespCh:
			slog.Debug("[FOLLOWER] Received vote response from peer, discarding",
				"peer", requestVoteRes.Data.NodeId,
				"granted", requestVoteRes.Data.VoteGranted,
				"term", requestVoteRes.Data.Term)
			return

		case heartbeatReq := <-s.eventLoop.heartbeatReqCh:
			slog.Debug("[FOLLOWER] Received heartbeat, resetting election timeout")
			s.becomeFollower(s.currentTerm, heartbeatReq.Data.LeaderId)
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
	s.votesReceived.Clear()
	s.votesReceived.Store(s.config.SelfID, true)

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
		"votes", votes,
		"majority", majority)

	for _, peer := range s.config.Servers {
		if peer.ID == s.config.SelfID {
			continue
		}

		go s.sendRequestVoteReqRpc(peer.Addr, &dto.RequestVoteArgs{
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
			return // close this goroutine, will start new election in a new goroutine

		case requestVoteReq := <-s.eventLoop.requestVoteReqCh:
			slog.Info("[CANDIDATE] Received RequestVoteReq",
				"candidate", requestVoteReq.Data.CandidateId,
				"term", requestVoteReq.Data.Term,
				"lastLogIndex", requestVoteReq.Data.LastLogIndex,
				"lastLogTerm", requestVoteReq.Data.LastLogTerm)
			go s.OnRequestVoteReq(requestVoteReq.Data)

		case requestVoteResp := <-s.eventLoop.requestVoteRespCh:
			slog.Info("[CANDIDATE] Received vote response from",
				"peer", requestVoteResp.Data.NodeId,
				"granted", requestVoteResp.Data.VoteGranted,
				"term", requestVoteResp.Data.Term)
			go s.OnRequestVoteResp(requestVoteResp.Data)

		case appendEntriesRequest := <-s.eventLoop.heartbeatReqCh:
			slog.Info("[CANDIDATE] Received heartbeat from leader", "leader", appendEntriesRequest.Data.LeaderId)
			if appendEntriesRequest.Data.Term >= s.currentTerm {
				s.becomeFollower(appendEntriesRequest.Data.Term, appendEntriesRequest.Data.LeaderId)
				return // close this goroutine
			}
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

	for _, peer := range s.config.Servers {
		if peer.ID == s.config.SelfID {
			continue
		}

		s.sentLength[peer.ID] = len(s.logEntry)
		s.ackLength[peer.ID] = 0

		prevLogTerm := int32(0)
		if len(s.logEntry) > 0 {
			prevLogTerm = s.logEntry[len(s.logEntry)-1].Term
		}

		prevLogIndex := int32(0)
		if len(s.logEntry) > 0 {
			prevLogIndex = int32(len(s.logEntry)) - 1
		}

		go s.sendAppendEntriesReqRpc(peer.Addr, &dto.AppendEntriesArgs{
			Term:         s.currentTerm,
			LeaderId:     s.config.SelfID,
			PrevLogIndex: prevLogIndex,
			PrevLogTerm:  prevLogTerm,
			Entries:      s.logEntry,
			LeaderCommit: s.commitLength,
		})
	}
}

func (s *Server) OnRequestVoteReq(requestVoteArgs *dto.RequestVoteArgs) {
	// Received vote request from candidate
	slog.Info("Received vote request from candidate", "candidate", requestVoteArgs.CandidateId)
	cTerm := requestVoteArgs.Term
	cID := requestVoteArgs.CandidateId
	cLastLogIndex := requestVoteArgs.LastLogIndex
	cLastLogTerm := requestVoteArgs.LastLogTerm

	if cTerm > s.currentTerm {
		s.becomeFollower(cTerm, "")
		go s.sendRequestVoteRespRpc(cID, &dto.RequestVoteReply{
			NodeId:      s.config.SelfID,
			Term:        s.currentTerm,
			VoteGranted: true,
		})

		return
	}

	// get the last term from the log
	lastTerm := int32(0)
	if len(s.logEntry) > 0 {
		lastTerm = s.logEntry[len(s.logEntry)-1].Term
	}

	logOk := cLastLogTerm > lastTerm || (cLastLogTerm == lastTerm && cLastLogIndex >= int32(len(s.logEntry)))

	if cTerm == s.currentTerm && logOk && (s.votedFor == "" || s.votedFor == cID) {
		go s.sendRequestVoteRespRpc(cID, &dto.RequestVoteReply{
			NodeId:      s.config.SelfID,
			Term:        s.currentTerm,
			VoteGranted: true,
		})

		s.votedFor = cID
	} else {
		go s.sendRequestVoteRespRpc(cID, &dto.RequestVoteReply{
			NodeId:      s.config.SelfID,
			Term:        s.currentTerm,
			VoteGranted: false,
		})
	}
}

func (s *Server) OnRequestVoteResp(requestVoteReply *dto.RequestVoteReply) {
	if requestVoteReply.Term > s.currentTerm {
		s.becomeFollower(requestVoteReply.Term, "")
		return
	}

	if s.currentRole != Candidate {
		return
	}

	if requestVoteReply.VoteGranted {
		s.votesReceived.Store(requestVoteReply.NodeId, true)
	}

	votes := 0
	s.votesReceived.Range(func(_, _ interface{}) bool {
		votes++
		return true
	})

	majority := len(s.config.Servers)/2 + 1
	if votes >= majority {
		slog.Info("Received majority votes, becoming leader")
		s.becomeLeader()
	}
}

func (s *Server) becomeFollower(term int32, leader string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.currentRole = Follower
	s.currentTerm = term
	s.currentLeader = leader
	s.votedFor = ""
	s.electionTimeout.Reset(randomTimeout(s.config.TimeoutMin, s.config.TimeoutMax))
}

func (s *Server) runLeader() {

	for _, peer := range s.config.Servers {
		if peer.ID == s.config.SelfID {
			continue
		}

		s.sentLength[peer.ID] = len(s.logEntry)
		s.ackLength[peer.ID] = 0

		prevLogTerm := int32(0)
		if len(s.logEntry) > 0 {
			prevLogTerm = s.logEntry[len(s.logEntry)-1].Term
		}

		prevLogIndex := int32(0)
		if len(s.logEntry) > 0 {
			prevLogIndex = int32(len(s.logEntry) - 1)
		}

		go s.sendAppendEntriesReqRpc(peer.Addr, &dto.AppendEntriesArgs{
			Term:         s.currentTerm,
			PrevLogIndex: prevLogIndex,
			PrevLogTerm:  prevLogTerm,
			Entries:      s.logEntry,
			LeaderCommit: s.commitLength,
			LeaderId:     s.config.SelfID,
		})
	}

	s.heartbeatTimer = time.NewTimer(time.Duration(s.config.Heartbeat) * time.Millisecond)

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.heartbeatTimer.C:
			return
		case requestVoteReq := <-s.eventLoop.requestVoteReqCh:
			slog.Info("[LEADER] Received RequestVoteReq", "candidate", requestVoteReq.Data.CandidateId)
			if requestVoteReq.Data.Term > s.currentTerm {
				s.becomeFollower(requestVoteReq.Data.Term, "")
				return
			}
		case heartbeatReq := <-s.eventLoop.heartbeatReqCh:
			slog.Info("[LEADER] Received heartbeat from another leader", "another_leader", heartbeatReq.Data.LeaderId)
			if heartbeatReq.Data.Term > s.currentTerm {
				s.becomeFollower(heartbeatReq.Data.Term, heartbeatReq.Data.LeaderId)
				return
			}
		}
	}
}
