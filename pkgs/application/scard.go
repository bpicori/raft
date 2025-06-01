package application

import (
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/events"
	"encoding/json"
	"errors"
	"log/slog"
)

// Scard handles the SCARD command to return the set cardinality (number of elements) of the set stored at key.
// Returns 0 if the key does not exist.
// This is a read-only operation that doesn't require Raft consensus.
func Scard(eventManager *events.EventManager, scardCommandEvent *events.ScardCommandEvent) {
	slog.Debug("[APPLICATION] Received scard command", "command", scardCommandEvent.Payload)

	err := validateScardCommand(scardCommandEvent)
	if err != nil {
		slog.Error("[APPLICATION][SCARD] Failed to validate command", "error", err)
		scardCommandEvent.Reply <- &dto.ScardCommandResponse{Cardinality: 0, Error: err.Error()}
		return
	}

	// Get current set
	var currentSet []string
	if existingValue, exists := hashMap.Load(scardCommandEvent.Payload.Key); exists {
		if existingStr, ok := existingValue.(string); ok {
			json.Unmarshal([]byte(existingStr), &currentSet)
		}
	}

	cardinality := int32(len(currentSet))
	scardCommandEvent.Reply <- &dto.ScardCommandResponse{Cardinality: cardinality, Error: ""}
}

func validateScardCommand(scardCommand *events.ScardCommandEvent) error {
	if scardCommand.Payload.Key == "" {
		return errors.New("key is required")
	}

	return nil
}
