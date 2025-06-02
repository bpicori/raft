package commands

import (
	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/tcp"
	"fmt"
	"log/slog"
	"strconv"
)

// HincrbyCommand executes an HINCRBY command against the Raft cluster.
// It increments the integer value of field in the hash stored at key by increment.
// If the field doesn't exist, it is set to 0 before performing the operation.
// Returns the value after the increment operation.
// Output format: (integer) %d\n
func HincrbyCommand(cfg *config.Config, key string, field string, incrementStr string) {
	leader := FindLeader(cfg)
	if leader == "" {
		slog.Error("No leader found")
		fmt.Println("Error: No leader found in the cluster. Try again later.")
		return
	}

	// Parse increment value
	increment, err := strconv.Atoi(incrementStr)
	if err != nil {
		fmt.Printf("Error: Invalid increment value '%s'. Must be an integer.\n", incrementStr)
		return
	}

	hincrbyCommand := &dto.RaftRPC{
		Type: consts.HincrbyCommand.String(),
		Args: &dto.RaftRPC_HincrbyCommandRequest{
			HincrbyCommandRequest: &dto.HincrbyCommandRequest{
				Key:       key,
				Field:     field,
				Increment: int32(increment),
			},
		},
	}

	resp, err := tcp.SendSyncRPC(leader, hincrbyCommand)

	if err != nil {
		slog.Error("Error sending hincrby command", "error", err)
		fmt.Printf("Error: Failed to hincrby field '%s' in key '%s': %v\n", field, key, err)
		return
	}

	hincrbyResponse := resp.GetHincrbyCommandResponse()
	if hincrbyResponse == nil {
		slog.Error("Error getting hincrby command response")
		fmt.Printf("Error: Failed to hincrby field '%s' in key '%s': invalid response\n", field, key)
		return
	}

	if hincrbyResponse.Error != "" {
		fmt.Printf("Error: %s\n", hincrbyResponse.Error)
		return
	}

	fmt.Printf("(integer) %d\n", hincrbyResponse.Value)
}
