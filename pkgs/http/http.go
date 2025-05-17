package http

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/consts"
)

var INTERVAL = 1 * time.Second
var SHUTDOWN_TIMEOUT = 6 * time.Second

type RaftControllerNode interface {
	GetCurrentRole() consts.Role
}

type HttpServer struct {
	config       config.Config
	raftNode     RaftControllerNode
	ctx          context.Context
	server       *http.Server
	serverCancel context.CancelFunc
}

func NewHttpServer(config config.Config, raftNode RaftControllerNode, ctx context.Context) *HttpServer {
	return &HttpServer{
		config:       config,
		raftNode:     raftNode,
		ctx:          ctx,
		server:       nil,
		serverCancel: nil,
	}
}

func (s *HttpServer) Start() {
	slog.Info("[HTTP_CTRL] Starting HTTP server lifecycle manager")
	defer slog.Info("[HTTP_CTRL] HTTP server lifecycle manager stopped.")

	for {
		// Step 1: Decide action based on Raft role and current server state
		currentRole := s.raftNode.GetCurrentRole()
		shouldServerBeRunning := (currentRole == consts.Leader)

		if shouldServerBeRunning {
			// start server if not already running
			if s.server == nil {
				s.startNewHttpServer()
			}
		} else {
			if s.server != nil { // Server is running, but it shouldn't be. Stop it.
				slog.Debug("[HTTP_CTRL] Node is not Leader (or server shouldn't run) and server is active. Requesting shutdown.", "addr", s.server.Addr)
				s.serverCancel()
			}
		}

		// Step 2: Wait for next cycle or main context cancellation
		select {
		case <-s.ctx.Done():
			slog.Debug("[HTTP_CTRL] Main context cancelled. Shutting down lifecycle manager.")
			if s.serverCancel != nil {
				slog.Debug("[HTTP_CTRL] Requesting shutdown of active/pending HTTP server due to main context cancellation.")
				s.serverCancel()
				select {
				case <-time.After(SHUTDOWN_TIMEOUT):
					slog.Warn("[HTTP_CTRL] Timeout waiting for HTTP server instance to shut down during main context cancellation.")
				}
			}
			return
		case <-time.After(INTERVAL):
			// continue to next iteration
		}
	}
}

func runHttpServer(srv *http.Server, ctx context.Context) {
	slog.Debug("[HTTP_SERVER] Goroutine starting", "addr", srv.Addr)
	errChan := make(chan error, 1)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("[HTTP_SERVER] ListenAndServe error", "error", err, "addr", srv.Addr)
			errChan <- err
		} else {
			slog.Debug("[HTTP_SERVER] ListenAndServe exited cleanly (or via ErrServerClosed).", "addr", srv.Addr)
			errChan <- nil // Explicitly send nil for clean exit or ErrServerClosed
		}
	}()

	select {
	case <-ctx.Done(): // Context was cancelled by manageServerLifecycle
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("[HTTP_SERVER] Graceful shutdown error", "error", err, "addr", srv.Addr)
		}
	case err := <-errChan: // ListenAndServe exited on its own (error or completed)
		if err != nil {
			slog.Warn("[HTTP_SERVER] ListenAndServe terminated with an error before context cancellation.", "error", err, "addr", srv.Addr)
		}
	}
	slog.Debug("[HTTP_SERVER] Goroutine finished.", "addr", srv.Addr)
}

func (s *HttpServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.raftNode.GetCurrentRole() == consts.Leader {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "Raft Leader answering"}`))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status": "Not a Raft Leader"}`))
	}
}

func (s *HttpServer) startNewHttpServer() {
	slog.Debug("[HTTP_CTRL] Node is Leader and server is not active. Starting HTTP server.", "port", s.config.HTTPPort)
	serverInstanceCtx, serverInstanceCancel := context.WithCancel(context.Background())

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRoot)

	newServer := &http.Server{
		Addr:    ":" + s.config.HTTPPort,
		Handler: mux,
	}

	s.server = newServer
	s.serverCancel = serverInstanceCancel

	go runHttpServer(newServer, serverInstanceCtx)
	slog.Debug("[HTTP_CTRL] HTTP server instance goroutine launched.", "addr", newServer.Addr)
}
