package main

import (
	"bpicori/raft"
	"bpicori/raft/pkgs/logger"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func init() {
	logger.LogSetup()
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

	slog.Info("Shutting down...")
	server.Stop()
	slog.Info("Server stopped")
}
