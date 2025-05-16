package commands

import (
	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/tcp"
	"fmt"
	"log/slog"
	"math/rand"
	"time"
)

func GetCommand(cfg *config.Config, key string) {
	leader := RandomServer(cfg)
	if leader == "" {
		slog.Error("No leader found")
		fmt.Println("Error: No leader found in the cluster. Try again later.")
		return
	}

	getCommand := &dto.RaftRPC{
		Type: consts.GetCommand.String(),
		Args: &dto.RaftRPC_GetCommandRequest{
			GetCommandRequest: &dto.GetCommandRequest{
				Key: key,
			},
		},
	}
	resp, err := tcp.SendSyncRPC(leader, getCommand)

	if err != nil {
		slog.Error("Error sending get command", "error", err)
		fmt.Println("Error: Failed to get key from leader. Try again later.")
		return
	}

	getResponse := resp.GetGetCommandResponse()

	if getResponse == nil {
		slog.Error("Error getting get command response", "error", err)
		fmt.Println("Error: Failed to get key from leader. Try again later.")
		return
	}

	fmt.Println(getResponse.Value)
}

func RandomServer(cfg *config.Config) string {
	rand.Seed(time.Now().UnixNano())

	servers := make([]string, 0, len(cfg.Servers))
	for _, server := range cfg.Servers {
		servers = append(servers, server.Addr)
	}

	return servers[rand.Intn(len(servers))]
}
