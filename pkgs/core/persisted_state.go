package core

import (
	"bpicori/raft/pkgs/dto"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
)


type ServerState struct {
	CurrentTerm  int32           `json:"currentTerm"`
	VotedFor     string          `json:"votedFor"`
	LogEntry     []*dto.LogEntry `json:"logEntry"`
	CommitLength int32           `json:"commitLength"`
}

func (s *ServerState) SaveToFile(serverId string, path string) error {
	fileName := fmt.Sprintf("%s.json", serverId)
	filePath := fmt.Sprintf("%s/%s", path, fileName)

	slog.Info("Saving state to file", "path", filePath)

	// Check if file already exists and overwrite
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	return encoder.Encode(s)
}