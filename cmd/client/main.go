package main

import (
	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/core"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/logger"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
)

func init() {
	logger.LogSetup()
}

func main() {
	flagValues, operation, args := parseFlags()
	cfg := buildServerConfig(flagValues.servers)

	switch operation {
	case "status":
		getClusterStatus(cfg)
	case "add":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Key and value are required for add operation\n")
			showUsage()
			os.Exit(1)
		}
		key := args[0]
		value := args[1]
		fmt.Printf("Adding key '%s' with value '%s' (not implemented)\n", key, value)
	case "rm":
		if len(args) < 1 {
			fmt.Fprintf(os.Stderr, "Key is required for rm operation\n")
			showUsage()
			os.Exit(1)
		}
		key := args[0]
		fmt.Printf("Removing key '%s' (not implemented)\n", key)
	default:
		fmt.Fprintf(os.Stderr, "Unknown operation: %s\n", operation)
		showUsage()
		os.Exit(1)
	}
}

type flagValues struct {
	servers string
}

func showUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s -servers=host1:port1,host2:port2,... <operation> [arguments...]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Operations:\n")
	fmt.Fprintf(os.Stderr, "  status            - Get the cluster status\n")
	fmt.Fprintf(os.Stderr, "  add <key> <value> - Add a key to the storage\n")
	fmt.Fprintf(os.Stderr, "  rm <key>          - Remove a key from the storage\n")
	flag.PrintDefaults()
}

func parseFlags() (flagValues, string, []string) {
	servers := flag.String("servers", "", "Comma-separated list of servers in format host:port")

	// Define custom usage function
	flag.Usage = showUsage

	flag.Parse()

	if *servers == "" {
		fmt.Fprintf(os.Stderr, "Servers list is required\n")
		showUsage()
		os.Exit(1)
	}

	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Operation required: status, add, or rm\n")
		showUsage()
		os.Exit(1)
	}

	operation := args[0]
	operationArgs := args[1:]

	return flagValues{
		servers: *servers,
	}, operation, operationArgs
}

func buildServerConfig(serversStr string) *config.Config {
	serverList := strings.Split(serversStr, ",")
	if len(serverList) == 0 {
		slog.Error("No servers provided")
		os.Exit(1)
	}

	serverConfigs := make(map[string]config.ServerConfig)
	for _, addr := range serverList {
		id := addr
		serverConfigs[id] = config.ServerConfig{
			ID:   id,
			Addr: addr,
		}
	}

	return &config.Config{
		Servers: serverConfigs,
	}
}

func getClusterStatus(cfg *config.Config) {
	results := make([]*dto.NodeStatus, 0)

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

		results = append(results, nodeStatusResp)
	}

	printTable(results)
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
		})
	}

	// Render the table
	fmt.Println()
	t.Render()
	fmt.Println()
}
