package core

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/events"
)

var (
	Follower  = consts.Follower
	Candidate = consts.Candidate
	Leader    = consts.Leader
)

type Server struct {
	mu sync.RWMutex // lock when changing server state

	/* Fields that need to be persisted */
	currentTerm  int32           // latest term server has seen
	votedFor     string          // in election state, the candidateId that this server voted for
	logEntry     []*dto.LogEntry // log entries
	commitLength int32           // index of highest log entry known to be committed
	/* End of persisted fields */

	/* Volatile state on all servers */
	currentRole      consts.Role      // current role of the server
	currentLeader    string           // the current leader
	votesReceivedMap sync.Map         // map of votes received from other servers
	sentLength       map[string]int32 // length of log entries sent to other servers
	ackedLength      map[string]int32 // length of log entries acknowledged by other servers
	/* End of volatile fields */

	config          config.Config // cluster configuration
	eventLoop       *events.EventManager // event loop
	electionTimeout *time.Timer   // timer for election timeout
	heartbeatTimer  *time.Timer   // timer for heartbeat

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
	eventLoop := events.NewEventManager()

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

		case requestVoteReq := <-s.eventLoop.VoteRequestChan:
			slog.Info(
				"[FOLLOWER] Received RequestVoteReq",
				"candidate", requestVoteReq.CandidateId,
				"term", requestVoteReq.Term,
				"lastLogIndex", requestVoteReq.LastLogIndex,
				"lastLogTerm", requestVoteReq.LastLogTerm)
			s.OnVoteRequest(&requestVoteReq)
			if s.currentRole != Follower {
				return
			}

		case requestVoteRes := <-s.eventLoop.VoteResponseChan:
			slog.Debug("[FOLLOWER] Received vote response from peer, discarding",
				"peer", requestVoteRes.NodeId,
				"granted", requestVoteRes.VoteGranted,
				"term", requestVoteRes.Term)
			s.OnVoteResponse(&requestVoteRes)
			if s.currentRole != Follower {
				return
			}

		case logRequest := <-s.eventLoop.LogRequestChan:
			slog.Debug("[FOLLOWER] Received {LogRequest}", "leader", logRequest.LeaderId)
			s.OnLogRequest(&logRequest)
			if s.currentRole != Follower {
				return
			}

		case logResponse := <-s.eventLoop.LogResponseChan:
			slog.Debug("[FOLLOWER] Received log response from peer",
				"peer", logResponse.FollowerId,
				"term", logResponse.Term,
				"ack", logResponse.Ack,
				"success", logResponse.Success)
			s.OnLogResponse(&logResponse)
			if s.currentRole != Follower {
				return
			}
		case <-s.electionTimeout.C:
			slog.Info("[FOLLOWER] Election timeout from Follower state, starting new election")
			s.startElection()
			return
		}
	}
}

