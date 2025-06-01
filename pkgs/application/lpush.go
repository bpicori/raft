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

func Lpush(eventManager *events.EventManager, lpushCommandEvent *events.LpushCommandEvent) {
	slog.Debug("[APPLICATION] Received lpush command", "command", lpushCommandEvent.Payload)

	err := validateLpushCommand(lpushCommandEvent)
	if err != nil {
		slog.Error("[APPLICATION][LPUSH] Failed to validate command", "error", err)
		lpushCommandEvent.Reply <- &dto.LpushCommandResponse{Length: 0, Error: err.Error()}
		return
	}

	uuid := uuid.New().String()
	ch := make(chan bool)

	appendLogEntryEvent := events.AppendLogEntryEvent{
		Command: &dto.Command{
			Operation: consts.LpushOp.String(),
			Args: &dto.Command_LpushCommand{
				LpushCommand: &dto.LpushCommand{
					Key:      lpushCommandEvent.Payload.Key,
					Elements: lpushCommandEvent.Payload.Elements,
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

			// Get current list or create new one
			var currentList []string
			if existingValue, exists := hashMap.Load(lpushCommandEvent.Payload.Key); exists {
				if existingStr, ok := existingValue.(string); ok {
					json.Unmarshal([]byte(existingStr), &currentList)
				}
			}

			// Prepend new elements (LPUSH semantics - elements are added left-to-right)
			newList := append(lpushCommandEvent.Payload.Elements, currentList...)

			// Store updated list as JSON
			jsonData, _ := json.Marshal(newList)
			hashMap.Store(lpushCommandEvent.Payload.Key, string(jsonData))

			lpushCommandEvent.Reply <- &dto.LpushCommandResponse{Length: int32(len(newList)), Error: ""}
		case <-time.After(currentTimeout):
			slog.Error("[APPLICATION] No response from append log entry", "uuid", uuid)
			lpushCommandEvent.Reply <- &dto.LpushCommandResponse{Length: 0, Error: "timeout"}
		}
	}()
}

func replicateLpushCommand(lpushCommand *dto.LpushCommand) {
	// Get current list or create new one
	var currentList []string
	if existingValue, exists := hashMap.Load(lpushCommand.Key); exists {
		if existingStr, ok := existingValue.(string); ok {
			json.Unmarshal([]byte(existingStr), &currentList)
		}
	}

	// Prepend new elements (LPUSH semantics - elements are added left-to-right)
	newList := append(lpushCommand.Elements, currentList...)

	// Store updated list as JSON
	jsonData, _ := json.Marshal(newList)
	hashMap.Store(lpushCommand.Key, string(jsonData))
}

func validateLpushCommand(lpushCommand *events.LpushCommandEvent) error {
	if lpushCommand.Payload.Key == "" {
		return errors.New("key is required")
	}

	if len(lpushCommand.Payload.Elements) == 0 {
		return errors.New("at least one element is required")
	}

	return nil
}
