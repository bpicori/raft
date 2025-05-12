package main

import (
	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/tcp"
	"fmt"
	"log/slog"
)

func Decr(cfg *config.Config, key string) {
	leader := findLeader(cfg)
	if leader == "" {
		slog.Error("No leader found")
		fmt.Println("Error: No leader found in the cluster. Try again later.")
		return
	}

	decrCommand := &dto.RaftRPC{
		Type: consts.DecrCommand.String(),
		Args: &dto.RaftRPC_DecrCommandRequest{
			DecrCommandRequest: &dto.DecrCommandRequest{
				Key: key,
			},
		},
	}

	resp, err := tcp.SendSyncRPC(leader, decrCommand)

	if err != nil {
		slog.Error("Error sending decr command", "error", err)
		fmt.Printf("Error: Failed to decrement key '%s': %v\n", key, err)
		return
	}

	slog.Info("Response from leader", "response", resp)

}
