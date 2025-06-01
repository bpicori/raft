package application

import (
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/events"
	"encoding/json"
	"errors"
	"log/slog"
)

// Llen handles the LLEN command to return the length of a list stored at a key.
// This is a read-only operation that doesn't require Raft consensus.
// Returns 0 if key doesn't exist or if the key exists but is not a list.
func Llen(eventManager *events.EventManager, llenCommandEvent *events.LlenCommandEvent) {
	slog.Debug("[APPLICATION] Received llen command", "command", llenCommandEvent.Payload)

	err := validateLlenCommand(llenCommandEvent)
	if err != nil {
		slog.Error("[APPLICATION][LLEN] Failed to validate command", "error", err)
		llenCommandEvent.Reply <- &dto.LlenCommandResponse{Length: 0, Error: err.Error()}
		return
	}

	// Get the list from storage
	var length int32 = 0
	if existingValue, exists := hashMap.Load(llenCommandEvent.Payload.Key); exists {
		if existingStr, ok := existingValue.(string); ok {
			var currentList []string
			if err := json.Unmarshal([]byte(existingStr), &currentList); err != nil {
				// If it's not a valid JSON list, it's not a list type - return 0
				slog.Debug("[APPLICATION][LLEN] Key exists but is not a list", "key", llenCommandEvent.Payload.Key)
				llenCommandEvent.Reply <- &dto.LlenCommandResponse{Length: 0, Error: ""}
				return
			}
			length = int32(len(currentList))
		}
	}

	slog.Debug("[APPLICATION][LLEN] Returning list length", "key", llenCommandEvent.Payload.Key, "length", length)
	llenCommandEvent.Reply <- &dto.LlenCommandResponse{Length: length, Error: ""}
}

// validateLlenCommand validates the LLEN command payload
func validateLlenCommand(llenCommandEvent *events.LlenCommandEvent) error {
	if llenCommandEvent.Payload == nil {
		return errors.New("payload is nil")
	}

	if llenCommandEvent.Payload.Key == "" {
		return errors.New("key is required")
	}

	return nil
}
