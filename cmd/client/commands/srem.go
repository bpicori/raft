package commands

import (
	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/tcp"
	"fmt"
	"log/slog"
)

// SremCommand executes a SREM command against the Raft cluster.
// It removes one or more members from the set stored at key.
// Returns the number of members that were removed from the set (not including non-existing members).
// Output format: (integer) %d\n
func SremCommand(cfg *config.Config, key string, members []string) {
	leader := FindLeader(cfg)
	if leader == "" {
		slog.Error("No leader found")
		fmt.Println("Error: No leader found in the cluster. Try again later.")
		return
	}

	sremCommand := &dto.RaftRPC{
		Type: consts.SremCommand.String(),
		Args: &dto.RaftRPC_SremCommandRequest{
			SremCommandRequest: &dto.SremCommandRequest{
				Key:     key,
				Members: members,
			},
		},
	}

	resp, err := tcp.SendSyncRPC(leader, sremCommand)

	if err != nil {
		slog.Error("Error sending srem command", "error", err)
		fmt.Printf("Error: Failed to srem from key '%s': %v\n", key, err)
		return
	}

	if sremResponse := resp.GetSremCommandResponse(); sremResponse != nil {
		if sremResponse.Error != "" {
			fmt.Printf("Error: %s\n", sremResponse.Error)
		} else {
			fmt.Printf("(integer) %d\n", sremResponse.Removed)
		}
	} else {
		fmt.Println("Error: Invalid response from server")
	}
}
