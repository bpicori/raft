package commands

import (
	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/tcp"
	"fmt"
	"log/slog"
)

// LlenCommand executes an LLEN command against the Raft cluster.
// It returns the length of the list stored at the given key.
// If the key doesn't exist or is not a list, returns 0.
func LlenCommand(cfg *config.Config, key string) {
	leader := FindLeader(cfg)
	if leader == "" {
		slog.Error("No leader found")
		fmt.Println("Error: No leader found in the cluster. Try again later.")
		return
	}

	llenCommand := &dto.RaftRPC{
		Type: consts.LlenCommand.String(),
		Args: &dto.RaftRPC_LlenCommandRequest{
			LlenCommandRequest: &dto.LlenCommandRequest{
				Key: key,
			},
		},
	}

	resp, err := tcp.SendSyncRPC(leader, llenCommand)

	if err != nil {
		slog.Error("Error sending llen command", "error", err)
		fmt.Printf("Error: Failed to get length of list at key '%s': %v\n", key, err)
		return
	}

	if llenResponse := resp.GetLlenCommandResponse(); llenResponse != nil {
		if llenResponse.Error != "" {
			fmt.Printf("Error: %s\n", llenResponse.Error)
		} else {
			fmt.Printf("(integer) %d\n", llenResponse.Length)
		}
	} else {
		fmt.Println("Error: Invalid response from server")
	}
}
