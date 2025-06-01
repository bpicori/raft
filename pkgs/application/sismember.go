package application

import (
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/events"
	"encoding/json"
	"errors"
	"log/slog"
)

// Sismember handles the SISMEMBER command to test if member is a member of the set stored at key.
// Returns 1 if the element is a member of the set, 0 if not or if key does not exist.
// This is a read-only operation that doesn't require Raft consensus.
func Sismember(eventManager *events.EventManager, sismemberCommandEvent *events.SismemberCommandEvent) {
	slog.Debug("[APPLICATION] Received sismember command", "command", sismemberCommandEvent.Payload)

	err := validateSismemberCommand(sismemberCommandEvent)
	if err != nil {
		slog.Error("[APPLICATION][SISMEMBER] Failed to validate command", "error", err)
		sismemberCommandEvent.Reply <- &dto.SismemberCommandResponse{IsMember: 0, Error: err.Error()}
		return
	}

	// Get current set
	var currentSet []string
	if existingValue, exists := hashMap.Load(sismemberCommandEvent.Payload.Key); exists {
		if existingStr, ok := existingValue.(string); ok {
			json.Unmarshal([]byte(existingStr), &currentSet)
		}
	}

	// Check if member exists in set
	isMember := int32(0)
	for _, member := range currentSet {
		if member == sismemberCommandEvent.Payload.Member {
			isMember = 1
			break
		}
	}

	sismemberCommandEvent.Reply <- &dto.SismemberCommandResponse{IsMember: isMember, Error: ""}
}

func validateSismemberCommand(sismemberCommand *events.SismemberCommandEvent) error {
	if sismemberCommand.Payload.Key == "" {
		return errors.New("key is required")
	}

	if sismemberCommand.Payload.Member == "" {
		return errors.New("member is required")
	}

	return nil
}
