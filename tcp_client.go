package raft

import (
	"fmt"
	"log/slog"
	"net"

	"google.golang.org/protobuf/proto"
)

// Helper function to send a protobuf message
func sendProtobufMessage(conn net.Conn, message proto.Message) error {
	data, err := proto.Marshal(message)
	if err != nil {
		slog.Debug("Error marshaling message", "error", err)
		return fmt.Errorf("error marshaling message: %v", err)
	}

	_, err = conn.Write(data)
	if err != nil {
		slog.Debug("Error sending data", "error", err)
		return fmt.Errorf("error sending data: %v", err)
	}

	return nil
}

func (s *Server) sendRequestVoteReqRpc(addr string, args *RequestVoteArgs) error {
	conn, err := s.connectionPool.GetConnection(addr)
	if err != nil {
		slog.Info("[TCP_CLIENT][sendRequestVoteReqRpc] Error getting connection", "addr", addr, "error", err)
		return fmt.Errorf("error getting connection: %v", err)
	}
	defer conn.Close()

	// Create the RPC request
	rpcRequest := &RaftRPC{
		Type: "RequestVoteReq",
		Args: &RaftRPC_RequestVoteArgs{RequestVoteArgs: args},
	}

	slog.Debug("[TCP_CLIENT][sendRequestVoteReqRpc] Sending RPC", "addr", addr, "args", args)

	// Encode and send the request
	if err := sendProtobufMessage(conn, rpcRequest); err != nil {
		return fmt.Errorf("error sending RequestVoteReq RPC: %v", err)
	}

	return nil
}

func (s *Server) sendRequestVoteRespRpc(addr string, reply *RequestVoteReply) error {
	conn, err := s.connectionPool.GetConnection(addr)
	if err != nil {
		slog.Debug("[TCP_CLIENT][sendRequestVoteRespRpc] Error getting connection", "addr", addr, "error", err)
		return fmt.Errorf("error getting connection: %v", err)
	}
	defer conn.Close()

	// Create the RPC request
	rpcRequest := &RaftRPC{
		Type: "RequestVoteResp",
		Args: &RaftRPC_RequestVoteReply{RequestVoteReply: reply},
	}

	slog.Debug("[TCP_CLIENT][sendRequestVoteRespRpc] Sending RPC", "addr", addr, "reply", reply)

	// Encode and send the request
	if err := sendProtobufMessage(conn, rpcRequest); err != nil {
		return fmt.Errorf("error sending RequestVoteResp RPC: %v", err)
	}

	return nil
}

func (s *Server) sendAppendEntriesReqRpc(addr string, args *AppendEntriesArgs) error {
	conn, err := s.connectionPool.GetConnection(addr)
	if err != nil {
		slog.Debug("[TCP_CLIENT] Error getting connection", "addr", addr, "error", err)
		return fmt.Errorf("error getting connection: %v", err)
	}
	defer conn.Close()

	// Create the RPC request
	rpcRequest := &RaftRPC{
		Type: "AppendEntriesReq",
		Args: &RaftRPC_AppendEntriesArgs{AppendEntriesArgs: args},
	}

	slog.Debug("[TCP_CLIENT] Sending AppendEntriesReq RPC", "addr", addr, "args", args)

	// Encode and send the request
	if err := sendProtobufMessage(conn, rpcRequest); err != nil {
		return fmt.Errorf("error sending AppendEntriesReq RPC: %v", err)
	}

	return nil
}

func (s *Server) sendAppendEntriesRespRpc(addr string, reply *AppendEntriesReply) error {
	conn, err := s.connectionPool.GetConnection(addr)
	if err != nil {
		slog.Debug("[TCP_CLIENT] Error getting connection", "addr", addr, "error", err)
		return fmt.Errorf("error getting connection: %v", err)
	}
	defer conn.Close()

	// Create the RPC request
	rpcRequest := &RaftRPC{
		Type: "AppendEntriesResp",
		Args: &RaftRPC_AppendEntriesReply{AppendEntriesReply: reply},
	}

	slog.Debug("[TCP_CLIENT] Sending AppendEntriesResp RPC", "addr", addr, "reply", reply)

	// Encode and send the request
	if err := sendProtobufMessage(conn, rpcRequest); err != nil {
		return fmt.Errorf("error sending AppendEntriesResp RPC: %v", err)
	}

	return nil
}

