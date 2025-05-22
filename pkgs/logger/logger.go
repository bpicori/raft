package logger

import (
	"log/slog"
	"os"
)

func LogSetup() {
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
