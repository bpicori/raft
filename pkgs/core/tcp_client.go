package core

import (
	"bpicori/raft/pkgs/dto"
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

func (s *Server) sendRequestVoteReqRpc(addr string, args *dto.RequestVoteArgs) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		slog.Info("[TCP_CLIENT][sendRequestVoteReqRpc] Error getting connection", "addr", addr, "error", err)
		return fmt.Errorf("error getting connection: %v", err)
	}
	defer conn.Close()

	// Create the RPC request
	rpcRequest := &dto.RaftRPC{
		Type: "RequestVoteReq",
		Args: &dto.RaftRPC_RequestVoteArgs{RequestVoteArgs: args},
	}

	slog.Debug("[TCP_CLIENT][sendRequestVoteReqRpc] Sending RPC", "addr", addr, "args", args)

	// Encode and send the request
	if err := sendProtobufMessage(conn, rpcRequest); err != nil {
		return fmt.Errorf("error sending RequestVoteReq RPC: %v", err)
	}

	return nil
}

func (s *Server) sendRequestVoteRespRpc(addr string, reply *dto.RequestVoteReply) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		slog.Debug("[TCP_CLIENT][sendRequestVoteRespRpc] Error getting connection", "addr", addr, "error", err)
		return fmt.Errorf("error getting connection: %v", err)
	}
	defer conn.Close()

	// Create the RPC request
	rpcRequest := &dto.RaftRPC{
		Type: "RequestVoteResp",
		Args: &dto.RaftRPC_RequestVoteReply{RequestVoteReply: reply},
	}

	slog.Debug("[TCP_CLIENT][sendRequestVoteRespRpc] Sending RPC", "addr", addr, "reply", reply)

	// Encode and send the request
	if err := sendProtobufMessage(conn, rpcRequest); err != nil {
		return fmt.Errorf("error sending RequestVoteResp RPC: %v", err)
	}

	return nil
}

func (s *Server) sendAppendEntriesReqRpc(addr string, args *dto.AppendEntriesArgs) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		slog.Debug("[TCP_CLIENT] Error getting connection", "addr", addr, "error", err)
		return fmt.Errorf("error getting connection: %v", err)
	}
	defer conn.Close()

	// Create the RPC request
	rpcRequest := &dto.RaftRPC{
		Type: "AppendEntriesReq",
		Args: &dto.RaftRPC_AppendEntriesArgs{AppendEntriesArgs: args},
	}

	slog.Debug("[TCP_CLIENT] Sending AppendEntriesReq RPC", "addr", addr, "args", args)

	// Encode and send the request
	if err := sendProtobufMessage(conn, rpcRequest); err != nil {
		return fmt.Errorf("error sending AppendEntriesReq RPC: %v", err)
	}

	return nil
}

func (s *Server) sendAppendEntriesRespRpc(addr string, reply *dto.AppendEntriesReply) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		slog.Debug("[TCP_CLIENT] Error getting connection", "addr", addr, "error", err)
		return fmt.Errorf("error getting connection: %v", err)
	}
	defer conn.Close()

	// Create the RPC request
	rpcRequest := &dto.RaftRPC{
		Type: "AppendEntriesResp",
		Args: &dto.RaftRPC_AppendEntriesReply{AppendEntriesReply: reply},
	}

	slog.Debug("[TCP_CLIENT] Sending AppendEntriesResp RPC", "addr", addr, "reply", reply)

	// Encode and send the request
	if err := sendProtobufMessage(conn, rpcRequest); err != nil {
		return fmt.Errorf("error sending AppendEntriesResp RPC: %v", err)
	}

	return nil
}

func createConnection(addr string) (net.Conn, error) {
	// Create a new connection
	newConn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("error connecting to peer: %v", err)
	}

	return newConn, nil
}
