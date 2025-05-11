package raft

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"sync"
	"time"

	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/events"
	"bpicori/raft/pkgs/storage"
	"bpicori/raft/pkgs/tcp"
)

var (
	Follower  = consts.Follower
	Candidate = consts.Candidate
	Leader    = consts.Leader
)

var replyMap = struct {
	mu        sync.Mutex
	responses map[string]chan bool
}{
	responses: make(map[string]chan bool),
}

type Raft struct {
	mu sync.RWMutex // lock when changing server state

	/* Fields that need to be persisted */
	currentTerm  int32           // latest term server has seen
	votedFor     string          // in election state, the candidateId that this server voted for
	LogEntry     []*dto.LogEntry // log entries
	CommitLength int32           // index of highest log entry known to be committed
	/* End of persisted fields */

	/* Volatile state on all servers */
	currentRole      consts.Role      // current role of the server
	currentLeader    string           // the current leader
	votesReceivedMap sync.Map         // map of votes received from other servers
	sentLength       map[string]int32 // length of log entries sent to other servers
	ackedLength      map[string]int32 // length of log entries acknowledged by other servers
	/* End of volatile fields */

	config       config.Config        // cluster configuration
	eventManager *events.EventManager // event manager

	/* Lifecycle */
	ctx context.Context
	wg  *sync.WaitGroup
	/* End of lifecycle */
}

// NewRaft creates a new server with a random election timeout.
func NewRaft(
	eventManager *events.EventManager,
	config config.Config,
	ctx context.Context,
	wg *sync.WaitGroup,
) *Raft {
	state, err := storage.LoadStateMachine(config.SelfID, config.PersistentFilePath)
	if err != nil {
		panic(fmt.Sprintf("Error loading state machine %v", err))
	}

	s := &Raft{
		config:           config,
		eventManager:     eventManager,
		currentTerm:      state.CurrentTerm,  // should be fetched from persistent storage
		votedFor:         state.VotedFor,     // should be fetched from persistent storage
		LogEntry:         state.LogEntry,     // should be fetched from persistent storage
		CommitLength:     state.CommitLength, // should be fetched from persistent storage
		currentRole:      Follower,
		currentLeader:    "",
		votesReceivedMap: sync.Map{},
		sentLength:       make(map[string]int32),
		ackedLength:      make(map[string]int32),
		ctx:              ctx,
		wg:               wg,
	}
	s.eventManager.ResetElectionTimer(randomTimeout(s.config.TimeoutMin, s.config.TimeoutMax))
	return s
}

func (s *Raft) PersistState() {
	persistedState := dto.StateMachineState{
		CurrentTerm:  s.currentTerm,
		VotedFor:     s.votedFor,
		LogEntry:     s.LogEntry,
		CommitLength: s.CommitLength,
	}

	err := storage.PersistStateMachine(s.config.SelfID, s.config.PersistentFilePath, &persistedState)
	if err != nil {
		slog.Error("Error saving state to file", "error", err)
	}

	slog.Info("State saved to file", "term", s.currentTerm, "votedFor", s.votedFor, "commitLength", s.CommitLength)
}

