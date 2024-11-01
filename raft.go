package raft

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"
)

type Role int

const (
	Follower Role = iota
	Candidate
	Leader
)

type LogEntry struct {
	Term    int
	Command interface{} // any command
}

type ServerState struct {
	CurrentTerm  int        `json:"currentTerm"`
	VotedFor     string     `json:"votedFor"`
	LogEntry     []LogEntry `json:"logEntry"`
	CommitLength int        `json:"commitLength"`
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
	currentTerm  int        // latest term server has seen
	votedFor     string     // candidateId that received vote in current term
	logEntry     []LogEntry // log entries
	commitLength int        // index of highest log entry known to be committed
	// Volatile state on all servers
	currentRole   Role
	currentLeader string
	votesReceived sync.Map
	sentLength    map[string]int
	ackLength     map[string]int
	// cluster configuration
	config Config
	// event loop
	eventLoop      *EventChannels
	connectionPool *ConnectionPool
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
	config, err := LoadConfig()
	if err != nil {
		panic(fmt.Sprintf("Error loading config %v", err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	currentTerm, votedFor, logEntry, commitLength := LoadPersistedState(config)
	eventLoop := NewEventLoop()
	connectionPool := NewConnectionPool()

	s := &Server{
		config:          config,
		eventLoop:       eventLoop,
		connectionPool:  connectionPool,
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
	// s.connectionPool.Close()
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

func LoadPersistedState(config Config) (currentTerm int, votedFor string, logEntry []LogEntry, commitLength int) {
	fileName := fmt.Sprintf("%s.json", config.SelfID)
	filePath := fmt.Sprintf("%s/%s", config.PersistentFilePath, fileName)

	file, err := os.Open(filePath)
	if err != nil {
		// this is the first time the server is starting
		logEntry = make([]LogEntry, 0)
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
		state.LogEntry = make([]LogEntry, 0)
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
				"candidate", requestVoteReq.Data.CandidateID,
				"term", requestVoteReq.Data.Term,
				"lastLogIndex", requestVoteReq.Data.LastLogIndex,
				"lastLogTerm", requestVoteReq.Data.LastLogTerm)
			s.OnRequestVoteReq(requestVoteReq.Data)

		case requestVoteRes := <-s.eventLoop.requestVoteRespCh:
			slog.Debug("[FOLLOWER] Received vote response from peer, discarding",
				"peer", requestVoteRes.Data.NodeID,
				"granted", requestVoteRes.Data.VoteGranted,
				"term", requestVoteRes.Data.Term)
			return

		case heartbeatReq := <-s.eventLoop.heartbeatReqCh:
			slog.Debug("[FOLLOWER] Received heartbeat, resetting election timeout")
			s.becomeFollower(s.currentTerm, heartbeatReq.Data.LeaderID)
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
	lastTerm := 0
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

		go s.sendRequestVoteReqRpc(peer.Addr, RequestVoteArgs{
			NodeID:       s.config.SelfID,
			Term:         s.currentTerm,
			CandidateID:  s.config.SelfID,
			LastLogIndex: len(s.logEntry),
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
				"candidate", requestVoteReq.Data.CandidateID,
				"term", requestVoteReq.Data.Term,
				"lastLogIndex", requestVoteReq.Data.LastLogIndex,
				"lastLogTerm", requestVoteReq.Data.LastLogTerm)
			go s.OnRequestVoteReq(requestVoteReq.Data)

		case requestVoteResp := <-s.eventLoop.requestVoteRespCh:
			slog.Info("[CANDIDATE] Received vote response from",
				"peer", requestVoteResp.Data.NodeID,
				"granted", requestVoteResp.Data.VoteGranted,
				"term", requestVoteResp.Data.Term)
			go s.OnRequestVoteResp(requestVoteResp.Data)

		case appendEntriesRequest := <-s.eventLoop.heartbeatReqCh:
			slog.Info("[CANDIDATE] Received heartbeat from leader", "leader", appendEntriesRequest.Data.LeaderID)
			if appendEntriesRequest.Data.Term >= s.currentTerm {
				s.becomeFollower(appendEntriesRequest.Data.Term, appendEntriesRequest.Data.LeaderID)
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

	s.eventLoop.leaderElectedCh <- Event[bool]{Data: true}

	for _, peer := range s.config.Servers {
		if peer.ID == s.config.SelfID {
			continue
		}

		s.sentLength[peer.ID] = len(s.logEntry)
		s.ackLength[peer.ID] = 0

		prevLogTerm := 0
		if len(s.logEntry) > 0 {
			prevLogTerm = s.logEntry[len(s.logEntry)-1].Term
		}

		prevLogIndex := 0
		if len(s.logEntry) > 0 {
			prevLogIndex = len(s.logEntry) - 1
		}

		go s.sendAppendEntriesReqRpc(peer.Addr, AppendEntriesArgs{
			Term:         s.currentTerm,
			LeaderID:     s.config.SelfID,
			PrevLogIndex: prevLogIndex,
			PrevLogTerm:  prevLogTerm,
			Entries:      s.logEntry,
			LeaderCommit: s.commitLength,
		})
	}
}

func (s *Server) OnRequestVoteReq(requestVoteArgs RequestVoteArgs) {
	// Received vote request from candidate
	slog.Info("Received vote request from candidate", "candidate", requestVoteArgs.CandidateID)
	cTerm := requestVoteArgs.Term
	cID := requestVoteArgs.CandidateID
	cLastLogIndex := requestVoteArgs.LastLogIndex
	cLastLogTerm := requestVoteArgs.LastLogTerm

	if cTerm > s.currentTerm {
		s.becomeFollower(cTerm, "")
		go s.sendRequestVoteRespRpc(cID, RequestVoteReply{
			NodeID:      s.config.SelfID,
			Term:        s.currentTerm,
			VoteGranted: true,
		})

		return
	}

	// get the last term from the log
	lastTerm := 0
	if len(s.logEntry) > 0 {
		lastTerm = s.logEntry[len(s.logEntry)-1].Term
	}

	logOk := cLastLogTerm > lastTerm || (cLastLogTerm == lastTerm && cLastLogIndex >= len(s.logEntry))

	if cTerm == s.currentTerm && logOk && (s.votedFor == "" || s.votedFor == cID) {
		go s.sendRequestVoteRespRpc(cID, RequestVoteReply{
			NodeID:      s.config.SelfID,
			Term:        s.currentTerm,
			VoteGranted: true,
		})

		s.votedFor = cID
	} else {
		go s.sendRequestVoteRespRpc(cID, RequestVoteReply{
			NodeID:      s.config.SelfID,
			Term:        s.currentTerm,
			VoteGranted: false,
		})
	}
}

func (s *Server) OnRequestVoteResp(requestVoteReply RequestVoteReply) {
	if requestVoteReply.Term > s.currentTerm {
		s.becomeFollower(requestVoteReply.Term, "")
		return
	}

	if s.currentRole != Candidate {
		return
	}

	if requestVoteReply.VoteGranted {
		s.votesReceived.Store(requestVoteReply.NodeID, true)
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

func (s *Server) becomeFollower(term int, leader string) {
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

		prevLogTerm := 0
		if len(s.logEntry) > 0 {
			prevLogTerm = s.logEntry[len(s.logEntry)-1].Term
		}

		prevLogIndex := 0
		if len(s.logEntry) > 0 {
			prevLogIndex = len(s.logEntry) - 1
		}

		s.sendAppendEntriesReqRpc(peer.Addr, AppendEntriesArgs{
			Term:         s.currentTerm,
			LeaderID:     s.config.SelfID,
			PrevLogIndex: prevLogIndex,
			PrevLogTerm:  prevLogTerm,
			Entries:      s.logEntry,
			LeaderCommit: s.commitLength,
		})
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
