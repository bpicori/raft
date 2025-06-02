package application

import (
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/events"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// Hincrby handles the HINCRBY command to increment the integer value of field in the hash stored at key by increment.
// If the field doesn't exist, it is set to 0 before performing the operation.
// Returns the value after the increment operation.
// This is a write operation that requires Raft consensus.
func Hincrby(eventManager *events.EventManager, hincrbyCommandEvent *events.HincrbyCommandEvent) {
	slog.Debug("[APPLICATION] Received hincrby command", "command", hincrbyCommandEvent.Payload)

	err := validateHincrbyCommand(hincrbyCommandEvent)
	if err != nil {
		slog.Error("[APPLICATION][HINCRBY] Failed to validate command", "error", err)
		hincrbyCommandEvent.Reply <- &dto.HincrbyCommandResponse{Value: 0, Error: err.Error()}
		return
	}

	// Check if the field exists and is a valid integer
	currentValue, err := getHashFieldAsInt32(hincrbyCommandEvent.Payload.Key, hincrbyCommandEvent.Payload.Field)
	if err != nil {
		slog.Error("[APPLICATION][HINCRBY] Field value is not an integer", "error", err)
		hincrbyCommandEvent.Reply <- &dto.HincrbyCommandResponse{Value: 0, Error: "hash value is not an integer"}
		return
	}

	uuid := uuid.New().String()
	ch := make(chan bool)

	appendLogEntryEvent := events.AppendLogEntryEvent{
		Command: &dto.Command{
			Operation: consts.HincrbyOp.String(),
			Args: &dto.Command_HincrbyCommand{
				HincrbyCommand: &dto.HincrbyCommand{
					Key:       hincrbyCommandEvent.Payload.Key,
					Field:     hincrbyCommandEvent.Payload.Field,
					Increment: hincrbyCommandEvent.Payload.Increment,
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
			
			// Calculate new value and update the hash
			newValue := currentValue + hincrbyCommandEvent.Payload.Increment
			updateHashField(hincrbyCommandEvent.Payload.Key, hincrbyCommandEvent.Payload.Field, strconv.Itoa(int(newValue)))
			
			hincrbyCommandEvent.Reply <- &dto.HincrbyCommandResponse{Value: newValue, Error: ""}
		case <-time.After(currentTimeout):
			slog.Error("[APPLICATION][HINCRBY] No response from append log entry", "uuid", uuid)
			hincrbyCommandEvent.Reply <- &dto.HincrbyCommandResponse{Value: 0, Error: "timeout"}
		}
	}()
}

func replicateHincrbyCommand(hincrbyCommand *dto.HincrbyCommand) {
	// Get current value (defaults to 0 if field doesn't exist)
	currentValue, err := getHashFieldAsInt32(hincrbyCommand.Key, hincrbyCommand.Field)
	if err != nil {
		slog.Error("[APPLICATION][HINCRBY] Failed to get field value when replicating command", "key", hincrbyCommand.Key, "field", hincrbyCommand.Field, "error", err)
		return
	}

	// Calculate new value and update
	newValue := currentValue + hincrbyCommand.Increment
	updateHashField(hincrbyCommand.Key, hincrbyCommand.Field, strconv.Itoa(int(newValue)))
}

func validateHincrbyCommand(hincrbyCommand *events.HincrbyCommandEvent) error {
	if hincrbyCommand.Payload.Key == "" {
		return errors.New("key is required")
	}

	if hincrbyCommand.Payload.Field == "" {
		return errors.New("field is required")
	}

	return nil
}

// getHashFieldAsInt32 gets a hash field value and converts it to int32
// Returns 0 if the field doesn't exist or the hash doesn't exist
// Returns an error if the field exists but is not a valid integer
func getHashFieldAsInt32(key, field string) (int32, error) {
	// Get the hash from storage
	var currentHash map[string]string
	if existingValue, exists := hashMap.Load(key); exists {
		if existingStr, ok := existingValue.(string); ok {
			if err := json.Unmarshal([]byte(existingStr), &currentHash); err != nil {
				// If it's not a valid JSON hash, it's not a hash type - return 0
				return 0, nil
			}
		}
	}

	// If hash doesn't exist or field doesn't exist, return 0
	if currentHash == nil {
		return 0, nil
	}

	value, exists := currentHash[field]
	if !exists {
		return 0, nil
	}

	// Try to convert to integer
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}

	return int32(intValue), nil
}

// updateHashField updates a single field in a hash
func updateHashField(key, field, value string) {
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

	// Update field
	currentHash[field] = value

	// Store updated hash as JSON
	jsonData, _ := json.Marshal(currentHash)
	hashMap.Store(key, string(jsonData))
}
