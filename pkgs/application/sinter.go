package application

import (
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/events"
	"encoding/json"
	"errors"
	"log/slog"
)

// Sinter handles the SINTER command to return the members of the set resulting from the intersection of all given sets.
// Returns an array of intersecting members.
// This is a read-only operation that doesn't require Raft consensus.
func Sinter(eventManager *events.EventManager, sinterCommandEvent *events.SinterCommandEvent) {
	slog.Debug("[APPLICATION] Received sinter command", "command", sinterCommandEvent.Payload)

	err := validateSinterCommand(sinterCommandEvent)
	if err != nil {
		slog.Error("[APPLICATION][SINTER] Failed to validate command", "error", err)
		sinterCommandEvent.Reply <- &dto.SinterCommandResponse{Members: []string{}, Error: err.Error()}
		return
	}

	keys := sinterCommandEvent.Payload.Keys
	if len(keys) == 0 {
		sinterCommandEvent.Reply <- &dto.SinterCommandResponse{Members: []string{}, Error: ""}
		return
	}

	// Get the first set to start intersection
	var firstSet []string
	if existingValue, exists := hashMap.Load(keys[0]); exists {
		if existingStr, ok := existingValue.(string); ok {
			json.Unmarshal([]byte(existingStr), &firstSet)
		}
	}

	// If first set is empty or doesn't exist, intersection is empty
	if len(firstSet) == 0 {
		sinterCommandEvent.Reply <- &dto.SinterCommandResponse{Members: []string{}, Error: ""}
		return
	}

	// Convert first set to map for efficient lookups
	intersectionMap := make(map[string]bool)
	for _, member := range firstSet {
		intersectionMap[member] = true
	}

	// Intersect with each subsequent set
	for i := 1; i < len(keys); i++ {
		var currentSet []string
		if existingValue, exists := hashMap.Load(keys[i]); exists {
			if existingStr, ok := existingValue.(string); ok {
				json.Unmarshal([]byte(existingStr), &currentSet)
			}
		}

		// Create map for current set
		currentSetMap := make(map[string]bool)
		for _, member := range currentSet {
			currentSetMap[member] = true
		}

		// Keep only members that exist in both sets
		newIntersectionMap := make(map[string]bool)
		for member := range intersectionMap {
			if currentSetMap[member] {
				newIntersectionMap[member] = true
			}
		}
		intersectionMap = newIntersectionMap

		// If intersection becomes empty, no need to continue
		if len(intersectionMap) == 0 {
			break
		}
	}

	// Convert intersection map back to slice
	result := make([]string, 0, len(intersectionMap))
	for member := range intersectionMap {
		result = append(result, member)
	}

	sinterCommandEvent.Reply <- &dto.SinterCommandResponse{Members: result, Error: ""}
}

func validateSinterCommand(sinterCommand *events.SinterCommandEvent) error {
	if len(sinterCommand.Payload.Keys) == 0 {
		return errors.New("at least one key is required")
	}

	for _, key := range sinterCommand.Payload.Keys {
		if key == "" {
			return errors.New("all keys must be non-empty")
		}
	}

	return nil
}
