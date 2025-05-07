package main

import (
	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/core"
	"bpicori/raft/pkgs/dto"
	"log/slog"
)

func SetCommand(cfg *config.Config, key string, value string) {
	leader := findLeader(cfg)
	if leader == "" {
		slog.Error("No leader found")
		return
	}

	setCommand := &dto.RaftRPC{
		Type: core.SetCommand.String(),
		Args: &dto.RaftRPC_SetCommand{
			SetCommand: &dto.SetCommand{
				Key:   key,
				Value: value,
			},
		},
	}

	err := sendReceiveRPC(leader, setCommand, nil)
	if err != nil {
		slog.Error("Error sending set command", "error", err)
		return
	}

	slog.Info("Set command sent to leader", "key", key, "value", value)
}
