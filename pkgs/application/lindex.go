package application

import (
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/events"
	"encoding/json"
	"errors"
	"log/slog"
)

// Lindex handles the LINDEX command to return the element at the specified index in a list.
// This is a read-only operation that doesn't require Raft consensus.
// Index can be negative (Redis-style: -1 is last element, -2 is second-to-last, etc.)
// Returns "(nil)" if key doesn't exist, list is empty, or index is out of bounds.
func Lindex(eventManager *events.EventManager, lindexCommandEvent *events.LindexCommandEvent) {
	slog.Debug("[APPLICATION] Received lindex command", "command", lindexCommandEvent.Payload)

	err := validateLindexCommand(lindexCommandEvent)
	if err != nil {
		slog.Error("[APPLICATION][LINDEX] Failed to validate command", "error", err)
		lindexCommandEvent.Reply <- &dto.LindexCommandResponse{Element: "", Error: err.Error()}
		return
	}

	// Get the list from storage
	var currentList []string
	if existingValue, exists := hashMap.Load(lindexCommandEvent.Payload.Key); exists {
		if existingStr, ok := existingValue.(string); ok {
			if err := json.Unmarshal([]byte(existingStr), &currentList); err != nil {
				slog.Error("[APPLICATION][LINDEX] Failed to unmarshal list", "error", err)
				lindexCommandEvent.Reply <- &dto.LindexCommandResponse{Element: "", Error: "internal error"}
				return
			}
		}
	}

	// If list is empty or doesn't exist, return "(nil)"
	if len(currentList) == 0 {
		lindexCommandEvent.Reply <- &dto.LindexCommandResponse{Element: "(nil)", Error: ""}
		return
	}

	// Convert index to positive index if negative
	index := lindexCommandEvent.Payload.Index
	if index < 0 {
		index = int32(len(currentList)) + index
	}

	// Check if index is out of bounds
	if index < 0 || index >= int32(len(currentList)) {
		lindexCommandEvent.Reply <- &dto.LindexCommandResponse{Element: "(nil)", Error: ""}
		return
	}

	// Return the element at the specified index
	element := currentList[index]
	lindexCommandEvent.Reply <- &dto.LindexCommandResponse{Element: element, Error: ""}
}

// validateLindexCommand validates the input parameters for an LINDEX command.
// Returns an error if the key is empty or missing.
func validateLindexCommand(lindexCommand *events.LindexCommandEvent) error {
	if lindexCommand.Payload.Key == "" {
		return errors.New("key is required")
	}

	return nil
}
