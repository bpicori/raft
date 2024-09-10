package raft

import (
	"encoding/json"
	"fmt"
)

func (s *Server) sendRequestVoteReqRpc(addr string, args RequestVoteArgs) error {
	conn, err := s.connectionPool.GetConnection(addr)
	if err != nil {
		return fmt.Errorf("error getting connection: %v", err)
	}

	// Create the RPC request
	rpcRequest := RaftRPC{
		Type: "RequestVoteReq",
		Args: args,
	}

	// Encode and send the request
	encoder := json.NewEncoder(conn)
	err = encoder.Encode(rpcRequest)
	if err != nil {
		return fmt.Errorf("error encoding request: %v", err)
	}

	return nil
}

func (s *Server) sendRequestVoteRespRpc(addr string, reply RequestVoteReply) error {
	conn, err := s.connectionPool.GetConnection(addr)
	if err != nil {
		return fmt.Errorf("error getting connection: %v", err)
	}

	// Create the RPC request
	rpcRequest := RaftRPC{
		Type: "RequestVoteResp",
		Args: reply,
	}

	// Encode and send the request
	encoder := json.NewEncoder(conn)
	err = encoder.Encode(rpcRequest)
	if err != nil {
		return fmt.Errorf("error encoding request: %v", err)
	}

	return nil
}
