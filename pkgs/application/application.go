package application

import (
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/events"
	"context"
	"log/slog"
	"sync"
	"time"
)

// Timeout for waiting for append log entry response
var TIMEOUT = 5 * time.Second

// global hashmap to store the key-value pairs
var hashMap = sync.Map{}

type ApplicationParam struct {
	EventManager *events.EventManager
	Context      context.Context
	LogEntry     []*dto.LogEntry
	CommitLength int32
}

func Start(param *ApplicationParam) {
	eventManager := param.EventManager
	ctx := param.Context

	logEntry := param.LogEntry
	commitLength := param.CommitLength

	replicateLogEntries(logEntry, commitLength)

	slog.Info("[APPLICATION] Starting application")

	for {
		select {
		case <-ctx.Done():
			slog.Info("[APPLICATION] Context done, shutting down application")
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
		case lpushCommandEvent := <-eventManager.LpushCommandRequestChan:
			go Lpush(eventManager, &lpushCommandEvent)
		case lpopCommandEvent := <-eventManager.LpopCommandRequestChan:
			go Lpop(eventManager, &lpopCommandEvent)
		case lindexCommandEvent := <-eventManager.LindexCommandRequestChan:
			go Lindex(eventManager, &lindexCommandEvent)
		case llenCommandEvent := <-eventManager.LlenCommandRequestChan:
			go Llen(eventManager, &llenCommandEvent)
		case keysCommandEvent := <-eventManager.KeysCommandRequestChan:
			go Keys(eventManager, &keysCommandEvent)
		case saddCommandEvent := <-eventManager.SaddCommandRequestChan:
			go Sadd(eventManager, &saddCommandEvent)
		case sremCommandEvent := <-eventManager.SremCommandRequestChan:
			go Srem(eventManager, &sremCommandEvent)
		case sismemberCommandEvent := <-eventManager.SismemberCommandRequestChan:
			go Sismember(eventManager, &sismemberCommandEvent)
		case sinterCommandEvent := <-eventManager.SinterCommandRequestChan:
			go Sinter(eventManager, &sinterCommandEvent)
		case scardCommandEvent := <-eventManager.ScardCommandRequestChan:
			go Scard(eventManager, &scardCommandEvent)
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
	operation := consts.MapStringToCommandType(command.Operation)

	switch operation {
	case consts.SetOp:
		replicateSetCommand(command.GetSetCommand())
	case consts.IncrementOp:
		replicateIncrCommand(command.GetIncrCommand())
	case consts.DecrementOp:
		replicateDecrCommand(command.GetDecrCommand())
	case consts.DeleteOp:
		replicateRemoveCommand(command.GetRemoveCommand())
	case consts.LpushOp:
		replicateLpushCommand(command.GetLpushCommand())
	case consts.LpopOp:
		replicateLpopCommand(command.GetLpopCommand())
	case consts.LlenOp:
		// LLEN is a read-only operation, no replication needed
		slog.Debug("[APPLICATION] LLEN operation does not require replication")
	case consts.SaddOp:
		replicateSaddCommand(command.GetSaddCommand())
	case consts.SremOp:
		replicateSremCommand(command.GetSremCommand())
	}
}
