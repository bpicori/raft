package commands

import (
	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/tcp"
	"fmt"
	"log/slog"
)

// HsetCommand executes an HSET command against the Raft cluster.
// It sets field-value pairs in a hash stored at key.
// Returns the number of fields that were added (not updated).
// Output format: (integer) %d\n
func HsetCommand(cfg *config.Config, key string, fieldValuePairs []string) {
	leader := FindLeader(cfg)
	if leader == "" {
		slog.Error("No leader found")
		fmt.Println("Error: No leader found in the cluster. Try again later.")
		return
	}

	// Validate that we have an even number of arguments (field-value pairs)
	if len(fieldValuePairs)%2 != 0 {
		fmt.Println("Error: HSET requires field-value pairs. Each field must have a corresponding value.")
		return
	}

	// Convert field-value pairs to map
	fields := make(map[string]string)
	for i := 0; i < len(fieldValuePairs); i += 2 {
		field := fieldValuePairs[i]
		value := fieldValuePairs[i+1]
		fields[field] = value
	}

	hsetCommand := &dto.RaftRPC{
		Type: consts.HsetCommand.String(),
		Args: &dto.RaftRPC_HsetCommandRequest{
			HsetCommandRequest: &dto.HsetCommandRequest{
				Key:    key,
				Fields: fields,
			},
		},
	}

	resp, err := tcp.SendSyncRPC(leader, hsetCommand)

	if err != nil {
		slog.Error("Error sending hset command", "error", err)
		fmt.Printf("Error: Failed to hset key '%s': %v\n", key, err)
		return
	}

	hsetResponse := resp.GetHsetCommandResponse()
	if hsetResponse == nil {
		slog.Error("Error getting hset command response")
		fmt.Printf("Error: Failed to hset key '%s': invalid response\n", key)
		return
	}

	if hsetResponse.Error != "" {
		fmt.Printf("Error: %s\n", hsetResponse.Error)
		return
	}

	fmt.Printf("(integer) %d\n", hsetResponse.Added)
}
