package raft

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
)

func sendRPC(addr string, rpcRequest RaftRPC) (RaftRPC, error) {
	// Create a connection
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return RaftRPC{}, fmt.Errorf("error connecting to server %s: %v", addr, err)
	}
	defer conn.Close()

	// Create the RPC request

	// Encode and send the request
	encoder := json.NewEncoder(conn)
	err = encoder.Encode(rpcRequest)
	if err != nil {
		return RaftRPC{}, fmt.Errorf("error encoding request: %v", err)
	}

	// Receive and decode the response
	reader := bufio.NewReader(conn)
	decoder := json.NewDecoder(reader)

	var response RaftRPC
	err = decoder.Decode(&response)
	if err != nil {
		return RaftRPC{}, fmt.Errorf("error decoding response: %v", err)
	}

	return response, nil
}
