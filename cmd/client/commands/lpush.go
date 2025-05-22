package commands

import (
	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/tcp"
	"fmt"
	"log/slog"
)

func LpushCommand(cfg *config.Config, key string, elements []string) {
	leader := FindLeader(cfg)
	if leader == "" {
		slog.Error("No leader found")
		fmt.Println("Error: No leader found in the cluster. Try again later.")
		return
	}

	lpushCommand := &dto.RaftRPC{
		Type: consts.LpushCommand.String(),
		Args: &dto.RaftRPC_LpushCommandRequest{
			LpushCommandRequest: &dto.LpushCommandRequest{
				Key:      key,
				Elements: elements,
			},
		},
	}

	resp, err := tcp.SendSyncRPC(leader, lpushCommand)

	if err != nil {
		slog.Error("Error sending lpush command", "error", err)
		fmt.Printf("Error: Failed to lpush to key '%s': %v\n", key, err)
		return
	}

	lpushResponse := resp.GetLpushCommandResponse()
	if lpushResponse == nil {
		slog.Error("Error getting lpush command response", "error", err)
		fmt.Printf("Error: Failed to lpush to key '%s': %v\n", key, err)
		return
	}

	if lpushResponse.Error != "" {
		fmt.Printf("Error: %s\n", lpushResponse.Error)
		return
	}

	fmt.Printf("(integer) %d\n", lpushResponse.Length)
}
