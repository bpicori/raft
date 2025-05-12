package application

import (
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/events"
	"context"
	"log/slog"
	"sync"
)

// global hashmap to store the key-value pairs
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
			go Set(eventManager, &setCommandEvent)
		case getCommandEvent := <-eventManager.GetCommandRequestChan:
			go Get(eventManager, &getCommandEvent)
		case incrCommandEvent := <-eventManager.IncrCommandRequestChan:
			go Incr(eventManager, &incrCommandEvent)
		case decrCommandEvent := <-eventManager.DecrCommandRequestChan:
			go Decr(eventManager, &decrCommandEvent)
		case removeCommandEvent := <-eventManager.RemoveCommandRequestChan:
			go Remove(eventManager, &removeCommandEvent)
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
		replicateSetCommand(command.GetSetCommand())
	case dto.CommandOperation_INCREMENT:
		replicateIncrCommand(command.GetIncrCommand())
	case dto.CommandOperation_DECREMENT:
		replicateDecrCommand(command.GetDecrCommand())
	case dto.CommandOperation_DELETE:
		replicateRemoveCommand(command.GetRemoveCommand())
	}
}
