package raft

import (
	"context"
	"fmt"
	"log/slog"
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
	// add a mutex to access server
	mu sync.Mutex

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
		case rr := <-s.voteChannel:

			args := rr.Request

			if args.Term < s.currentTerm {
				rr.Response = RequestVoteReply{
					Term:        s.currentTerm,
					VoteGranted: false,
				}
				close(rr.Done)
				continue
			}

			rr.Response = RequestVoteReply{
				Term:        s.currentTerm,
				VoteGranted: false,
			}
			close(rr.Done)
		}
	}
}

// RequestVote is called by candidates to gather votes.
// func (s *Server) RequestVote(args *RequestVoteArgs) (*RequestVoteReply, error) {
// 	// TODO: Implement RequestVote RPC.

// 	return nil, nil

// }

// // run is the main event loop for the server.
// func (s *Server) run() {
// 	for {
// 		switch s.state {
// 		case Follower:
// 			s.runFollower()

// 		case Candidate:
// 			s.runCandidate()

// 		case Leader:
// 			s.runLeader()
// 		}
// 	}
// }

// func (s *Server) runFollower() {
// 	select {
// 	case <-s.electionTimeout.C:
// 		s.state = Candidate
// 	}
// }

// func (s *Server) runCandidate() {

// 	s.currentTerm = s.currentTerm + 1

// 	// send RequestVote RPCs to all other servers

// }

// func (s *Server) runLeader() {
// 	// TODO: Implement the leader state.
// }
