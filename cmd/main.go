package main

import (
	"bpicori/raft"
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
