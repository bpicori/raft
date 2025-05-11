package main

import (
	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/tcp"
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
		Type: consts.SetCommand.String(),
		Args: &dto.RaftRPC_SetCommandRequest{
			SetCommandRequest: &dto.SetCommandRequest{
				Key:   key,
				Value: value,
			},
		},
	}

	resp, err := tcp.SendSyncRPC(leader, setCommand)

	if err != nil {
		slog.Error("Error sending set command", "error", err)
		fmt.Printf("Error: Failed to set key '%s': %v\n", key, err)
		return
	}

	slog.Info("Response from leader", "response", resp)
}

func findLeader(cfg *config.Config) string {
	for _, server := range cfg.Servers {

		nodeStatusReq := &dto.RaftRPC{
			Type: consts.NodeStatus.String(),
			Args: &dto.RaftRPC_NodeStatusRequest{
				NodeStatusRequest: &dto.NodeStatusRequest{},
			},
		}

		rpcResp, err := tcp.SendSyncRPC(server.Addr, nodeStatusReq)
		if err != nil {
			slog.Error("Error in NodeStatus RPC", "server", server.Addr, "error", err)
			continue
		}

		nodeStatusResp := rpcResp.GetNodeStatusResponse()
		if nodeStatusResp.CurrentLeader != "" {
			return nodeStatusResp.CurrentLeader
		}
	}
	return ""
}
