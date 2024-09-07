package raft

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/rpc"
	"sync"
	"time"
)

type ServerState int

const (
	Follower ServerState = iota
	Candidate
	Leader
)

type LogEntry struct {
	Term    int
	Command interface{} // any command
}

type Server struct {
	// mu sync.Mutex
	mu sync.RWMutex

	// Persistent state on all servers
	currentTerm int
	votedFor    string
	logEntry    []LogEntry

	// Volatile state on all servers
	commitIndex int
	lastApplied int

	// Volatile state on leaders
	nextIndex  []int
	matchIndex []int

	// the current server state
	state ServerState

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

	s := &Server{
		config:          config,
		state:           Follower,
		currentTerm:     0,
		votedFor:        config.SelfID,
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
	s.wg.Wait()
}

func (s *Server) RunStateMachine() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			slog.Info("Shutting down state machine")
			return
		case <-s.heartbeat:
			slog.Info("Received heartbeat")
		default:
			switch s.state {
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
	s.state = Candidate
	s.currentTerm++
	s.votedFor = s.config.SelfID
	s.mu.Unlock()

	slog.Info("Starting election", "term", s.currentTerm)
	go s.runCandidate()
}

func (s *Server) runCandidate() {
	votes := 1
	majority := len(s.config.Servers)/2 + 1
	slog.Info("Running candidate", "votes", votes, "majority", majority)

	s.mu.Lock()
	startTerm := s.currentTerm
	lastLogIndex := len(s.logEntry) - 1
	var lastLogTerm int
	if lastLogIndex >= 0 {
		lastLogTerm = s.logEntry[lastLogIndex].Term
	}
	args := RequestVoteArgs{
		Term:         startTerm,
		CandidateID:  s.config.SelfID,
		LastLogIndex: lastLogIndex,
		LastLogTerm:  lastLogTerm,
	}
	s.mu.Unlock()

	for _, server := range s.config.Servers {
		if server.ID == s.config.SelfID {
			continue
		}
		go s.sendRequestVote(server, args)
	}

	electionTimeout := time.After(randomTimeout(150, 300))

	for {
		select {
		case <-s.ctx.Done():
			return
		case rr := <-s.voteChannel:
			s.mu.Lock()
			if s.state != Candidate {
				s.mu.Unlock()
				return
			}
			if rr.Request.Term != s.currentTerm {
				rr.Response = RequestVoteReply{
					Term:        s.currentTerm,
					VoteGranted: false,
				}
				s.mu.Unlock()
				// reply to the channel with negative response
				<-rr.Done
				continue
			}
			if rr.Response.Term > s.currentTerm {
				// Discover higher term, become follower
				s.mu.Unlock()
				s.becomeFollower(rr.Response.Term)
				return
			}
			if rr.Response.VoteGranted {
				votes++
				if votes >= majority {
					s.becomeLeader()
					s.mu.Unlock()
					return
				}
			}
			s.mu.Unlock()
		case <-electionTimeout:
			// Election timeout, start new election
			slog.Info("Election timeout from Candidate state, starting new election")
			s.startElection()
			return
		case <-s.heartbeat:
			// Received heartbeat from leader, become follower
			s.becomeFollower(s.currentTerm)
			return
		}
	}

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

	s.state = Leader
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

	s.state = Follower
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
