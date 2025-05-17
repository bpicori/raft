package application

import (
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/events"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

func Remove(eventManager *events.EventManager, removeCommandEvent *events.RemoveCommandEvent) {
	slog.Debug("[APPLICATION][REMOVE] Received command", "command", removeCommandEvent.Payload)

	err := validateRemoveCommand(removeCommandEvent)
	if err != nil {
		slog.Error("[APPLICATION][REMOVE] Failed to validate command", "error", err)
		removeCommandEvent.Reply <- &dto.GenericResponse{Ok: false}
		return
	}

	uuid := uuid.New().String()
	ch := make(chan bool)

	appendLogEntryEvent := events.AppendLogEntryEvent{
		Command: &dto.Command{
			Operation: consts.DeleteOp.String(),
			Args: &dto.Command_RemoveCommand{
				RemoveCommand: &dto.RemoveCommand{
					Key: removeCommandEvent.Payload.Key,
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
			slog.Debug("[APPLICATION][REMOVE] Received response from append log entry", "uuid", uuid)
			removeCommandEvent.Reply <- &dto.GenericResponse{Ok: true}
		case <-time.After(TIMEOUT):
			slog.Error("[APPLICATION][REMOVE] No response from append log entry", "uuid", uuid)
			removeCommandEvent.Reply <- &dto.GenericResponse{Ok: false}
		}
	}()
}

func validateRemoveCommand(removeCommandEvent *events.RemoveCommandEvent) error {
	if removeCommandEvent.Payload.Key == "" {
		return errors.New("key is required")
	}
	return nil
}

func replicateRemoveCommand(removeCommand *dto.RemoveCommand) {
	hashMap.Delete(removeCommand.Key)
}
