package main

import (
	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/tcp"
	"fmt"
	"log/slog"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
)

func GetClusterStatus(cfg *config.Config) {
	results := make([]*dto.NodeStatusResponse, 0)

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

		results = append(results, nodeStatusResp)
	}

	printTable(results)
}

func printTable(results []*dto.NodeStatusResponse) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	// Simple, elegant style
	t.SetStyle(table.StyleRounded)

	// Add table header
	t.AppendHeader(table.Row{"Node ID", "Term", "Voted For", "Role", "Leader", "Log Count", "Last Log Entry"})

	// Add data rows
	for _, result := range results {
		t.AppendRow(table.Row{
			result.NodeId,
			result.CurrentTerm,
			result.VotedFor,
			result.CurrentRole,
			result.CurrentLeader,
			len(result.LogEntries),
			result.LogEntries[len(result.LogEntries)-1],
		})
	}

	// Render the table
	fmt.Println()
	t.Render()
	fmt.Println()
}
