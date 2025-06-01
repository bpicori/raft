package commands

import (
	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/tcp"
	"fmt"
	"log/slog"
)

func Rm(cfg *config.Config, key string) {
	leader := FindLeader(cfg)
	if leader == "" {
		slog.Error("No leader found")
		fmt.Println("Error: No leader found in the cluster. Try again later.")
		return
	}

	rmCommand := &dto.RaftRPC{
		Type: consts.RemoveCommand.String(),
		Args: &dto.RaftRPC_RemoveCommandRequest{
			RemoveCommandRequest: &dto.RemoveCommandRequest{
				Key: key,
			},
		},
	}

	resp, err := tcp.SendSyncRPC(leader, rmCommand)

	if err != nil {
		slog.Error("Error sending rm command", "error", err)
		fmt.Printf("Error: Failed to remove key '%s': %v\n", key, err)
		return
	}

	genericResponse := resp.GetGenericResponse()
	if genericResponse == nil {
		slog.Error("Error getting rm command response", "error", err)
		fmt.Printf("Error: Failed to remove key '%s': %v\n", key, err)
		return
	}

	if genericResponse.Ok {
		fmt.Println("OK")
	} else {
		fmt.Printf("Error: Failed to remove key '%s'\n", key)
	}
}
