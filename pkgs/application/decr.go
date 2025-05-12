package application

import (
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/events"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
)

var (
	ErrKeyRequired = errors.New("key is required")
	ErrValueInt    = errors.New("value is not an integer")
)

func Decr(eventManager *events.EventManager, decrCommandEvent *events.DecrCommandEvent) {
	slog.Debug("[APPLICATION][DECR] Received command", "command", decrCommandEvent.Payload)

	err := validateDecrCommand(decrCommandEvent)
	if err != nil {
		slog.Error("[APPLICATION][DECR] Failed to validate command", "error", err)
		decrCommandEvent.Reply <- &dto.DecrCommandResponse{Value: 0, Error: err.Error()}
		return
	}

	uuid := uuid.New().String()
	ch := make(chan bool)

	appendLogEntryEvent := events.AppendLogEntryEvent{
		Command: &dto.Command{
			Operation: consts.DecrementOp.String(),
			Args: &dto.Command_DecrCommand{
				DecrCommand: &dto.DecrCommand{
					Key: decrCommandEvent.Payload.Key,
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
			slog.Debug("[APPLICATION][DECR] Received response from append log entry", "uuid", uuid)
			currentValueInt, err := loadAndConvertToInt32(decrCommandEvent.Payload.Key)
			if err != nil {
				slog.Error("[APPLICATION][DECR] Failed to load value to reply", "key", decrCommandEvent.Payload.Key, "error", err)
				decrCommandEvent.Reply <- &dto.DecrCommandResponse{Value: 0}
				return
			}

			decrementedValue := currentValueInt - 1
			hashMap.Store(decrCommandEvent.Payload.Key, strconv.Itoa(int(decrementedValue)))

			decrCommandEvent.Reply <- &dto.DecrCommandResponse{Value: decrementedValue}
		case <-time.After(TIMEOUT):
			slog.Error("[APPLICATION][DECR] No response from append log entry", "uuid", uuid)
			decrCommandEvent.Reply <- &dto.DecrCommandResponse{Value: 0}
		}
	}()
}

func replicateDecrCommand(decrCommand *dto.DecrCommand) {
	currentValueInt, err := loadAndConvertToInt32(decrCommand.Key)
	if err != nil {
		slog.Error("[APPLICATION][DECR] Failed to load value when replicating command", "key", decrCommand.Key, "error", err)
		return
	}

	decrementedValue := currentValueInt - 1
	hashMap.Store(decrCommand.Key, strconv.Itoa(int(decrementedValue)))
}

func validateDecrCommand(decrCommand *events.DecrCommandEvent) error {
	// validate key
	if decrCommand.Payload.Key == "" {
		return ErrKeyRequired
	}

	// validate value
	_, err := loadAndConvertToInt32(decrCommand.Payload.Key)
	if err != nil {
		return ErrValueInt
	}

	return nil
}
