package application

import (
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/events"
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

var hashMap = sync.Map{}

type ApplicationParam struct {
	EventManager *events.EventManager
	Context      context.Context
	WaitGroup    *sync.WaitGroup
	LogEntry     []*dto.LogEntry
	CommitLength int32
}

func Start(param *ApplicationParam) {
	eventManager := param.EventManager
	ctx := param.Context
	wg := param.WaitGroup

	logEntry := param.LogEntry
	commitLength := param.CommitLength

	replicateLogEntries(logEntry, commitLength)

	slog.Info("[APPLICATION] Starting application")

	for {
		select {
		case <-ctx.Done():
			slog.Info("[APPLICATION] Context done, shutting down application")
			wg.Done()
			return
		case setCommandEvent := <-eventManager.SetCommandRequestChan:
			slog.Debug("[APPLICATION] Received set command", "command", setCommandEvent.Payload)

			uuid := uuid.New().String()
			ch := make(chan bool)

			appendLogEntryEvent := events.AppendLogEntryEvent{
				Command: &dto.Command{
					Operation: dto.CommandOperation_SET,
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
					setCommandEvent.Reply <- &dto.OkResponse{Ok: true}
					hashMap.Store(setCommandEvent.Payload.Key, setCommandEvent.Payload.Value)
				case <-time.After(5 * time.Second):
					slog.Error("[APPLICATION] No response from append log entry", "uuid", uuid)
					setCommandEvent.Reply <- &dto.OkResponse{Ok: false}
				}
			}()
		case getCommandEvent := <-eventManager.GetCommandRequestChan:
			slog.Debug("[APPLICATION] Received get command", "command", getCommandEvent.Payload)

			value, ok := hashMap.Load(getCommandEvent.Payload.Key)
			if !ok {
				getCommandEvent.Reply <- &dto.GetCommandResponse{Value: ""}
				continue
			}

			getCommandEvent.Reply <- &dto.GetCommandResponse{Value: value.(string)}
		case syncCommandEvent := <-eventManager.SyncCommandRequestChan:
			slog.Debug("[APPLICATION] Received sync command", "command", syncCommandEvent.LogEntry)

			replicateLogEntry(syncCommandEvent.LogEntry)
		}
	}
}

func replicateLogEntries(logEntry []*dto.LogEntry, commitLength int32) {
	for i := range int(commitLength) {
		replicateLogEntry(logEntry[i])
	}
}

func replicateLogEntry(logEntry *dto.LogEntry) {
	command := logEntry.Command
	operation := command.Operation

	switch operation {
	case dto.CommandOperation_SET:
		args := command.GetSetCommand()
		hashMap.Store(args.Key, args.Value)
	}
}