func (s *Raft) Start() {
	for {
		select {
		case <-s.ctx.Done():
			slog.Info("[STATE_MACHINE] Gracefully shutting down state machine")
			s.PersistState()
			s.wg.Done()
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

func (s *Raft) runFollower() {
	for {
		select {
		case <-s.ctx.Done():
			slog.Debug("[FOLLOWER] Context done, shutting down follower")
			return

		case requestVoteReq := <-s.eventManager.VoteRequestChan:
			slog.Info(
				"[FOLLOWER] Received RequestVoteReq",
				"candidate", requestVoteReq.CandidateId,
				"term", requestVoteReq.Term,
				"lastLogIndex", requestVoteReq.LastLogIndex,
				"lastLogTerm", requestVoteReq.LastLogTerm)
			s.onVoteRequest(requestVoteReq)
			if s.currentRole != Follower {
				return
			}

		case requestVoteRes := <-s.eventManager.VoteResponseChan:
			slog.Debug("[FOLLOWER] Received vote response from peer, discarding",
				"peer", requestVoteRes.NodeId,
				"granted", requestVoteRes.VoteGranted,
				"term", requestVoteRes.Term)
			s.onVoteResponse(requestVoteRes)
			if s.currentRole != Follower {
				return
			}

		case logRequest := <-s.eventManager.LogRequestChan:
			slog.Debug("[FOLLOWER] Received {LogRequest}", "leader", logRequest.LeaderId)
			s.onLogRequest(logRequest)
			if s.currentRole != Follower {
				return
			}

		case logResponse := <-s.eventManager.LogResponseChan:
			slog.Debug("[FOLLOWER] Received log response from peer",
				"peer", logResponse.FollowerId,
				"term", logResponse.Term,
				"ack", logResponse.Ack,
				"success", logResponse.Success)
			s.onLogResponse(logResponse)
			if s.currentRole != Follower {
				return
			}
		case <-s.eventManager.ElectionTimerChan():
			slog.Info("[FOLLOWER] Election timeout from Follower state, starting new election")
			s.startElection()
			return
		case replyCh := <-s.eventManager.NodeStatusChan:
			replyCh <- &dto.NodeStatusResponse{
				NodeId:        s.config.SelfID,
				CurrentTerm:   s.currentTerm,
				VotedFor:      s.votedFor,
				CurrentRole:   consts.MapRoleToString(s.currentRole),
				CurrentLeader: s.currentLeader,
				CommitLength:  s.CommitLength,
				LogEntries:    s.LogEntry,
			}
		}
	}
}

func (s *Raft) runCandidate() {
	votes := 1
	majority := len(s.config.Servers)/2 + 1
	lastTerm := int32(0)
	if len(s.LogEntry) > 0 {
		lastTerm = s.LogEntry[len(s.LogEntry)-1].Term
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
					LastLogIndex: int32(len(s.LogEntry)),
					LastLogTerm:  lastTerm,
				},
			},
		}
		go tcp.SendAsyncRPC(follower.Addr, rpc)
	}

	s.eventManager.ResetElectionTimer(randomTimeout(s.config.TimeoutMin, s.config.TimeoutMax))

	for {
		select {
		case <-s.ctx.Done():
			slog.Debug("[CANDIDATE] Context done, shutting down candidate")
			return
		case <-s.eventManager.ElectionTimerChan():
			slog.Info("[CANDIDATE] Election timeout, starting new election")
			s.startElection()
			return

		case requestVoteReq := <-s.eventManager.VoteRequestChan:
			slog.Info("[CANDIDATE] Received RequestVoteReq",
				"candidate", requestVoteReq.CandidateId,
				"term", requestVoteReq.Term,
				"lastLogIndex", requestVoteReq.LastLogIndex,
				"lastLogTerm", requestVoteReq.LastLogTerm)
			s.onVoteRequest(requestVoteReq)
			if s.currentRole != Candidate {
				return
			}

		case requestVoteResp := <-s.eventManager.VoteResponseChan:
			slog.Info("[CANDIDATE] Received vote response from",
				"peer", requestVoteResp.NodeId,
				"granted", requestVoteResp.VoteGranted,
				"term", requestVoteResp.Term)
			s.onVoteResponse(requestVoteResp)
			if s.currentRole != Candidate {
				return
			}
		case logRequest := <-s.eventManager.LogRequestChan:
			slog.Info("[CANDIDATE] Received heartbeat from leader", "leader", logRequest.LeaderId)
			s.onLogRequest(logRequest)
			if s.currentRole != Candidate {
				return
			}

		case replyCh := <-s.eventManager.NodeStatusChan:
			replyCh <- &dto.NodeStatusResponse{
				NodeId:        s.config.SelfID,
				CurrentTerm:   s.currentTerm,
				VotedFor:      s.votedFor,
				CurrentRole:   consts.MapRoleToString(s.currentRole),
				CurrentLeader: s.currentLeader,
				CommitLength:  s.CommitLength,
				LogEntries:    s.LogEntry,
			}
		}
	}
}

