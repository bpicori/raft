package raft

import (
	"encoding/json"
	"fmt"
	"log/slog"
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

	slog.Debug("[TCP_CLIENT] Sending RequestVoteReq RPC", "addr", addr, "args", args)

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

	slog.Debug("[TCP_CLIENT] Sending RequestVoteResp RPC", "addr", addr, "reply", reply)

	// Encode and send the request
	encoder := json.NewEncoder(conn)
	err = encoder.Encode(rpcRequest)
	if err != nil {
		return fmt.Errorf("error encoding request: %v", err)
	}

	return nil
}

func (s *Server) sendAppendEntriesReqRpc(addr string, args AppendEntriesArgs) error {
	conn, err := s.connectionPool.GetConnection(addr)
	if err != nil {
		return fmt.Errorf("error getting connection: %v", err)
	}

	// Create the RPC request
	rpcRequest := RaftRPC{
		Type: "AppendEntriesReq",
		Args: args,
	}

	slog.Debug("[TCP_CLIENT] Sending AppendEntriesReq RPC", "addr", addr, "args", args)

	// Encode and send the request
	encoder := json.NewEncoder(conn)
	err = encoder.Encode(rpcRequest)
	if err != nil {
		return fmt.Errorf("error encoding request: %v", err)
	}

	return nil
}

func (s *Server) sendAppendEntriesRespRpc(addr string, reply AppendEntriesReply) error {
	conn, err := s.connectionPool.GetConnection(addr)
	if err != nil {
		return fmt.Errorf("error getting connection: %v", err)
	}

	// Create the RPC request
	rpcRequest := RaftRPC{
		Type: "AppendEntriesResp",
		Args: reply,
	}

	slog.Debug("[TCP_CLIENT] Sending AppendEntriesResp RPC", "addr", addr, "reply", reply)

	// Encode and send the request
	encoder := json.NewEncoder(conn)
	err = encoder.Encode(rpcRequest)
	if err != nil {
		return fmt.Errorf("error encoding request: %v", err)
	}

	return nil
}
