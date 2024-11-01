package main

import (
	"bpicori/raft"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func logSetup() {
	debugEnv := os.Getenv("DEBUG")
	logLevel := slog.LevelInfo
	if debugEnv == "true" {
		logLevel = slog.LevelDebug
	}

	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})
	logger := slog.New(jsonHandler)

	slog.SetDefault(logger)
}

func init() {
	logSetup()
}

func main() {
	var (
		raftServers    string
		currentServer  string
		persistentPath string
		httpPort       string
		timeoutMin     string
		timeoutMax     string
		heartbeat      string
	)
	flag.StringVar(&raftServers, "servers", "", "Comma-separated list of Raft server addresses, e.g. localhost:8080,localhost:8081")
	flag.StringVar(&currentServer, "current", "", "Address of the current server")
	flag.StringVar(&persistentPath, "persistent-path", "", "Path to store persistent state")
	flag.StringVar(&httpPort, "http-port", "", "Port for HTTP server")
	flag.StringVar(&timeoutMin, "timeout-min", "", "Minimum timeout for election")
	flag.StringVar(&timeoutMax, "timeout-max", "", "Maximum timeout for election")
	flag.StringVar(&heartbeat, "heartbeat", "", "Heartbeat interval")
	flag.Parse()

	// If not provided try to get from environment variables
	if raftServers == "" {
		raftServers = os.Getenv("RAFT_SERVERS")
	}
	if currentServer == "" {
		currentServer = os.Getenv("CURRENT_SERVER")
	}
	if persistentPath == "" {
		persistentPath = os.Getenv("PERSISTENT_FILE_PATH")
	}
	if httpPort == "" {
		httpPort = os.Getenv("HTTP_PORT")
	}
	if timeoutMin == "" {
		timeoutMin = os.Getenv("TIMEOUT_MIN")
	}
	if timeoutMax == "" {
		timeoutMax = os.Getenv("TIMEOUT_MAX")
	}
	if heartbeat == "" {
		heartbeat = os.Getenv("HEARTBEAT")
	}
	slog.Info(httpPort)

	if raftServers == "" || currentServer == "" || persistentPath == "" || httpPort == "" || timeoutMin == "" || timeoutMax == "" || heartbeat == "" {
		missingFlags := []string{}
		if raftServers == "" {
			missingFlags = append(missingFlags, "servers")
		}
		if currentServer == "" {
			missingFlags = append(missingFlags, "current")
		}
		if persistentPath == "" {
			missingFlags = append(missingFlags, "persistent-path")
		}
		if httpPort == "" {
			missingFlags = append(missingFlags, "http-port")
		}
		if timeoutMin == "" {
			missingFlags = append(missingFlags, "timeout-min")
		}
		if timeoutMax == "" {
			missingFlags = append(missingFlags, "timeout-max")
		}
		if heartbeat == "" {
			missingFlags = append(missingFlags, "heartbeat")
		}
		slog.Error("Missing required flags", "flags", missingFlags)
		os.Exit(1)
	}

	os.Setenv("RAFT_SERVERS", raftServers)
	os.Setenv("CURRENT_SERVER", currentServer)
	os.Setenv("PERSISTENT_FILE_PATH", persistentPath)
	os.Setenv("HTTP_PORT", httpPort)
	os.Setenv("TIMEOUT_MIN", timeoutMin)
	os.Setenv("TIMEOUT_MAX", timeoutMax)
	os.Setenv("HEARTBEAT", heartbeat)

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
