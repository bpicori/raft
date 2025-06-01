package commands

import (
	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/tcp"
	"fmt"
	"log/slog"
)

// SaddCommand executes a SADD command against the Raft cluster.
// It adds one or more members to a set stored at key.
// Returns the number of elements that were added to the set (not including elements already present).
// Output format: (integer) %d\n
func SaddCommand(cfg *config.Config, key string, members []string) {
	leader := FindLeader(cfg)
	if leader == "" {
		slog.Error("No leader found")
		fmt.Println("Error: No leader found in the cluster. Try again later.")
		return
	}

	saddCommand := &dto.RaftRPC{
		Type: consts.SaddCommand.String(),
		Args: &dto.RaftRPC_SaddCommandRequest{
			SaddCommandRequest: &dto.SaddCommandRequest{
				Key:     key,
				Members: members,
			},
		},
	}

	resp, err := tcp.SendSyncRPC(leader, saddCommand)

	if err != nil {
		slog.Error("Error sending sadd command", "error", err)
		fmt.Printf("Error: Failed to sadd to key '%s': %v\n", key, err)
		return
	}

	if saddResponse := resp.GetSaddCommandResponse(); saddResponse != nil {
		if saddResponse.Error != "" {
			fmt.Printf("Error: %s\n", saddResponse.Error)
		} else {
			fmt.Printf("(integer) %d\n", saddResponse.Added)
		}
	} else {
		fmt.Println("Error: Invalid response from server")
	}
}
