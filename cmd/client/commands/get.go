package commands

import (
	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/tcp"
	"fmt"
	"log/slog"
)

func GetCommand(cfg *config.Config, key string) {
	leader := FindLeader(cfg)
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

