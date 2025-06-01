package application

import (
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/events"
	"log/slog"
)

// Keys handles the KEYS command to return all keys that currently exist in the key/value store.
// This is a read-only operation that doesn't require Raft consensus.
// Returns an empty list if no keys exist.
func Keys(eventManager *events.EventManager, keysCommandEvent *events.KeysCommandEvent) {
	slog.Debug("[APPLICATION] Received keys command", "command", keysCommandEvent.Payload)

	var keys []string

	// Iterate over all keys in the hashMap using Range
	hashMap.Range(func(key, value interface{}) bool {
		if keyStr, ok := key.(string); ok {
			keys = append(keys, keyStr)
		}
		return true // continue iteration
	})

	slog.Debug("[APPLICATION][KEYS] Returning keys", "count", len(keys), "keys", keys)
	keysCommandEvent.Reply <- &dto.KeysCommandResponse{Keys: keys, Error: ""}
}
