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
	"bpicori/raft/pkgs/http"
	"bpicori/raft/pkgs/logger"
	"bpicori/raft/pkgs/raft"
	"bpicori/raft/pkgs/storage"
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

	storage := storage.NewStorage(config.SelfID, config.PersistentFilePath)

	// initialize raft server
	server := raft.NewRaft(
		eventManager,
		storage,
		config,
		ctx,
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		server.Start()
	}()

	// initialize tcp server
	wg.Add(1)
	go func() {
		defer wg.Done()
		tcp.Start(config.SelfServer.Addr, eventManager, ctx)
	}()

	// initialize application
	wg.Add(1)
	applicationParam := &application.ApplicationParam{
		EventManager: eventManager,
		Context:      ctx,
		LogEntry:     server.LogEntry,
		CommitLength: server.CommitLength,
	}
	go func() {
		defer wg.Done()
		application.Start(applicationParam)
	}()

	// initialize http server
	httpServer := http.NewHttpServer(config, server, ctx)
	wg.Add(1)
	go func() {
		defer wg.Done()
		httpServer.Start()
	}()

	// set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for termination signal
	<-sigChan

	slog.Info("[MAIN] Received termination signal, shutting down...")
	cancel()
	slog.Info("[MAIN] Waiting for all goroutines to stop...")
	wg.Wait()
	slog.Info("[MAIN] All goroutines stopped")
}