func (s *Server) startElection() {
	s.currentRole = Candidate
	s.currentTerm += 1
	s.votedFor = s.config.SelfID
	s.votesReceivedMap.Clear()
	s.votesReceivedMap.Store(s.config.SelfID, true)
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
		rpc := &dto.RaftRPC{
			Type: consts.VoteRequest.String(),
			Args: &dto.RaftRPC_VoteRequest{
				VoteRequest: &dto.VoteRequest{
					NodeId:       s.config.SelfID,
					Term:         s.currentTerm,
					CandidateId:  s.config.SelfID,
					LastLogIndex: int32(len(s.logEntry)),
					LastLogTerm:  lastTerm,
				},
			},
		}
		go SendAsyncRPC(follower.Addr, rpc)
	}

	s.electionTimeout.Reset(randomTimeout(s.config.TimeoutMin, s.config.TimeoutMax))

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.electionTimeout.C:
			slog.Info("[CANDIDATE] Election timeout, starting new election")
			s.startElection()
			return

		case requestVoteReq := <-s.eventLoop.VoteRequestChan:
			slog.Info("[CANDIDATE] Received RequestVoteReq",
				"candidate", requestVoteReq.CandidateId,
				"term", requestVoteReq.Term,
				"lastLogIndex", requestVoteReq.LastLogIndex,
				"lastLogTerm", requestVoteReq.LastLogTerm)
			s.OnVoteRequest(&requestVoteReq)
			if s.currentRole != Candidate {
				return
			}

		case requestVoteResp := <-s.eventLoop.VoteResponseChan:
			slog.Info("[CANDIDATE] Received vote response from",
				"peer", requestVoteResp.NodeId,
				"granted", requestVoteResp.VoteGranted,
				"term", requestVoteResp.Term)
			s.OnVoteResponse(&requestVoteResp)
			if s.currentRole != Candidate {
				return
			}
		case logRequest := <-s.eventLoop.LogRequestChan:
			slog.Info("[CANDIDATE] Received heartbeat from leader", "leader", logRequest.LeaderId)
			s.OnLogRequest(&logRequest)
			if s.currentRole != Candidate {
				return
			}

		}
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
		rpc := &dto.RaftRPC{
			Type: consts.VoteResponse.String(),
			Args: &dto.RaftRPC_VoteResponse{
				VoteResponse: &dto.VoteResponse{
					NodeId:      s.config.SelfID,
					Term:        s.currentTerm,
					VoteGranted: true,
				},
			}}
		go SendAsyncRPC(cID, rpc)
	} else {
		rpc := &dto.RaftRPC{
			Type: consts.VoteResponse.String(),
			Args: &dto.RaftRPC_VoteResponse{
				VoteResponse: &dto.VoteResponse{
					NodeId:      s.config.SelfID,
					Term:        s.currentTerm,
					VoteGranted: false,
				},
			},
		}
		go SendAsyncRPC(cID, rpc)
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
			s.currentRole = Leader
			s.currentLeader = s.config.SelfID
			s.electionTimeout.Stop()

			for _, follower := range s.otherServers() {
				s.sentLength[follower.ID] = int32(len(s.logEntry))
				s.ackedLength[follower.ID] = 0
				go s.ReplicateLog(s.config.SelfID, follower.ID)
			}
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
		go s.ReplicateLog(s.config.SelfID, follower.ID)
	}

	s.heartbeatTimer = time.NewTimer(time.Duration(s.config.Heartbeat) * time.Millisecond)

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.heartbeatTimer.C:
			slog.Debug("[LEADER] Heartbeat timer triggered. Sending updates to followers.")
			for _, follower := range s.otherServers() {
				go s.ReplicateLog(s.config.SelfID, follower.ID)
			}
			s.heartbeatTimer.Reset(time.Duration(s.config.Heartbeat) * time.Millisecond)
		case voteRequest := <-s.eventLoop.VoteRequestChan:
			slog.Debug("[LEADER] Received {VoteRequest} from", "candidate", voteRequest.CandidateId)
			s.OnVoteRequest(&voteRequest)
			if s.currentRole != Leader {
				return
			}
		case voteResponse := <-s.eventLoop.VoteResponseChan:
			slog.Debug("[LEADER] Received {VoteResponse} from", "peer", voteResponse.NodeId)
			s.OnVoteResponse(&voteResponse)
			if s.currentRole != Leader {
				return
			}
		case logRequest := <-s.eventLoop.LogRequestChan:
			slog.Debug("[LEADER] Received {LogRequest} from", "leader", logRequest.LeaderId)
			s.OnLogRequest(&logRequest)
			if s.currentRole != Leader {
				return
			}
		case logResponse := <-s.eventLoop.LogResponseChan:
			slog.Debug("[LEADER] Received {LogResponse} from", "peer", logResponse.FollowerId)
			s.OnLogResponse(&logResponse)
			if s.currentRole != Leader {
				return
			}
		case setCommand := <-s.eventLoop.SetCommandChan:
			slog.Info("[LEADER] Received {SetCommand}", "key", setCommand.Key, "value", setCommand.Value)
			s.logEntry = append(s.logEntry, &dto.LogEntry{
				Term: s.currentTerm,
				Command: &dto.Command{
					Operation: "set",
					Key:       setCommand.Key,
					Value:     setCommand.Value,
				},
			})
			s.ackedLength[s.config.SelfID] = int32(len(s.logEntry))
			for _, follower := range s.otherServers() {
				go s.ReplicateLog(s.config.SelfID, follower.ID)
			}
			s.heartbeatTimer.Reset(time.Duration(s.config.Heartbeat) * time.Millisecond)
		}
	}
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
		s.electionTimeout.Reset(randomTimeout(s.config.TimeoutMin, s.config.TimeoutMax))
	}

	logOk := len(s.logEntry) >= int(prefixLength) && (prefixLength == 0 || s.logEntry[prefixLength-1].Term == prefixTerm)

	if term == s.currentTerm && logOk {
		s.AppendEntries(prefixLength, leaderCommit, suffix)
		ack := prefixLength + int32(len(suffix))
		rpc := &dto.RaftRPC{
			Type: consts.LogResponse.String(),
			Args: &dto.RaftRPC_LogResponse{
				LogResponse: &dto.LogResponse{
					FollowerId: s.config.SelfID,
					Term:       s.currentTerm,
					Ack:        ack,
					Success:    true,
				},
			},
		}
		go SendAsyncRPC(leaderId, rpc)
	} else {
		rpc := &dto.RaftRPC{
			Type: consts.LogResponse.String(),
			Args: &dto.RaftRPC_LogResponse{
				LogResponse: &dto.LogResponse{
					FollowerId: s.config.SelfID,
					Term:       s.currentTerm,
					Ack:        0,
					Success:    false,
				},
			},
		}
		go SendAsyncRPC(leaderId, rpc)
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
			s.ReplicateLog(s.config.SelfID, followerId)
		}
	} else if term > s.currentTerm {
		s.currentTerm = term
		s.currentRole = Follower
		s.votedFor = ""
		s.electionTimeout.Reset(randomTimeout(s.config.TimeoutMin, s.config.TimeoutMax))
	}
}

