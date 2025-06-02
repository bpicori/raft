package commands

import (
	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/tcp"
	"fmt"
	"log/slog"
)

// HmgetCommand executes an HMGET command against the Raft cluster.
// It returns the values associated with the specified fields in the hash stored at key.
// Returns "(nil)" for fields that don't exist.
// Output format: Multiple lines with (string) %s\n or (nil) for each field
func HmgetCommand(cfg *config.Config, key string, fields []string) {
	leader := FindLeader(cfg)
	if leader == "" {
		slog.Error("No leader found")
		fmt.Println("Error: No leader found in the cluster. Try again later.")
		return
	}

	if len(fields) == 0 {
		fmt.Println("Error: HMGET requires at least one field.")
		return
	}

	hmgetCommand := &dto.RaftRPC{
		Type: consts.HmgetCommand.String(),
		Args: &dto.RaftRPC_HmgetCommandRequest{
			HmgetCommandRequest: &dto.HmgetCommandRequest{
				Key:    key,
				Fields: fields,
			},
		},
	}

	resp, err := tcp.SendSyncRPC(leader, hmgetCommand)

	if err != nil {
		slog.Error("Error sending hmget command", "error", err)
		fmt.Printf("Error: Failed to hmget fields from key '%s': %v\n", key, err)
		return
	}

	hmgetResponse := resp.GetHmgetCommandResponse()
	if hmgetResponse == nil {
		slog.Error("Error getting hmget command response")
		fmt.Printf("Error: Failed to hmget fields from key '%s': invalid response\n", key)
		return
	}

	if hmgetResponse.Error != "" {
		fmt.Printf("Error: %s\n", hmgetResponse.Error)
		return
	}

	// Print each value in order
	for i, value := range hmgetResponse.Values {
		if i < len(fields) {
			if value == "(nil)" {
				fmt.Println("(nil)")
			} else {
				fmt.Printf("(string) %s\n", value)
			}
		}
	}
}
