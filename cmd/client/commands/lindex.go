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

// LindexCommand executes an LINDEX command against the Raft cluster.
// It returns the element at the specified index in the list stored at the given key.
// Index can be negative (Redis-style: -1 is last element, -2 is second-to-last, etc.)
// If the list is empty, key doesn't exist, or index is out of bounds, returns "(nil)".
func LindexCommand(cfg *config.Config, key string, indexStr string) {
	// Parse the index
	index, err := strconv.ParseInt(indexStr, 10, 32)
	if err != nil {
		fmt.Printf("Error: Invalid index '%s'. Index must be a valid integer.\n", indexStr)
		return
	}

	// Use any server for read operations (like GET)
	server := RandomServer(cfg)
	if server == "" {
		slog.Error("No server found")
		fmt.Println("Error: No server found in the cluster. Try again later.")
		return
	}

	lindexCommand := &dto.RaftRPC{
		Type: consts.LindexCommand.String(),
		Args: &dto.RaftRPC_LindexCommandRequest{
			LindexCommandRequest: &dto.LindexCommandRequest{
				Key:   key,
				Index: int32(index),
			},
		},
	}

	resp, err := tcp.SendSyncRPC(server, lindexCommand)

	if err != nil {
		slog.Error("Error sending lindex command", "error", err)
		fmt.Printf("Error: Failed to lindex key '%s': %v\n", key, err)
		return
	}

	if lindexResponse := resp.GetLindexCommandResponse(); lindexResponse != nil {
		if lindexResponse.Error != "" {
			fmt.Printf("Error: %s\n", lindexResponse.Error)
		} else {
			fmt.Println(lindexResponse.Element)
		}
	} else {
		fmt.Println("Error: Invalid response from server")
	}
}