func (s *Server) ReplicateLog(leaderId string, followerId string) {
	prefixLength := s.sentLength[followerId]
	suffix := s.logEntry[prefixLength:]

	prefixTerm := int32(0)
	if prefixLength > 0 {
		prefixTerm = s.logEntry[prefixLength-1].Term
	}

	rpc := &dto.RaftRPC{
		Type: consts.LogRequest.String(),
		Args: &dto.RaftRPC_LogRequest{
			LogRequest: &dto.LogRequest{
				LeaderId:     leaderId,
				Term:         s.currentTerm,
				PrefixLength: int32(prefixLength),
				PrefixTerm:   prefixTerm,
				Suffix:       suffix,
				LeaderCommit: s.commitLength,
			},
		},
	}
	go SendAsyncRPC(followerId, rpc)
}

func (s *Server) AppendEntries(prefixLength int32, leaderCommit int32, suffix []*dto.LogEntry) {
	suffixLength := int32(len(suffix))
	logLength := int32(len(s.logEntry))

	// discard conflicting log entries
	if suffixLength > 0 && logLength > prefixLength {
		index := math.Min(float64(logLength), float64(prefixLength+suffixLength)) - 1
		logAtIndex := s.logEntry[int(index)] // last log entry in the prefix
		suffixTerm := suffix[int32(index)-prefixLength].Term

		if logAtIndex.Term != suffixTerm {
			s.logEntry = s.logEntry[:prefixLength]
		}
	}

	// append new log entries if suffix is not empty
	if (prefixLength + suffixLength) > logLength {
		s.logEntry = append(s.logEntry, suffix...)
	}

	if leaderCommit > s.commitLength {
		s.commitLength = leaderCommit
		// TODO: deliver log entries to application (run the command to update the state)
	}
}

func (s *Server) CommitLogEntries() {

	majority := len(s.config.Servers)/2 + 1

	for s.commitLength < int32(len(s.logEntry)) {
		acks := 0
		for _, follower := range s.otherServers() {
			if s.ackedLength[follower.ID] >= s.commitLength {
				acks++
			}
		}

		if acks >= majority {
			// TODO: deliver log entries to application
			s.commitLength = s.commitLength + 1
		} else {
			break
		}
	}
}
