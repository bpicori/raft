package main

import (
	"fmt"
	"log/slog"

	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/tcp"
)


func Incr(cfg *config.Config, key string) {
	leader := findLeader(cfg)
	if leader == "" {
		slog.Error("No leader found")
		fmt.Println("Error: No leader found in the cluster. Try again later.")
		return
	}

	incrCommand := &dto.RaftRPC{
		Type: consts.IncrCommand.String(),
		Args: &dto.RaftRPC_IncrCommandRequest{
			IncrCommandRequest: &dto.IncrCommandRequest{
				Key: key,
			},
		},
	}

	resp, err := tcp.SendSyncRPC(leader, incrCommand)

	if err != nil {
		slog.Error("Error sending incr command", "error", err)
		fmt.Printf("Error: Failed to increment key '%s': %v\n", key, err)
		return
	}

	slog.Info("Response from leader", "response", resp)
	
}