package commands

import (
	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/tcp"
	"fmt"
	"log/slog"
)

// LpopCommand executes an LPOP command against the Raft cluster.
// It removes and returns the leftmost element from the list stored at the given key.
// If the list is empty or the key doesn't exist, returns "(nil)".
func LpopCommand(cfg *config.Config, key string) {
	leader := FindLeader(cfg)
	if leader == "" {
		slog.Error("No leader found")
		fmt.Println("Error: No leader found in the cluster. Try again later.")
		return
	}

	lpopCommand := &dto.RaftRPC{
		Type: consts.LpopCommand.String(),
		Args: &dto.RaftRPC_LpopCommandRequest{
			LpopCommandRequest: &dto.LpopCommandRequest{
				Key: key,
			},
		},
	}

	resp, err := tcp.SendSyncRPC(leader, lpopCommand)

	if err != nil {
		slog.Error("Error sending lpop command", "error", err)
		fmt.Printf("Error: Failed to lpop from key '%s': %v\n", key, err)
		return
	}

	lpopResponse := resp.GetLpopCommandResponse()
	if lpopResponse == nil {
		slog.Error("Error getting lpop command response", "error", err)
		fmt.Printf("Error: Failed to lpop from key '%s': %v\n", key, err)
		return
	}

	if lpopResponse.Error != "" {
		fmt.Printf("Error: %s\n", lpopResponse.Error)
		return
	}

	if lpopResponse.Element == "" {
		fmt.Println("(nil)")
	} else {
		fmt.Printf("\"%s\"\n", lpopResponse.Element)
	}
}
