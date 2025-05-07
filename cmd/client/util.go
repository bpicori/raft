package main

import (
	"fmt"
	"log/slog"
	"net"
	"os"

	"google.golang.org/protobuf/proto"
)

func openConnection(addr string) net.Conn {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		slog.Error("Error getting connection", "error", err)
		os.Exit(1)
	}

	return conn
}

func sendReceiveRPC(addr string, req proto.Message, resp proto.Message) error {
	conn := openConnection(addr)
	defer conn.Close()

	data, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	_, err = conn.Write(data)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}


	buffer := make([]byte, 4096) // Consider making buffer size configurable or dynamic
	n, err := conn.Read(buffer)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("received empty response")
	}

	err = proto.Unmarshal(buffer[:n], resp)
	if err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return nil
}