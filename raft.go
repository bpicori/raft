package raft

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
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

type RequestVoteArgs struct {
	server       *Server
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

type RequestVoteReply struct {
	Term        int  // currentTerm, for candidate to update itself
	VoteGranted bool // true means candidate received vote
}

type ServerConfig struct {
	ID   string
	Addr string
}

type Config struct {
	Servers    map[string]ServerConfig
	SelfID     string
	SelfServer ServerConfig
}

type Server struct {
	// Server Port
	port int

	// Persistent state on all servers
	currentTerm int
	votedFor    int
	logEntry    []LogEntry

	// Volatile state on all servers
	commitIndex int
	lastApplied int

	// Volatile state on leaders
	nextIndex  []int
	matchIndex []int

	// the current server state
	state ServerState

	config Config

	// Channels for communication
	electionTimeout *time.Timer
	heartbeat       chan bool
}

// NewServer creates a new server with a random election timeout.
func NewServer() (*Server, error) {
	config, err := LoadConfig()

	if err != nil {
		return &Server{}, err
	}

	s := &Server{
		config:          config,
		state:           Follower,
		electionTimeout: time.NewTimer(randomTimeout()),
	}
	return s, nil
}

func LoadConfig() (Config, error) {
	serversStr := os.Getenv("RAFT_SERVERS")
	currentSrv := os.Getenv("CURRENT_SERVER")

	if serversStr == "" || currentSrv == "" {
		return Config{}, fmt.Errorf("RAFT_SERVERS and CURRENT_SERVER must be set")
	}

	servers := make(map[string]ServerConfig)
	var selfID string
	var selfServer ServerConfig

	for _, addr := range strings.Split(serversStr, ",") {
		host, port, err := net.SplitHostPort(addr)

		if err != nil {
			return Config{}, fmt.Errorf("invalid address format: %s", addr)
		}

		id := fmt.Sprintf("%s:%s", host, port)
		server := ServerConfig{ID: id, Addr: addr}
		servers[id] = server

		if addr == currentSrv {
			selfID = id
			selfServer = server
		}

	}

	if selfID == "" {
		return Config{}, fmt.Errorf("current server %s not found in RAFT_SERVERS", currentSrv)
	}

	return Config{
		Servers:    servers,
		SelfID:     selfID,
		SelfServer: selfServer,
	}, nil

}

// RequestVote is called by candidates to gather votes.
func (s *Server) RequestVote(args *RequestVoteArgs) (*RequestVoteReply, error) {
	// TODO: Implement RequestVote RPC.

	return nil, nil

}

// run is the main event loop for the server.
func (s *Server) run() {
	for {
		switch s.state {
		case Follower:
			s.runFollower()

		case Candidate:
			s.runCandidate()

		case Leader:
			s.runLeader()
		}
	}
}

func (s *Server) runFollower() {
	select {
	case <-s.electionTimeout.C:
		s.state = Candidate
	}
}

func (s *Server) runCandidate() {

	s.currentTerm = s.currentTerm + 1

	// send RequestVote RPCs to all other servers

}

func (s *Server) runLeader() {
	// TODO: Implement the leader state.
}

// randomTimeout returns a random number between 150ms and 300ms.
func randomTimeout() time.Duration {
	return time.Duration(150+rand.Intn(150)) * time.Microsecond
}