func (s *Raft) runLeader() {
	for _, follower := range s.otherServers() {
		go s.replicateLog(s.config.SelfID, follower.ID)
	}

	s.eventManager.ResetHeartbeatTimer(time.Duration(s.config.Heartbeat) * time.Millisecond)

	for {
		select {
		case <-s.ctx.Done():
			slog.Debug("[LEADER] Context done, shutting down leader")
			return
		case <-s.eventManager.HeartbeatTimerChan():
			slog.Debug("[LEADER] Heartbeat timer triggered. Sending updates to followers.")
			for _, follower := range s.otherServers() {
				go s.replicateLog(s.config.SelfID, follower.ID)
			}
			s.eventManager.ResetHeartbeatTimer(time.Duration(s.config.Heartbeat) * time.Millisecond)
		case voteRequest := <-s.eventManager.VoteRequestChan:
			slog.Debug("[LEADER] Received {VoteRequest} from", "candidate", voteRequest.CandidateId)
			s.onVoteRequest(voteRequest)
			if s.currentRole != Leader {
				return
			}
		case voteResponse := <-s.eventManager.VoteResponseChan:
			slog.Debug("[LEADER] Received {VoteResponse} from", "peer", voteResponse.NodeId)
			s.onVoteResponse(voteResponse)
			if s.currentRole != Leader {
				return
			}
		case logRequest := <-s.eventManager.LogRequestChan:
			slog.Debug("[LEADER] Received {LogRequest} from", "leader", logRequest.LeaderId)
			s.onLogRequest(logRequest)
			if s.currentRole != Leader {
				return
			}
		case logResponse := <-s.eventManager.LogResponseChan:
			slog.Debug("[LEADER] Received {LogResponse} from", "peer", logResponse.FollowerId)
			s.onLogResponse(logResponse)
			if s.currentRole != Leader {
				return
			}
		case appendLogEntry := <-s.eventManager.AppendLogEntryChan:
			slog.Debug("[LEADER] Received {AppendLogEntry}", "command", appendLogEntry.Command, "uuid", appendLogEntry.Uuid)

			logEntry := dto.LogEntry{
				Term:    s.currentTerm,
				Command: appendLogEntry.Command,
				Uuid:    appendLogEntry.Uuid,
			}
			s.LogEntry = append(s.LogEntry, &logEntry)

			replyMap.mu.Lock()
			replyMap.responses[appendLogEntry.Uuid] = appendLogEntry.Reply
			replyMap.mu.Unlock()

			s.ackedLength[s.config.SelfID] = int32(len(s.LogEntry))
			for _, follower := range s.otherServers() {
				go s.replicateLog(s.config.SelfID, follower.ID)
			}
			s.eventManager.ResetHeartbeatTimer(time.Duration(s.config.Heartbeat) * time.Millisecond)
		case replyCh := <-s.eventManager.NodeStatusChan:
			replyCh <- &dto.NodeStatusResponse{
				NodeId:        s.config.SelfID,
				CurrentTerm:   s.currentTerm,
				VotedFor:      s.votedFor,
				CurrentRole:   consts.MapRoleToString(s.currentRole),
				CurrentLeader: s.currentLeader,
				CommitLength:  s.CommitLength,
				LogEntries:    s.LogEntry,
			}
		}
	}
}

func (s *Raft) startElection() {
	s.currentRole = Candidate
	s.currentTerm += 1
	s.votedFor = s.config.SelfID
	s.votesReceivedMap.Clear()
	s.votesReceivedMap.Store(s.config.SelfID, true)
}

func (s *Raft) onVoteRequest(requestVoteArgs *dto.VoteRequest) {
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

	if len(s.LogEntry) > 0 {
		lastTerm = s.LogEntry[len(s.LogEntry)-1].Term
	}

	logOk := cLastLogTerm > lastTerm || (cLastLogTerm == lastTerm && cLastLogIndex >= int32(len(s.LogEntry)))

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
			},
		}
		go tcp.SendAsyncRPC(cID, rpc)
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
		go tcp.SendAsyncRPC(cID, rpc)
	}
}

func (s *Raft) onVoteResponse(requestVoteReply *dto.VoteResponse) {
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
			s.eventManager.StopElectionTimer()

			for _, follower := range s.otherServers() {
				s.sentLength[follower.ID] = int32(len(s.LogEntry))
				s.ackedLength[follower.ID] = 0
				go s.replicateLog(s.config.SelfID, follower.ID)
			}
		}
	} else if voterTerm > s.currentTerm {
		s.currentTerm = voterTerm
		s.currentRole = Follower
		s.votedFor = ""
		s.eventManager.ResetElectionTimer(randomTimeout(s.config.TimeoutMin, s.config.TimeoutMax))
	}
}

func (s *Raft) onLogRequest(logRequest *dto.LogRequest) {
	leaderId := logRequest.LeaderId
	term := logRequest.Term
	prefixLength := logRequest.PrefixLength
	prefixTerm := logRequest.PrefixTerm
	suffix := logRequest.Suffix
	leaderCommit := logRequest.LeaderCommit

	if term > s.currentTerm {
		s.currentTerm = term
		s.votedFor = ""
		s.eventManager.ResetElectionTimer(randomTimeout(s.config.TimeoutMin, s.config.TimeoutMax))
	}

	if term == s.currentTerm {
		s.currentRole = Follower
		s.currentLeader = leaderId
		s.eventManager.ResetElectionTimer(randomTimeout(s.config.TimeoutMin, s.config.TimeoutMax))
	}

	logOk := len(s.LogEntry) >= int(prefixLength) && (prefixLength == 0 || s.LogEntry[prefixLength-1].Term == prefixTerm)

	if term == s.currentTerm && logOk {
		s.appendEntries(prefixLength, leaderCommit, suffix)
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
		go tcp.SendAsyncRPC(leaderId, rpc)
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
		go tcp.SendAsyncRPC(leaderId, rpc)
	}
}

