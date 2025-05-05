package core

import (
	"bpicori/raft/pkgs/dto"
	"fmt"
	"log/slog"
	"net"
)

func (s *Server) sendVoteRequest(addr string, args *dto.VoteRequest) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		slog.Info("[TCP_CLIENT][sendVoteRequest] Error getting connection", "addr", addr, "error", err)
		return fmt.Errorf("error getting connection: %v", err)
	}
	defer conn.Close()

	// Create the RPC request
	rpcRequest := &dto.RaftRPC{
		Type: VoteRequest.String(),
		Args: &dto.RaftRPC_VoteRequest{VoteRequest: args},
	}


	// Encode and send the request
	if err := sendProtobufMessage(conn, rpcRequest); err != nil {
		return fmt.Errorf("error sending VoteRequest RPC: %v", err)
	}

	return nil
}

func (s *Server) sendVoteResponse(addr string, reply *dto.VoteResponse) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		slog.Debug("[TCP_CLIENT][sendVoteResponse] Error getting connection", "addr", addr, "error", err)
		return fmt.Errorf("error getting connection: %v", err)
	}
	defer conn.Close()

	// Create the RPC request
	rpcRequest := &dto.RaftRPC{
		Type: VoteResponse.String(),
		Args: &dto.RaftRPC_VoteResponse{VoteResponse: reply},
	}


	// Encode and send the request
	if err := sendProtobufMessage(conn, rpcRequest); err != nil {
		return fmt.Errorf("error sending VoteResponse RPC: %v", err)
	}

	return nil
}

func (s *Server) sendLogRequest(addr string, args *dto.LogRequest) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		slog.Debug("[TCP_CLIENT] Error getting connection", "addr", addr, "error", err)
		return fmt.Errorf("error getting connection: %v", err)
	}
	defer conn.Close()

	rpcRequest := &dto.RaftRPC{
		Type: LogRequest.String(),
		Args: &dto.RaftRPC_LogRequest{LogRequest: args},
	}

	if err := sendProtobufMessage(conn, rpcRequest); err != nil {
		return fmt.Errorf("error sending LogRequest RPC: %v", err)
	}

	return nil
}

func (s *Server) sendLogResponse(addr string, reply *dto.LogResponse) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		slog.Debug("[TCP_CLIENT] Error getting connection", "addr", addr, "error", err)
		return fmt.Errorf("error getting connection: %v", err)
	}
	defer conn.Close()

	rpcRequest := &dto.RaftRPC{
		Type: LogResponse.String(),
		Args: &dto.RaftRPC_LogResponse{LogResponse: reply},
	}

	if err := sendProtobufMessage(conn, rpcRequest); err != nil {
		return fmt.Errorf("error sending LogResponse RPC: %v", err)
	}

	return nil
}
