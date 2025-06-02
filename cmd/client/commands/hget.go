package commands

import (
	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/tcp"
	"fmt"
	"log/slog"
)

// HgetCommand executes an HGET command against the Raft cluster.
// It returns the value associated with field in the hash stored at key.
// Returns "(nil)" if the field doesn't exist.
// Output format: (string) %s\n or (nil)
func HgetCommand(cfg *config.Config, key string, field string) {
	leader := FindLeader(cfg)
	if leader == "" {
		slog.Error("No leader found")
		fmt.Println("Error: No leader found in the cluster. Try again later.")
		return
	}

	hgetCommand := &dto.RaftRPC{
		Type: consts.HgetCommand.String(),
		Args: &dto.RaftRPC_HgetCommandRequest{
			HgetCommandRequest: &dto.HgetCommandRequest{
				Key:   key,
				Field: field,
			},
		},
	}

	resp, err := tcp.SendSyncRPC(leader, hgetCommand)

	if err != nil {
		slog.Error("Error sending hget command", "error", err)
		fmt.Printf("Error: Failed to hget field '%s' from key '%s': %v\n", field, key, err)
		return
	}

	hgetResponse := resp.GetHgetCommandResponse()
	if hgetResponse == nil {
		slog.Error("Error getting hget command response")
		fmt.Printf("Error: Failed to hget field '%s' from key '%s': invalid response\n", field, key)
		return
	}

	if hgetResponse.Error != "" {
		fmt.Printf("Error: %s\n", hgetResponse.Error)
		return
	}

	if hgetResponse.Value == "(nil)" {
		fmt.Println("(nil)")
	} else {
		fmt.Printf("(string) %s\n", hgetResponse.Value)
	}
}
