package commands

import (
	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/tcp"
	"fmt"
	"log/slog"
)

// KeysCommand executes a KEYS command against the Raft cluster.
// It returns all keys that currently exist in the key/value store.
// Each key is displayed as "(string) <key_name>\n".
// If no keys exist, returns an empty response.
func KeysCommand(cfg *config.Config) {
	leader := FindLeader(cfg)
	if leader == "" {
		slog.Error("No leader found")
		fmt.Println("Error: No leader found in the cluster. Try again later.")
		return
	}

	keysCommand := &dto.RaftRPC{
		Type: consts.KeysCommand.String(),
		Args: &dto.RaftRPC_KeysCommandRequest{
			KeysCommandRequest: &dto.KeysCommandRequest{},
		},
	}

	resp, err := tcp.SendSyncRPC(leader, keysCommand)

	if err != nil {
		slog.Error("Error sending keys command", "error", err)
		fmt.Printf("Error: Failed to get keys from leader: %v\n", err)
		return
	}

	if keysResponse := resp.GetKeysCommandResponse(); keysResponse != nil {
		if keysResponse.Error != "" {
			fmt.Printf("Error: %s\n", keysResponse.Error)
		} else {
			// Display each key in the standardized format: (string) <key_name>
			for _, key := range keysResponse.Keys {
				fmt.Printf("(string) %s\n", key)
			}
			// If no keys exist, no output is produced (empty response)
		}
	} else {
		fmt.Println("Error: Invalid response from server")
	}
}
