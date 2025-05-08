package main

import (
	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/core"
	"bpicori/raft/pkgs/dto"
	"fmt"
	"log/slog"
)

func SetCommand(cfg *config.Config, key string, value string) {
	leader := findLeader(cfg)
	if leader == "" {
		slog.Error("No leader found")
		fmt.Println("Error: No leader found in the cluster. Try again later.")
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

	err := sendReceiveRPC(leader, setCommand, &dto.RaftRPC{})
	if err != nil {
		slog.Error("Error sending set command", "error", err)
		fmt.Printf("Error: Failed to set key '%s': %v\n", key, err)
		return
	}

	slog.Info("Set command sent to leader", "key", key, "value", value)
}

func findLeader(cfg *config.Config) string {
	for _, server := range cfg.Servers {
		nodeStatusReq := &dto.RaftRPC{
			Type: core.NodeStatus.String(),
		}
		nodeStatusResp := &dto.NodeStatus{}

		err := sendReceiveRPC(server.Addr, nodeStatusReq, nodeStatusResp)
		if err != nil {
			slog.Error("Error in NodeStatus RPC", "server", server.Addr, "error", err)
			continue
		}

		if nodeStatusResp.CurrentLeader != "" {
			return nodeStatusResp.CurrentLeader
		}
	}
	return ""
}
