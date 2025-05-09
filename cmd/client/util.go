package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"

	"google.golang.org/protobuf/proto"
)

func openConnection(addr string) (net.Conn, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		slog.Error("Error getting connection", "error", err)
		return nil, err
	}

	return conn, nil
}

func sendReceiveRPC(addr string, req proto.Message, resp proto.Message) error {
	conn, err := openConnection(addr)
	if err != nil {
		return err
	}
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

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func printHelp() {
	fmt.Println("\nAvailable commands:")
	fmt.Println("  status            - Get the cluster status")
	fmt.Println("  set <key> <value> - Add a key to the storage")
	fmt.Println("  rm <key>          - Remove a key from the storage")
	fmt.Println("  clear/cls         - Clear the screen")
	fmt.Println("  help              - Show this help message")
	fmt.Println("  exit/quit         - Exit the CLI")
	fmt.Println()
}

func showUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s -servers=host1:port1,host2:port2,... [operation] [arguments...]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Operations:\n")
	fmt.Fprintf(os.Stderr, "  status            - Get the cluster status\n")
	fmt.Fprintf(os.Stderr, "  set <key> <value> - Add a key to the storage\n")
	fmt.Fprintf(os.Stderr, "  rm <key>          - Remove a key from the storage\n")
	fmt.Fprintf(os.Stderr, "If no operation is provided, the CLI will start in interactive mode.\n")
	flag.PrintDefaults()
}
