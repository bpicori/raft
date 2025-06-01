package commands

import (
	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/tcp"
	"fmt"
	"log/slog"
)

// SinterCommand executes a SINTER command against the Raft cluster.
// It returns the members of the set resulting from the intersection of all given sets.
// Returns an array of intersecting members.
// Output format: Each member as "(string) %s\n"
func SinterCommand(cfg *config.Config, keys []string) {
	leader := FindLeader(cfg)
	if leader == "" {
		slog.Error("No leader found")
		fmt.Println("Error: No leader found in the cluster. Try again later.")
		return
	}

	sinterCommand := &dto.RaftRPC{
		Type: consts.SinterCommand.String(),
		Args: &dto.RaftRPC_SinterCommandRequest{
			SinterCommandRequest: &dto.SinterCommandRequest{
				Keys: keys,
			},
		},
	}

	resp, err := tcp.SendSyncRPC(leader, sinterCommand)

	if err != nil {
		slog.Error("Error sending sinter command", "error", err)
		fmt.Printf("Error: Failed to sinter keys: %v\n", err)
		return
	}

	if sinterResponse := resp.GetSinterCommandResponse(); sinterResponse != nil {
		if sinterResponse.Error != "" {
			fmt.Printf("Error: %s\n", sinterResponse.Error)
		} else {
			// Display each member in the standardized format: (string) <member>
			for _, member := range sinterResponse.Members {
				fmt.Printf("(string) %s\n", member)
			}
			// If no members exist, no output is produced (empty response)
		}
	} else {
		fmt.Println("Error: Invalid response from server")
	}
}
