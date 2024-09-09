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
	NodeID       string
	Term         int    `json:"term"`
	CandidateID  string `json:"candidateId"`
	LastLogIndex int    `json:"lastLogIndex"`
	LastLogTerm  int    `json:"lastLogTerm"`
}

// RequestVoteReply represents the reply for a RequestVote RPC
type RequestVoteReply struct {
	NodeID      string
	Term        int  `json:"term"`
	VoteGranted bool `json:"voteGranted"`
}

type AppendEntriesArgs struct {
	NodeID       string
	Term         int        `json:"term"`
	LeaderID     string     `json:"leaderId"`
	PrevLogIndex int        `json:"prevLogIndex"`
	PrevLogTerm  int        `json:"prevLogTerm"`
	Entries      []LogEntry `json:"entries"`
	LeaderCommit int        `json:"leaderCommit"`
}

type AppendEntriesReply struct {
	Term    int  `json:"term"`
	Success bool `json:"success"`
}

func handleConnection(conn net.Conn, server *Server) {
	defer conn.Close()

	decoder := json.NewDecoder(conn)

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

				server.eventLoop.requestVoteReqCh <- Event[RequestVoteArgs]{
					Type: RequestVoteReq,
					Data: requestVoteArgs,
				}
			}
		case "AppendEntries":
			if args, ok := rpc.Args.(map[string]interface{}); ok {
				entries := make([]LogEntry, 0)
				for _, entry := range args["entries"].([]interface{}) {
					entryMap := entry.(map[string]interface{})
					entries = append(entries, LogEntry{
						Term:    int(entryMap["term"].(float64)),
						Command: entryMap["command"].(string),
					})
				}

				appendEntriesArgs := AppendEntriesArgs{
					Term:         int(args["term"].(float64)),
					LeaderID:     args["leaderId"].(string),
					PrevLogIndex: int(args["prevLogIndex"].(float64)),
					PrevLogTerm:  int(args["prevLogTerm"].(float64)),
					Entries:      entries,
					LeaderCommit: int(args["leaderCommit"].(float64)),
				}

				if len(entries) > 0 {
					slog.Debug("Received AppendEntries RPC", "entries", entries)

					server.eventLoop.appendEntriesReqCh <- Event[AppendEntriesArgs]{
						Type: AppendEntriesReq,
						Data: appendEntriesArgs,
					}
				} else {
					slog.Debug("Received Heartbeat RPC", "leader", appendEntriesArgs.LeaderID)

					server.eventLoop.heartbeatReqCh <- Event[AppendEntriesArgs]{
						Type: HeartbeatReq,
						Data: appendEntriesArgs,
					}
				}
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
		slog.Info("Gracefully shutting down TCP server")
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
