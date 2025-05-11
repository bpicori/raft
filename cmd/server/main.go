package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"bpicori/raft/pkgs/application"
	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/events"
	"bpicori/raft/pkgs/logger"
	"bpicori/raft/pkgs/raft"
	"bpicori/raft/pkgs/tcp"
)

func init() {
	logger.LogSetup()
}

func main() {
	// initialize config
	config, err := config.LoadConfig(false)
	if err != nil {
		panic(fmt.Sprintf("Error loading config %v", err))
	}

	// initialize event manager
	eventManager := events.NewEventManager()

	// initialize lifecycle
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}

	// initialize raft server
	server := raft.NewRaft(
		eventManager,
		config,
		ctx,
		&wg,
	)
	wg.Add(1)
	go server.Start()

	// initialize tcp server
	wg.Add(1)
	go tcp.Start(config.SelfServer.Addr, eventManager, ctx, &wg)

	// initialize application
	wg.Add(1)
	go application.Start(eventManager, ctx, &wg)

	// set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for termination signal
	<-sigChan

	slog.Info("[MAIN] Received termination signal, shutting down...")
	cancel()
	wg.Wait()
	slog.Info("[MAIN] Server stopped")
}
