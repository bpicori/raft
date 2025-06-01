package commands

import (
	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/tcp"
	"fmt"
	"log/slog"
)

// ScardCommand executes a SCARD command against the Raft cluster.
// It returns the set cardinality (number of elements) of the set stored at key.
// Returns 0 if the key does not exist.
// Output format: (integer) %d\n
func ScardCommand(cfg *config.Config, key string) {
	leader := FindLeader(cfg)
	if leader == "" {
		slog.Error("No leader found")
		fmt.Println("Error: No leader found in the cluster. Try again later.")
		return
	}

	scardCommand := &dto.RaftRPC{
		Type: consts.ScardCommand.String(),
		Args: &dto.RaftRPC_ScardCommandRequest{
			ScardCommandRequest: &dto.ScardCommandRequest{
				Key: key,
			},
		},
	}

	resp, err := tcp.SendSyncRPC(leader, scardCommand)

	if err != nil {
		slog.Error("Error sending scard command", "error", err)
		fmt.Printf("Error: Failed to scard key '%s': %v\n", key, err)
		return
	}

	if scardResponse := resp.GetScardCommandResponse(); scardResponse != nil {
		if scardResponse.Error != "" {
			fmt.Printf("Error: %s\n", scardResponse.Error)
		} else {
			fmt.Printf("(integer) %d\n", scardResponse.Cardinality)
		}
	} else {
		fmt.Println("Error: Invalid response from server")
	}
}
