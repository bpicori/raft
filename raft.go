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
	votesReceived map[string]bool
	sentLength    map[string]int
	ackLength     map[string]int

	// cluster configuration
	config Config

	// event loop
	eventLoop      *EventLoop
	connectionPool *ConnectionPool

	// Channels for communication
	electionTimeout *time.Timer

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
		config: config,

		eventLoop:      eventLoop,
		connectionPool: connectionPool,

		currentTerm:  currentTerm,  // should be fetched from persistent storage
		votedFor:     votedFor,     // should be fetched from persistent storage
		logEntry:     logEntry,     // should be fetched from persistent storage
		commitLength: commitLength, // should be fetched from persistent storage

		currentRole:   Follower,
		currentLeader: "",
		votesReceived: make(map[string]bool),
		sentLength:    make(map[string]int),
		ackLength:     make(map[string]int),

		electionTimeout: time.NewTimer(randomTimeout(500, 1000)),
		ctx:             ctx,
		cancel:          cancel,
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

	s.connectionPool.Close()
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
		fmt.Printf("Error saving state: %v\n", err)
	}
}

func LoadPersistedState(config Config) (currentTerm int, votedFor string, logEntry []LogEntry, commitLength int) {
	fileName := fmt.Sprintf("%s.json", config.SelfID)
	filePath := fmt.Sprintf("%s/%s", config.PersistentFilePath, fileName)

	file, err := os.Open(filePath)
	if err != nil {
		return 0, "", nil, 0
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var state ServerState
	err = decoder.Decode(&state)
	if err != nil {
		return 0, "", nil, 0
	}

	return state.CurrentTerm, state.VotedFor, state.LogEntry, state.CommitLength
}

func (s *Server) RunStateMachine() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			slog.Info("Gracefully shutting down state machine")
			return
		default:
			switch s.currentRole {
			case Follower:
				slog.Info("Running Follower")
				s.runFollower()
			case Candidate:
				slog.Info("Running Candidate")
				s.runCandidate()
			case Leader:
				slog.Info("Running Leader")
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
			slog.Info("[FOLLOWER] Received vote response from peer, discarding",
				"peer", requestVoteRes.Data.NodeID,
				"granted", requestVoteRes.Data.VoteGranted,
				"term", requestVoteRes.Data.Term)
		case <-s.eventLoop.heartbeatReqCh:
			slog.Info("Received heartbeat, resetting election timeout")
			// TODO: implement this
			s.electionTimeout.Reset(randomTimeout(500, 1000))
		case <-s.electionTimeout.C:
			slog.Info("Election timeout from Follower state, starting new election")
			s.startElection()
		}
	}
}

func (s *Server) startElection() {
	s.mu.Lock()
	s.currentRole = Candidate
	s.currentTerm += 1
	s.votedFor = s.config.SelfID
	s.votesReceived[s.config.SelfID] = true
	s.mu.Unlock()

	slog.Info("Starting election", "term", s.currentTerm)
	go s.runCandidate()
}

