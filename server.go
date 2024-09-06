package raft

import (
	"encoding/json"
	"fmt"
	"log"
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
	// encoder := json.NewEncoder(conn)

	for {
		var rpc RaftRPC
		if err := decoder.Decode(&rpc); err != nil {
			fmt.Println("Error decoding RPC:", err)
			return
		}

		fmt.Printf("Received RPC: %+v\n", rpc)

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
				fmt.Println("Sending RequestVoteReply RPC")
				json.NewEncoder(conn).Encode(rr.Response)
			}
		}

	}

}

func StartServer(server *Server) {
	listener, err := net.Listen("tcp", server.config.SelfServer.Addr)

	if err != nil {
		panic("cannot start tcp server")
	}

	defer listener.Close()

	log.Printf("Server listening on %s", server.config.SelfServer.Addr)

	for {
		conn, err := listener.Accept()

		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}

		go handleConnection(conn, server)
	}
}
