package raft

import (
	"encoding/json"
	"log/slog"
	"net"
)

type RaftRPC struct {
	Type string      `json:"type"`
	Args interface{} `json:"args"`
}

// RequestVoteArgs represents the arguments for a RequestVote RPC
type RequestVoteArgs struct {
	Term         int    `json:"term"`
	CandidateID  string `json:"candidateId"`
	LastLogIndex int    `json:"lastLogIndex"`
	LastLogTerm  int    `json:"lastLogTerm"`
}

// RequestVoteReply represents the reply for a RequestVote RPC
type RequestVoteReply struct {
	Term        int  `json:"term"`
	VoteGranted bool `json:"voteGranted"`
}

func handleConnection(conn net.Conn, server *Server) {
	defer conn.Close()

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		var rpc RaftRPC
		if err := decoder.Decode(&rpc); err != nil {
			slog.Error("Error decoding RPC:", "error", err)
			return
		}

		slog.Debug("Received RPC", "body", rpc)

		switch rpc.Type {
		case "RequestVote":
			if args, ok := rpc.Args.(map[string]interface{}); ok {
				requestVoteArgs := RequestVoteArgs{
					Term:         int(args["term"].(float64)),
					CandidateID:  args["candidateId"].(string),
					LastLogIndex: int(args["lastLogIndex"].(float64)),
					LastLogTerm:  int(args["lastLogTerm"].(float64)),
				}

				rr := RequestResponse[RequestVoteArgs, RequestVoteReply]{
					Request: requestVoteArgs,
					Done:    make(chan struct{}),
				}

				server.voteChannel <- rr
				<-rr.Done
				encoder.Encode(rr.Response)
			}
		}

	}

}

func (s *Server) RunTcp() {
	defer s.wg.Done()

	listener, err := net.Listen("tcp", s.config.SelfServer.Addr)

	if err != nil {
		slog.Error("Error starting TCP server", "error", err)
		panic("cannot start tcp server")
	}

	defer listener.Close()

	go func() {
		<-s.ctx.Done()
		listener.Close()
	}()

	slog.Info("Server started", "address", s.config.SelfServer.Addr)

	for {
		select {
		case <-s.ctx.Done():
			slog.Info("Shutting down TCP server")
			return
		default:
			conn, err := listener.Accept()

			if err != nil {
				slog.Info("TCP server stopped accepting connections", "error", err)
				return
			}

			go handleConnection(conn, s)
		}
	}
}
