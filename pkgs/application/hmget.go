package application

import (
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/events"
	"encoding/json"
	"errors"
	"log/slog"
)

// Hmget handles the HMGET command to return the values associated with the specified fields in the hash stored at key.
// Returns "(nil)" for fields that don't exist. The order of values corresponds to the order of fields requested.
// This is a read-only operation that doesn't require Raft consensus.
func Hmget(eventManager *events.EventManager, hmgetCommandEvent *events.HmgetCommandEvent) {
	slog.Debug("[APPLICATION] Received hmget command", "command", hmgetCommandEvent.Payload)

	err := validateHmgetCommand(hmgetCommandEvent)
	if err != nil {
		slog.Error("[APPLICATION][HMGET] Failed to validate command", "error", err)
		hmgetCommandEvent.Reply <- &dto.HmgetCommandResponse{Values: []string{}, Error: err.Error()}
		return
	}

	// Get the hash from storage
	var currentHash map[string]string
	if existingValue, exists := hashMap.Load(hmgetCommandEvent.Payload.Key); exists {
		if existingStr, ok := existingValue.(string); ok {
			if err := json.Unmarshal([]byte(existingStr), &currentHash); err != nil {
				// If it's not a valid JSON hash, it's not a hash type
				slog.Debug("[APPLICATION][HMGET] Key exists but is not a hash", "key", hmgetCommandEvent.Payload.Key)
				// Return "(nil)" for all requested fields
				values := make([]string, len(hmgetCommandEvent.Payload.Fields))
				for i := range values {
					values[i] = "(nil)"
				}
				hmgetCommandEvent.Reply <- &dto.HmgetCommandResponse{Values: values, Error: ""}
				return
			}
		}
	}

	// Prepare response values in the same order as requested fields
	values := make([]string, len(hmgetCommandEvent.Payload.Fields))
	for i, field := range hmgetCommandEvent.Payload.Fields {
		if currentHash != nil {
			if value, exists := currentHash[field]; exists {
				values[i] = value
			} else {
				values[i] = "(nil)"
			}
		} else {
			values[i] = "(nil)"
		}
	}

	slog.Debug("[APPLICATION][HMGET] Returning field values", "key", hmgetCommandEvent.Payload.Key, "fields", hmgetCommandEvent.Payload.Fields, "values", values)
	hmgetCommandEvent.Reply <- &dto.HmgetCommandResponse{Values: values, Error: ""}
}

func validateHmgetCommand(hmgetCommand *events.HmgetCommandEvent) error {
	if hmgetCommand.Payload.Key == "" {
		return errors.New("key is required")
	}

	if len(hmgetCommand.Payload.Fields) == 0 {
		return errors.New("at least one field is required")
	}

	// Validate that all field names are non-empty
	for _, field := range hmgetCommand.Payload.Fields {
		if field == "" {
			return errors.New("field names cannot be empty")
		}
	}

	return nil
}
