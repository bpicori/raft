package storage

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"bpicori/raft/pkgs/dto"
)

type StateMachineState struct {
	ServerId     string
	Path         string
	CurrentTerm  int32           `json:"currentTerm"`
	VotedFor     string          `json:"votedFor"`
	LogEntry     []*dto.LogEntry `json:"logEntry"`
	CommitLength int32           `json:"commitLength"`
}

func PersistStateMachine(state *StateMachineState) error {
	fileName := fmt.Sprintf("%s.json", state.ServerId)
	filePath := fmt.Sprintf("%s/%s", state.Path, fileName)

	slog.Info("Saving state to file", "path", filePath)

	// Check if file already exists and overwrite
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	return encoder.Encode(state)
}

func LoadStateMachine(serverId string, path string) (*StateMachineState, error) {
	fileName := fmt.Sprintf("%s.json", serverId)
	filePath := fmt.Sprintf("%s/%s", path, fileName)

	var state StateMachineState
	file, err := os.Open(filePath)
	if err != nil {
		// if file doesn't exist, first time running
		return &state, nil
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&state)
	if err != nil {
		return nil, err
	}

	if state.LogEntry == nil {
		state.LogEntry = make([]*dto.LogEntry, 0)
	}

	return &state, nil
}
