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

var TIMEOUT = 5 * time.Second

func Set(eventManager *events.EventManager, setCommandEvent *events.SetCommandEvent) {
	slog.Debug("[APPLICATION] Received set command", "command", setCommandEvent.Payload)

	err := validateSetCommand(setCommandEvent)
	if err != nil {
		slog.Error("[APPLICATION][SET] Failed to validate command", "error", err)
		setCommandEvent.Reply <- &dto.GenericResponse{Ok: false}
		return
	}

	uuid := uuid.New().String()
	ch := make(chan bool)

	appendLogEntryEvent := events.AppendLogEntryEvent{
		Command: &dto.Command{
			Operation: consts.SetOp.String(),
			Args: &dto.Command_SetCommand{
				SetCommand: &dto.SetCommand{
					Key:   setCommandEvent.Payload.Key,
					Value: setCommandEvent.Payload.Value,
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
			setCommandEvent.Reply <- &dto.GenericResponse{Ok: true}
			hashMap.Store(setCommandEvent.Payload.Key, setCommandEvent.Payload.Value)
		case <-time.After(TIMEOUT):
			slog.Error("[APPLICATION] No response from append log entry", "uuid", uuid)
			setCommandEvent.Reply <- &dto.GenericResponse{Ok: false}
		}
	}()
}

func replicateSetCommand(setCommand *dto.SetCommand) {
	hashMap.Store(setCommand.Key, setCommand.Value)
}

func validateSetCommand(setCommand *events.SetCommandEvent) error {
	if setCommand.Payload.Key == "" {
		return errors.New("key is required")
	}

	if setCommand.Payload.Value == "" {
		return errors.New("value is required")
	}

	return nil
}
