package commands

import (
	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/tcp"
	"fmt"
	"log/slog"
)

// SismemberCommand executes a SISMEMBER command against the Raft cluster.
// It tests if member is a member of the set stored at key.
// Returns 1 if the element is a member of the set, 0 if not or if key does not exist.
// Output format: (integer) %d\n
func SismemberCommand(cfg *config.Config, key string, member string) {
	leader := FindLeader(cfg)
	if leader == "" {
		slog.Error("No leader found")
		fmt.Println("Error: No leader found in the cluster. Try again later.")
		return
	}

	sismemberCommand := &dto.RaftRPC{
		Type: consts.SismemberCommand.String(),
		Args: &dto.RaftRPC_SismemberCommandRequest{
			SismemberCommandRequest: &dto.SismemberCommandRequest{
				Key:    key,
				Member: member,
			},
		},
	}

	resp, err := tcp.SendSyncRPC(leader, sismemberCommand)

	if err != nil {
		slog.Error("Error sending sismember command", "error", err)
		fmt.Printf("Error: Failed to sismember key '%s': %v\n", key, err)
		return
	}

	if sismemberResponse := resp.GetSismemberCommandResponse(); sismemberResponse != nil {
		if sismemberResponse.Error != "" {
			fmt.Printf("Error: %s\n", sismemberResponse.Error)
		} else {
			fmt.Printf("(integer) %d\n", sismemberResponse.IsMember)
		}
	} else {
		fmt.Println("Error: Invalid response from server")
	}
}
