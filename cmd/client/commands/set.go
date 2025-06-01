package commands

import (
	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/tcp"
	"fmt"
	"log/slog"
)

func SetCommand(cfg *config.Config, key string, value string) {
	leader := FindLeader(cfg)
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

	setResponse := resp.GetGenericResponse()
	if setResponse == nil {
		slog.Error("Error getting set command response", "error", err)
		fmt.Printf("Error: Failed to set key '%s': %v\n", key, err)
		return
	}

	if setResponse.Ok {
		fmt.Println("OK")
	} else {
		fmt.Printf("Error: Failed to set key '%s'\n", key)
	}
}
