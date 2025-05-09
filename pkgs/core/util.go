package core

import (
	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/dto"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"math/rand"
	"net"
	"os"
	"time"

	"google.golang.org/protobuf/proto"
)

var (
	WarningLogger *log.Logger
	InfoLogger    *log.Logger
	ErrorLogger   *log.Logger
)

// randomTimeout returns a random number between 150ms and 300ms.
func randomTimeout(from int, to int) time.Duration {
	return time.Duration(rand.Intn(to-from)+from) * time.Millisecond
}

// loadPersistedState loads the persisted state from the file system.
func loadPersistedState(config config.Config) (currentTerm int32, votedFor string, logEntry []*dto.LogEntry, commitLength int32) {
	fileName := fmt.Sprintf("%s.json", config.SelfID)
	filePath := fmt.Sprintf("%s/%s", config.PersistentFilePath, fileName)

	file, err := os.Open(filePath)
	if err != nil {
		// this is the first time the server is starting
		logEntry = make([]*dto.LogEntry, 0)
		return 0, "", logEntry, 0
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var state ServerState
	err = decoder.Decode(&state)
	if err != nil {
		return 0, "", nil, 0
	}

	if state.LogEntry == nil {
		state.LogEntry = make([]*dto.LogEntry, 0)
	}

	return state.CurrentTerm, state.VotedFor, state.LogEntry, state.CommitLength
}

// sendProtobufMessage sends a protobuf message to a connection.
func sendProtobufMessage(conn net.Conn, message proto.Message) error {
	data, err := proto.Marshal(message)
	if err != nil {
		slog.Debug("Error marshaling message", "error", err)
		return fmt.Errorf("error marshaling message: %v", err)
	}

	_, err = conn.Write(data)
	if err != nil {
		slog.Debug("Error sending data", "error", err)
		return fmt.Errorf("error sending data: %v", err)
	}

	return nil
}

