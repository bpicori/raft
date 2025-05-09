package core

import (
	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/dto"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"
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
