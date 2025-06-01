package application

import (
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/events"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// Sadd handles the SADD command to add one or more members to a set stored at key.
// Returns the number of elements that were added to the set (not including elements already present).
func Sadd(eventManager *events.EventManager, saddCommandEvent *events.SaddCommandEvent) {
	slog.Debug("[APPLICATION] Received sadd command", "command", saddCommandEvent.Payload)

	err := validateSaddCommand(saddCommandEvent)
	if err != nil {
		slog.Error("[APPLICATION][SADD] Failed to validate command", "error", err)
		saddCommandEvent.Reply <- &dto.SaddCommandResponse{Added: 0, Error: err.Error()}
		return
	}

	uuid := uuid.New().String()
	ch := make(chan bool)

	appendLogEntryEvent := events.AppendLogEntryEvent{
		Command: &dto.Command{
			Operation: consts.SaddOp.String(),
			Args: &dto.Command_SaddCommand{
				SaddCommand: &dto.SaddCommand{
					Key:     saddCommandEvent.Payload.Key,
					Members: saddCommandEvent.Payload.Members,
				},
			},
		},
		Uuid:  uuid,
		Reply: ch,
	}

	eventManager.AppendLogEntryChan <- appendLogEntryEvent

	go func() {
		select {
		case <-ch:
			slog.Debug("[APPLICATION] Received response from append log entry", "uuid", uuid)

			// Get current set or create new one
			var currentSet []string
			if existingValue, exists := hashMap.Load(saddCommandEvent.Payload.Key); exists {
				if existingStr, ok := existingValue.(string); ok {
					json.Unmarshal([]byte(existingStr), &currentSet)
				}
			}

			// Convert to map for efficient lookups
			setMap := make(map[string]bool)
			for _, member := range currentSet {
				setMap[member] = true
			}

			// Add new members and count how many were actually added
			addedCount := int32(0)
			for _, member := range saddCommandEvent.Payload.Members {
				if !setMap[member] {
					setMap[member] = true
					addedCount++
				}
			}

			// Convert back to slice
			newSet := make([]string, 0, len(setMap))
			for member := range setMap {
				newSet = append(newSet, member)
			}

			// Store updated set as JSON
			jsonData, _ := json.Marshal(newSet)
			hashMap.Store(saddCommandEvent.Payload.Key, string(jsonData))

			saddCommandEvent.Reply <- &dto.SaddCommandResponse{Added: addedCount, Error: ""}
		case <-time.After(TIMEOUT):
			slog.Error("[APPLICATION] No response from append log entry", "uuid", uuid)
			saddCommandEvent.Reply <- &dto.SaddCommandResponse{Added: 0, Error: "timeout"}
		}
	}()
}

func replicateSaddCommand(saddCommand *dto.SaddCommand) {
	// Get current set or create new one
	var currentSet []string
	if existingValue, exists := hashMap.Load(saddCommand.Key); exists {
		if existingStr, ok := existingValue.(string); ok {
			json.Unmarshal([]byte(existingStr), &currentSet)
		}
	}

	// Convert to map for efficient lookups
	setMap := make(map[string]bool)
	for _, member := range currentSet {
		setMap[member] = true
	}

	// Add new members
	for _, member := range saddCommand.Members {
		setMap[member] = true
	}

	// Convert back to slice
	newSet := make([]string, 0, len(setMap))
	for member := range setMap {
		newSet = append(newSet, member)
	}

	// Store updated set as JSON
	jsonData, _ := json.Marshal(newSet)
	hashMap.Store(saddCommand.Key, string(jsonData))
}

func validateSaddCommand(saddCommand *events.SaddCommandEvent) error {
	if saddCommand.Payload.Key == "" {
		return errors.New("key is required")
	}

	if len(saddCommand.Payload.Members) == 0 {
		return errors.New("at least one member is required")
	}

	return nil
}
