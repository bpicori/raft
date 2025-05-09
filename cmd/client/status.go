package main

import (
	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/core"
	"bpicori/raft/pkgs/dto"
	"fmt"
	"log/slog"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
)

func GetClusterStatus(cfg *config.Config) {
	results := make([]*dto.NodeStatus, 0)

	for _, server := range cfg.Servers {
		nodeStatusReq := &dto.RaftRPC{
			Type: consts.NodeStatus.String(),
		}

		rpcResp, err := core.SendSyncRPC(server.Addr, nodeStatusReq)

		if err != nil {
			slog.Error("Error in NodeStatus RPC", "server", server.Addr, "error", err)
			continue
		}

		nodeStatusResp := rpcResp.GetNodeStatus()

		results = append(results, nodeStatusResp)
	}

	printTable(results)
}

func printTable(results []*dto.NodeStatus) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	// Simple, elegant style
	t.SetStyle(table.StyleRounded)

	// Add table header
	t.AppendHeader(table.Row{"Node ID", "Term", "Voted For", "Role", "Leader"})

	// Add data rows
	for _, result := range results {
		t.AppendRow(table.Row{
			result.NodeId,
			result.CurrentTerm,
			result.VotedFor,
			result.CurrentRole,
			result.CurrentLeader,
			result.LogEntries,
		})
	}

	// Render the table
	fmt.Println()
	t.Render()
	fmt.Println()
}
