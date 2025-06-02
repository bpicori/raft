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

// Hset handles the HSET command to set field-value pairs in a hash stored at key.
// Returns the number of fields that were added (not updated).
// This is a write operation that requires Raft consensus.
func Hset(eventManager *events.EventManager, hsetCommandEvent *events.HsetCommandEvent) {
	slog.Debug("[APPLICATION] Received hset command", "command", hsetCommandEvent.Payload)

	err := validateHsetCommand(hsetCommandEvent)
	if err != nil {
		slog.Error("[APPLICATION][HSET] Failed to validate command", "error", err)
		hsetCommandEvent.Reply <- &dto.HsetCommandResponse{Added: 0, Error: err.Error()}
		return
	}

	uuid := uuid.New().String()
	ch := make(chan bool)

	appendLogEntryEvent := events.AppendLogEntryEvent{
		Command: &dto.Command{
			Operation: consts.HsetOp.String(),
			Args: &dto.Command_HsetCommand{
				HsetCommand: &dto.HsetCommand{
					Key:    hsetCommandEvent.Payload.Key,
					Fields: hsetCommandEvent.Payload.Fields,
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
			
			// Calculate how many fields were added (not updated)
			added := calculateAddedFields(hsetCommandEvent.Payload.Key, hsetCommandEvent.Payload.Fields)
			
			// Update the hash in local storage
			updateHashFields(hsetCommandEvent.Payload.Key, hsetCommandEvent.Payload.Fields)
			
			hsetCommandEvent.Reply <- &dto.HsetCommandResponse{Added: added, Error: ""}
		case <-time.After(currentTimeout):
			slog.Error("[APPLICATION][HSET] No response from append log entry", "uuid", uuid)
			hsetCommandEvent.Reply <- &dto.HsetCommandResponse{Added: 0, Error: "timeout"}
		}
	}()
}

func replicateHsetCommand(hsetCommand *dto.HsetCommand) {
	updateHashFields(hsetCommand.Key, hsetCommand.Fields)
}

func validateHsetCommand(hsetCommand *events.HsetCommandEvent) error {
	if hsetCommand.Payload.Key == "" {
		return errors.New("key is required")
	}

	if len(hsetCommand.Payload.Fields) == 0 {
		return errors.New("at least one field-value pair is required")
	}

	// Validate that all field names are non-empty
	for field := range hsetCommand.Payload.Fields {
		if field == "" {
			return errors.New("field names cannot be empty")
		}
	}

	return nil
}

// calculateAddedFields returns the number of fields that were added (not updated)
func calculateAddedFields(key string, newFields map[string]string) int32 {
	// Get current hash or create new one
	var currentHash map[string]string
	if existingValue, exists := hashMap.Load(key); exists {
		if existingStr, ok := existingValue.(string); ok {
			json.Unmarshal([]byte(existingStr), &currentHash)
		}
	}
	
	if currentHash == nil {
		currentHash = make(map[string]string)
	}

	// Count how many fields are new
	added := int32(0)
	for field := range newFields {
		if _, exists := currentHash[field]; !exists {
			added++
		}
	}

	return added
}

// updateHashFields updates the hash with new field-value pairs
func updateHashFields(key string, newFields map[string]string) {
	// Get current hash or create new one
	var currentHash map[string]string
	if existingValue, exists := hashMap.Load(key); exists {
		if existingStr, ok := existingValue.(string); ok {
			json.Unmarshal([]byte(existingStr), &currentHash)
		}
	}
	
	if currentHash == nil {
		currentHash = make(map[string]string)
	}

	// Update fields
	for field, value := range newFields {
		currentHash[field] = value
	}

	// Store updated hash as JSON
	jsonData, _ := json.Marshal(currentHash)
	hashMap.Store(key, string(jsonData))
}
