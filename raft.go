package raft

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/rpc"
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

	// Channels for communication
	electionTimeout *time.Timer
	heartbeat       chan bool
	voteChannel     chan RequestResponse[RequestVoteArgs, RequestVoteReply]

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

	s := &Server{
		config: config,

		currentTerm:  currentTerm,  // should be fetched from persistent storage
		votedFor:     votedFor,     // should be fetched from persistent storage
		logEntry:     logEntry,     // should be fetched from persistent storage
		commitLength: commitLength, // should be fetched from persistent storage

		currentRole:   Follower,
		currentLeader: "",
		votesReceived: make(map[string]bool),
		sentLength:    make(map[string]int),
		ackLength:     make(map[string]int),

		electionTimeout: time.NewTimer(randomTimeout(150, 300)),
		voteChannel:     make(chan RequestResponse[RequestVoteArgs, RequestVoteReply]),
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
		case <-s.heartbeat:
			slog.Info("Received heartbeat, resetting election timeout")
			s.electionTimeout.Reset(randomTimeout(150, 300))
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
	slog.Info("Running candidate", "votes", votes, "majority", majority)

	// startTerm := s.currentTerm
	// lastLogIndex := len(s.logEntry) - 1
	// var lastLogTerm int
	// if lastLogIndex >= 0 {
	// 	lastLogTerm = s.logEntry[lastLogIndex].Term
	// }
	// _ = RequestVoteArgs{
	// 	Term:         startTerm,
	// 	CandidateID:  s.config.SelfID,
	// 	LastLogIndex: lastLogIndex,
	// 	LastLogTerm:  lastLogTerm,
	// }
	// s.mu.Unlock()

	// for _, server := range s.config.Servers {
	// 	if server.ID == s.config.SelfID {
	// 		continue
	// 	}
	// 	// go s.sendRequestVote(server, args)
	// }

	// electionTimeout := time.After(randomTimeout(150, 300))

	// for {
	// 	select {
	// 	case <-s.ctx.Done():
	// 		return
	// 	case rr := <-s.voteChannel:
	// 		s.mu.Lock()
	// 		if s.currentRole != Candidate {
	// 			s.mu.Unlock()
	// 			return
	// 		}
	// 		if rr.Request.Term != s.currentTerm {
	// 			slog.Info("Received RequestVote response with different term, ignoring")
	// 			rr.Response = RequestVoteReply{
	// 				Term:        s.currentTerm,
	// 				VoteGranted: false,
	// 			}
	// 			s.mu.Unlock()
	// 			rr.Done <- struct{}{}

	// 			continue
	// 		}
	// 		if rr.Response.Term > s.currentTerm {
	// 			// Discover higher term, become follower
	// 			s.mu.Unlock()
	// 			s.becomeFollower(rr.Response.Term)
	// 			return
	// 		}
	// 		if rr.Response.VoteGranted {
	// 			votes++
	// 			if votes >= majority {
	// 				s.becomeLeader()
	// 				s.mu.Unlock()
	// 				return
	// 			}
	// 		}
	// 		s.mu.Unlock()
	// 	case <-electionTimeout:
	// 		// Election timeout, start new election
	// 		slog.Info("Election timeout from Candidate state, starting new election")
	// 		s.startElection()
	// 		return
	// 	case <-s.heartbeat:
	// 		// Received heartbeat from leader, become follower
	// 		s.becomeFollower(s.currentTerm)
	// 		return
	// 	}
	// }

}

func (s *Server) sendRequestVote(server ServerConfig, args RequestVoteArgs) {
	slog.Info("Sending RequestVote",
		"to", server.ID,
		"term", args.Term,
		"lastLogIndex", args.LastLogIndex,
		"lastLogTerm", args.LastLogTerm)

	var reply RequestVoteReply
	err := s.sendRPC(server.ID, "Raft.RequestVote", args, &reply)
	if err != nil {
		slog.Error("Error sending RequestVote RPC", "error", err, "to", server.ID)
		return
	}

	s.voteChannel <- RequestResponse[RequestVoteArgs, RequestVoteReply]{
		Request:  args,
		Response: reply,
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

func (s *Server) becomeFollower(term int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.currentRole = Follower
	s.currentTerm = term
	s.votedFor = ""
	s.electionTimeout.Reset(randomTimeout(150, 300))
}

func (s *Server) runLeader() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.heartbeat:
			s.sendHeartbeat()
		}
	}
}

func (s *Server) sendHeartbeat() {
	// TODO: Implement this
}

func (s *Server) sendRPC(serverID string, method string, args interface{}, reply interface{}) error {
	// In a real implementation, you would use actual RPC here.
	// This is a simplified version for demonstration purposes.

	// Find the server in the config
	var serverConfig ServerConfig
	for _, sc := range s.config.Servers {
		if sc.ID == serverID {
			serverConfig = sc
			break
		}
	}
	if serverConfig.ID == "" {
		return fmt.Errorf("server %s not found in config", serverID)
	}

	// Create a connection (in a real implementation, you might want to maintain a connection pool)
	conn, err := net.Dial("tcp", serverConfig.Addr)
	if err != nil {
		return fmt.Errorf("error connecting to server %s: %v", serverID, err)
	}
	defer conn.Close()

	// Create an RPC client
	client := rpc.NewClient(conn)
	defer client.Close()

	// Make the RPC call
	err = client.Call(method, args, reply)
	if err != nil {
		return fmt.Errorf("error making RPC call to %s.%s: %v", serverID, method, err)
	}

	return nil
}