func (s *Server) runCandidate() {
	votes := 1
	majority := len(s.config.Servers)/2 + 1
	lastTerm := 0
	if len(s.logEntry) > 0 {
		lastTerm = s.logEntry[len(s.logEntry)-1].Term
	}
	slog.Info("Running candidate", "votes", votes, "majority", majority)

	for _, peer := range s.config.Servers {
		if peer.ID == s.config.SelfID {
			continue
		}

		go s.sendRequestVoteReqRpc(peer.Addr, RequestVoteArgs{
			Term:         s.currentTerm,
			CandidateID:  s.config.SelfID,
			LastLogIndex: len(s.logEntry),
			LastLogTerm:  lastTerm,
		})
	}

	electionTimeout := time.After(randomTimeout(500, 1000))

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-electionTimeout:
			// Election timeout, start new election
			slog.Info("Election timeout from Candidate state, starting new election")
			s.startElection()
			return // close this goroutine, will start new election in a new goroutine

		case requestVoteReq := <-s.eventLoop.requestVoteReqCh:
			slog.Info("[CANDIDATE] Received RequestVoteReq",
				"candidate", requestVoteReq.Data.CandidateID,
				"term", requestVoteReq.Data.Term,
				"lastLogIndex", requestVoteReq.Data.LastLogIndex,
				"lastLogTerm", requestVoteReq.Data.LastLogTerm)

			s.OnRequestVoteReq(requestVoteReq.Data)

		case requestVoteResp := <-s.eventLoop.requestVoteRespCh:
			slog.Info("[CANDIDATE] Received vote response from peer",
				"peer", requestVoteResp.Data.NodeID,
				"granted", requestVoteResp.Data.VoteGranted,
				"term", requestVoteResp.Data.Term)

			s.OnRequestVoteResp(requestVoteResp.Data)

		case appendEntriesRequest := <-s.eventLoop.heartbeatReqCh:
			// Received heartbeat from leader
			slog.Info("Received heartbeat from leader", "leader", appendEntriesRequest.Data.LeaderID)
			if appendEntriesRequest.Data.Term >= s.currentTerm {
				s.becomeFollower(appendEntriesRequest.Data.Term)
				return // close this goroutine
			}
		}

		// case requestVoteResp := <-s.eventLoop.requestVoteRespCh:
		// Received vote from peer

	}

}

func (s *Server) becomeLeader() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.currentRole = Leader
	// Initialize nextIndex and matchIndex
	// s.nextIndex = make([]int, len(s.config.Peers))
	// s.matchIndex = make([]int, len(s.config.Peers))
	// for i := range s.config.Peers {
	// 	s.nextIndex[i] = len(s.logEntry)
	// 	s.matchIndex[i] = 0
	// }
}

func (s *Server) OnRequestVoteReq(requestVoteArgs RequestVoteArgs) {
	// Received vote request from candidate
	slog.Info("Received vote request from candidate", "candidate", requestVoteArgs.CandidateID)
	cTerm := requestVoteArgs.Term
	cID := requestVoteArgs.CandidateID
	cLastLogIndex := requestVoteArgs.LastLogIndex
	cLastLogTerm := requestVoteArgs.LastLogTerm

	if cTerm > s.currentTerm {
		s.becomeFollower(cTerm)
		s.sendRequestVoteRespRpc(cID, RequestVoteReply{
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

	// verify if candidate's log is at least as up-to-date as receiver's log
	logOk := cLastLogTerm > lastTerm || (cLastLogTerm == lastTerm && cLastLogIndex >= len(s.logEntry))

	if cTerm == s.currentTerm && logOk && (s.votedFor == "" || s.votedFor == cID) {
		s.sendRequestVoteRespRpc(cID, RequestVoteReply{
			Term:        s.currentTerm,
			VoteGranted: true,
		})
	} else {
		s.sendRequestVoteRespRpc(cID, RequestVoteReply{
			Term:        s.currentTerm,
			VoteGranted: false,
		})
	}
}

func (s *Server) OnRequestVoteResp(requestVoteReply RequestVoteReply) {
	if requestVoteReply.Term > s.currentTerm {
		s.becomeFollower(requestVoteReply.Term)
		return
	}

	if s.currentRole != Candidate {
		return
	}

	if requestVoteReply.VoteGranted {
		s.votesReceived[requestVoteReply.NodeID] = true
	}
	slog.Info("[IMPORTANT] Received vote response from peer")

	votes := 0
	for _, vote := range s.votesReceived {
		if vote {
			votes++
		}
	}

	majority := len(s.config.Servers)/2 + 1
	if votes >= majority {
		slog.Info("Received majority votes, becoming leader")
		panic("Becoming leader, panic attack :)")
	}
}

func (s *Server) becomeFollower(term int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.currentRole = Follower
	s.currentTerm = term
	s.votedFor = ""
	s.electionTimeout.Reset(randomTimeout(500, 1000))
}

func (s *Server) runLeader() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.eventLoop.heartbeatReqCh:
			// s.sendHeartbeat()
		}
	}
}
