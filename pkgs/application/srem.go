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

// Srem handles the SREM command to remove one or more members from the set stored at key.
// Returns the number of members that were removed from the set (not including non-existing members).
func Srem(eventManager *events.EventManager, sremCommandEvent *events.SremCommandEvent) {
	slog.Debug("[APPLICATION] Received srem command", "command", sremCommandEvent.Payload)

	err := validateSremCommand(sremCommandEvent)
	if err != nil {
		slog.Error("[APPLICATION][SREM] Failed to validate command", "error", err)
		sremCommandEvent.Reply <- &dto.SremCommandResponse{Removed: 0, Error: err.Error()}
		return
	}

	uuid := uuid.New().String()
	ch := make(chan bool)

	appendLogEntryEvent := events.AppendLogEntryEvent{
		Command: &dto.Command{
			Operation: consts.SremOp.String(),
			Args: &dto.Command_SremCommand{
				SremCommand: &dto.SremCommand{
					Key:     sremCommandEvent.Payload.Key,
					Members: sremCommandEvent.Payload.Members,
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

			// Get current set
			var currentSet []string
			if existingValue, exists := hashMap.Load(sremCommandEvent.Payload.Key); exists {
				if existingStr, ok := existingValue.(string); ok {
					json.Unmarshal([]byte(existingStr), &currentSet)
				}
			}

			// Convert to map for efficient lookups
			setMap := make(map[string]bool)
			for _, member := range currentSet {
				setMap[member] = true
			}

			// Remove members and count how many were actually removed
			removedCount := int32(0)
			for _, member := range sremCommandEvent.Payload.Members {
				if setMap[member] {
					delete(setMap, member)
					removedCount++
				}
			}

			// Convert back to slice
			newSet := make([]string, 0, len(setMap))
			for member := range setMap {
				newSet = append(newSet, member)
			}

			// Store updated set as JSON
			jsonData, _ := json.Marshal(newSet)
			hashMap.Store(sremCommandEvent.Payload.Key, string(jsonData))

			sremCommandEvent.Reply <- &dto.SremCommandResponse{Removed: removedCount, Error: ""}
		case <-time.After(TIMEOUT):
			slog.Error("[APPLICATION] No response from append log entry", "uuid", uuid)
			sremCommandEvent.Reply <- &dto.SremCommandResponse{Removed: 0, Error: "timeout"}
		}
	}()
}

func replicateSremCommand(sremCommand *dto.SremCommand) {
	// Get current set
	var currentSet []string
	if existingValue, exists := hashMap.Load(sremCommand.Key); exists {
		if existingStr, ok := existingValue.(string); ok {
			json.Unmarshal([]byte(existingStr), &currentSet)
		}
	}

	// Convert to map for efficient lookups
	setMap := make(map[string]bool)
	for _, member := range currentSet {
		setMap[member] = true
	}

	// Remove members
	for _, member := range sremCommand.Members {
		delete(setMap, member)
	}

	// Convert back to slice
	newSet := make([]string, 0, len(setMap))
	for member := range setMap {
		newSet = append(newSet, member)
	}

	// Store updated set as JSON
	jsonData, _ := json.Marshal(newSet)
	hashMap.Store(sremCommand.Key, string(jsonData))
}

func validateSremCommand(sremCommand *events.SremCommandEvent) error {
	if sremCommand.Payload.Key == "" {
		return errors.New("key is required")
	}

	if len(sremCommand.Payload.Members) == 0 {
		return errors.New("at least one member is required")
	}

	return nil
}