func (s *Raft) onLogResponse(logResponse *dto.LogResponse) {
	followerId := logResponse.FollowerId
	term := logResponse.Term
	ack := logResponse.Ack
	success := logResponse.Success

	if s.currentTerm == term && s.currentRole == Leader {
		if success && ack >= int32(s.ackedLength[followerId]) {
			s.sentLength[followerId] = ack
			s.ackedLength[followerId] = ack
			s.commitLogEntries()
		} else if s.sentLength[followerId] > 0 {
			s.sentLength[followerId] -= 1
			s.replicateLog(s.config.SelfID, followerId)
		}
	} else if term > s.currentTerm {
		s.currentTerm = term
		s.currentRole = Follower
		s.votedFor = ""
		s.eventManager.ResetElectionTimer(randomTimeout(s.config.TimeoutMin, s.config.TimeoutMax))
	}
}

func (s *Raft) replicateLog(leaderId string, followerId string) {
	prefixLength := s.sentLength[followerId]
	suffix := s.LogEntry[prefixLength:]

	prefixTerm := int32(0)
	if prefixLength > 0 {
		prefixTerm = s.LogEntry[prefixLength-1].Term
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
				LeaderCommit: s.CommitLength,
			},
		},
	}
	go tcp.SendAsyncRPC(followerId, rpc)
}

func (s *Raft) appendEntries(prefixLength int32, leaderCommit int32, suffix []*dto.LogEntry) {
	suffixLength := int32(len(suffix))
	logLength := int32(len(s.LogEntry))

	// discard conflicting log entries
	if suffixLength > 0 && logLength > prefixLength {
		index := math.Min(float64(logLength), float64(prefixLength+suffixLength)) - 1
		logAtIndex := s.LogEntry[int(index)] // last log entry in the prefix
		suffixTerm := suffix[int32(index)-prefixLength].Term

		if logAtIndex.Term != suffixTerm {
			s.LogEntry = s.LogEntry[:prefixLength]
		}
	}

	// append new log entries if suffix is not empty
	if (prefixLength + suffixLength) > logLength {
		s.LogEntry = append(s.LogEntry, suffix...)
	}

	if leaderCommit > s.CommitLength {
		slog.Debug("[FOLLOWER] Leader commit is greater than follower commit", "leaderCommit", leaderCommit, "followerCommit", s.CommitLength)

		// deliver log entries to application
		for i := s.CommitLength; i < leaderCommit; i++ {
			slog.Debug("[FOLLOWER] Delivering log entry to application", "logEntry", s.LogEntry[i])
			replyMap.mu.Lock()
			ch := replyMap.responses[s.LogEntry[i].Uuid]
			if ch != nil {
				ch <- true
			} else {
				slog.Warn("[FOLLOWER] No channel found for log entry", "logEntry", s.LogEntry[i], "uuid", s.LogEntry[i].Uuid)
			}
			replyMap.mu.Unlock()
		}

		s.CommitLength = leaderCommit
	}
}

func (s *Raft) commitLogEntries() {
	majority := len(s.config.Servers)/2 + 1

	for s.CommitLength < int32(len(s.LogEntry)) {
		acks := 0
		for _, follower := range s.otherServers() {
			if s.ackedLength[follower.ID] >= s.CommitLength {
				acks++
			}
		}

		if acks >= majority {
			slog.Debug("[LEADER] Committing log entry", "logEntry", s.LogEntry[s.CommitLength])

			// deliver log entries to application
			logEntry := s.LogEntry[s.CommitLength]
			replyMap.mu.Lock()
			ch := replyMap.responses[logEntry.Uuid]

			if ch != nil {
				ch <- true
			} else {
				slog.Warn("No channel found for log entry", "logEntry", logEntry, "uuid", logEntry.Uuid)
			}
			delete(replyMap.responses, logEntry.Uuid)
			replyMap.mu.Unlock()
			s.CommitLength = s.CommitLength + 1
		} else {
			break
		}
	}
}

func randomTimeout(from int, to int) time.Duration {
	return time.Duration(rand.Intn(to-from)+from) * time.Millisecond
}

func (s *Raft) otherServers() []config.ServerConfig {
	others := make([]config.ServerConfig, 0, len(s.config.Servers)-1)
	for _, server := range s.config.Servers {
		if server.ID != s.config.SelfID {
			others = append(others, server)
		}
	}
	return others
}
