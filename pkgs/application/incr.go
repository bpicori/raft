package application

import (
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/events"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
)

func Incr(eventManager *events.EventManager, incrCommandEvent *events.IncrCommandEvent) {
	slog.Debug("[APPLICATION][INCR] Received command", "command", incrCommandEvent.Payload)

	err := validateIncrCommand(incrCommandEvent)
	if err != nil {
		slog.Error("[APPLICATION][INCR] Failed to validate command", "error", err)
		incrCommandEvent.Reply <- &dto.IncrCommandResponse{Value: 0, Error: err.Error()}
		return
	}

	uuid := uuid.New().String()
	ch := make(chan bool)

	appendLogEntryEvent := events.AppendLogEntryEvent{
		Command: &dto.Command{
			Operation: dto.CommandOperation_INCREMENT,
			Args: &dto.Command_IncrCommand{
				IncrCommand: &dto.IncrCommand{
					Key: incrCommandEvent.Payload.Key,
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
			slog.Debug("[APPLICATION][INCR] Received response from append log entry", "uuid", uuid)
			currentValueInt, err := loadAndConvertToInt32(incrCommandEvent.Payload.Key)
			if err != nil {
				slog.Error("[APPLICATION][INCR] Failed to load value to reply", "key", incrCommandEvent.Payload.Key, "error", err)
				incrCommandEvent.Reply <- &dto.IncrCommandResponse{Value: 0}
				return
			}

			incrementedValue := currentValueInt + 1
			hashMap.Store(incrCommandEvent.Payload.Key, strconv.Itoa(int(incrementedValue)))

			incrCommandEvent.Reply <- &dto.IncrCommandResponse{Value: incrementedValue}
		case <-time.After(TIMEOUT):
			slog.Error("[APPLICATION][INCR] No response from append log entry", "uuid", uuid)
			incrCommandEvent.Reply <- &dto.IncrCommandResponse{Value: 0}
		}
	}()
}

func replicateIncrCommand(incrCommand *dto.IncrCommand) {
	currentValueInt, err := loadAndConvertToInt32(incrCommand.Key)
	if err != nil {
		slog.Error("[APPLICATION][INCR] Failed to load value when replicating command", "key", incrCommand.Key, "error", err)
		return
	}

	incrementedValue := currentValueInt + 1
	hashMap.Store(incrCommand.Key, strconv.Itoa(int(incrementedValue)))
}

func validateIncrCommand(incrCommand *events.IncrCommandEvent) error {
	// validate key
	if incrCommand.Payload.Key == "" {
		return ErrKeyRequired
	}

	// validate value
	_, err := loadAndConvertToInt32(incrCommand.Payload.Key)
	if err != nil {
		return ErrValueInt
	}

	return nil
}
