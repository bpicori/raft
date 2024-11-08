package main

import (
	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/logger"
	"log/slog"
	"os"
)

func init() {
	logger.LogSetup()
}

func main() {
	_, err := config.LoadConfig(true)
	if err != nil {
		slog.Error("Error loading config", "error", err)
		os.Exit(1)
	}

}
