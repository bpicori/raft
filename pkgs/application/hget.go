package application

import (
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/events"
	"encoding/json"
	"errors"
	"log/slog"
)

// Hget handles the HGET command to return the value associated with field in the hash stored at key.
// Returns "(nil)" if the field doesn't exist or if the key doesn't exist.
// This is a read-only operation that doesn't require Raft consensus.
func Hget(eventManager *events.EventManager, hgetCommandEvent *events.HgetCommandEvent) {
	slog.Debug("[APPLICATION] Received hget command", "command", hgetCommandEvent.Payload)

	err := validateHgetCommand(hgetCommandEvent)
	if err != nil {
		slog.Error("[APPLICATION][HGET] Failed to validate command", "error", err)
		hgetCommandEvent.Reply <- &dto.HgetCommandResponse{Value: "", Error: err.Error()}
		return
	}

	// Get the hash from storage
	var currentHash map[string]string
	if existingValue, exists := hashMap.Load(hgetCommandEvent.Payload.Key); exists {
		if existingStr, ok := existingValue.(string); ok {
			if err := json.Unmarshal([]byte(existingStr), &currentHash); err != nil {
				// If it's not a valid JSON hash, it's not a hash type
				slog.Debug("[APPLICATION][HGET] Key exists but is not a hash", "key", hgetCommandEvent.Payload.Key)
				hgetCommandEvent.Reply <- &dto.HgetCommandResponse{Value: "(nil)", Error: ""}
				return
			}
		}
	}

	// If hash doesn't exist or field doesn't exist, return "(nil)"
	if currentHash == nil {
		hgetCommandEvent.Reply <- &dto.HgetCommandResponse{Value: "(nil)", Error: ""}
		return
	}

	value, exists := currentHash[hgetCommandEvent.Payload.Field]
	if !exists {
		hgetCommandEvent.Reply <- &dto.HgetCommandResponse{Value: "(nil)", Error: ""}
		return
	}

	slog.Debug("[APPLICATION][HGET] Returning field value", "key", hgetCommandEvent.Payload.Key, "field", hgetCommandEvent.Payload.Field, "value", value)
	hgetCommandEvent.Reply <- &dto.HgetCommandResponse{Value: value, Error: ""}
}

func validateHgetCommand(hgetCommand *events.HgetCommandEvent) error {
	if hgetCommand.Payload.Key == "" {
		return errors.New("key is required")
	}

	if hgetCommand.Payload.Field == "" {
		return errors.New("field is required")
	}

	return nil
}
