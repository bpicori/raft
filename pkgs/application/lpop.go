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

// Lpop handles the LPOP command to remove and return the leftmost element from a list.
// If the list is empty or doesn't exist, returns an empty element with no error.
// This function coordinates with the Raft consensus algorithm to ensure the operation
// is replicated across all cluster nodes before returning the result.
func Lpop(eventManager *events.EventManager, lpopCommandEvent *events.LpopCommandEvent) {
	slog.Debug("[APPLICATION] Received lpop command", "command", lpopCommandEvent.Payload)

	err := validateLpopCommand(lpopCommandEvent)
	if err != nil {
		slog.Error("[APPLICATION][LPOP] Failed to validate command", "error", err)
		lpopCommandEvent.Reply <- &dto.LpopCommandResponse{Element: "", Error: err.Error()}
		return
	}

	uuid := uuid.New().String()
	ch := make(chan bool)

	appendLogEntryEvent := events.AppendLogEntryEvent{
		Command: &dto.Command{
			Operation: consts.LpopOp.String(),
			Args: &dto.Command_LpopCommand{
				LpopCommand: &dto.LpopCommand{
					Key: lpopCommandEvent.Payload.Key,
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

			// Get current list
			var currentList []string
			element := ""

			if existingValue, exists := hashMap.Load(lpopCommandEvent.Payload.Key); exists {
				if existingStr, ok := existingValue.(string); ok {
					json.Unmarshal([]byte(existingStr), &currentList)
				}
			}

			// If list is empty or key doesn't exist, return empty element with no error
			if len(currentList) == 0 {
				lpopCommandEvent.Reply <- &dto.LpopCommandResponse{Element: "", Error: ""}
				return
			}

			// Remove element from left side (index 0)
			element = currentList[0]
			newList := currentList[1:]

			// Store updated list as JSON, or remove key if list becomes empty
			if len(newList) == 0 {
				hashMap.Delete(lpopCommandEvent.Payload.Key)
			} else {
				jsonData, _ := json.Marshal(newList)
				hashMap.Store(lpopCommandEvent.Payload.Key, string(jsonData))
			}

			lpopCommandEvent.Reply <- &dto.LpopCommandResponse{Element: element, Error: ""}
		case <-time.After(TIMEOUT):
			slog.Error("[APPLICATION] No response from append log entry", "uuid", uuid)
			lpopCommandEvent.Reply <- &dto.LpopCommandResponse{Element: "", Error: "timeout"}
		}
	}()
}

// replicateLpopCommand applies the LPOP operation to the local state during log replication.
// This function is called when a committed LPOP command needs to be applied to the state machine.
func replicateLpopCommand(lpopCommand *dto.LpopCommand) {
	// Get current list
	var currentList []string

	if existingValue, exists := hashMap.Load(lpopCommand.Key); exists {
		if existingStr, ok := existingValue.(string); ok {
			json.Unmarshal([]byte(existingStr), &currentList)
		}
	}

	// If list is empty or key doesn't exist, do nothing
	if len(currentList) == 0 {
		return
	}

	// Remove element from left side (index 0)
	newList := currentList[1:]

	// Store updated list as JSON, or remove key if list becomes empty
	if len(newList) == 0 {
		hashMap.Delete(lpopCommand.Key)
	} else {
		jsonData, _ := json.Marshal(newList)
		hashMap.Store(lpopCommand.Key, string(jsonData))
	}
}

// validateLpopCommand validates the input parameters for an LPOP command.
// Returns an error if the key is empty or missing.
func validateLpopCommand(lpopCommand *events.LpopCommandEvent) error {
	if lpopCommand.Payload.Key == "" {
		return errors.New("key is required")
	}

	return nil
}
