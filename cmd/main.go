package main

import (
	"bpicori/raft"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func init() {
	os.Setenv("RAFT_SERVERS", "localhost:8080,localhost:8081,localhost:8082")
	os.Setenv("CURRENT_SERVER", "localhost:8080")

	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(jsonHandler)

	// Set the logger as the default logger
	slog.SetDefault(logger)
}

func main() {
	var (
		raftServers   string
		currentServer string
	)

	flag.StringVar(&raftServers, "servers", "", "Comma-separated list of Raft server addresses")
	flag.StringVar(&currentServer, "current", "", "Address of the current server")
	flag.Parse()

	// If not provided try to get from environment variables
	if raftServers == "" {
		raftServers = os.Getenv("RAFT_SERVERS")
	}
	if currentServer == "" {
		currentServer = os.Getenv("CURRENT_SERVER")
	}

	if raftServers == "" || currentServer == "" {
		slog.Error("Missing required flags")
		os.Exit(1)
	}

	os.Setenv("RAFT_SERVERS", raftServers)
	os.Setenv("CURRENT_SERVER", currentServer)

	server := raft.NewServer()

	if err := server.Start(); err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for termination signal
	<-sigChan

	fmt.Println("Shutting down...")
	server.Stop()
	fmt.Println("Server stopped")
}
