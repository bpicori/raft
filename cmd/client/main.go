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
)

func init() {
	logger.LogSetup()
}

func main() {
	flagValues := parseFlags()
	operation, key, _ := parseCommand(flagValues)
	cfg := buildServerConfig(flagValues.servers)

	switch operation {
	case "status":
		getClusterStatus(cfg)
	case "add":
		if key == "" {
			fmt.Fprintf(os.Stderr, "Key is required for add operation\n")
			flag.Usage()
			os.Exit(1)
		}
		fmt.Println("add operation not implemented")
	case "rm":
		if key == "" {
			fmt.Fprintf(os.Stderr, "Key is required for rm operation\n")
			flag.Usage()
			os.Exit(1)
		}
		fmt.Println("rm operation not implemented")
	default:
		fmt.Fprintf(os.Stderr, "Unknown operation. Use -status, -add, or -rm\n")
		flag.Usage()
		os.Exit(1)
	}
}

type flagValues struct {
	servers string
	status  bool
	add     string
	rm      string
}

func parseFlags() flagValues {
	servers := flag.String("servers", "", "Comma-separated list of servers in format host:port")
	status := flag.Bool("status", false, "Get the cluster status")
	add := flag.String("add", "", "Add a key to the storage (requires value argument)")
	rm := flag.String("rm", "", "Remove a key from the storage")

	flag.Parse()

	if *servers == "" {
		handleInvalidOperation("Servers list is required")
	}

	return flagValues{
		servers: *servers,
		status:  *status,
		add:     *add,
		rm:      *rm,
	}
}

func parseCommand(flags flagValues) (operation, key, value string) {
	if flags.status {
		operation = "status"
	} else if flags.add != "" {
		operation = "add"
		key = flags.add
		args := flag.Args()
		if len(args) > 0 {
			value = args[0]
		} else {
			handleInvalidOperation("Value is required for add operation")
		}
	} else if flags.rm != "" {
		operation = "rm"
		key = flags.rm
	} else {
		handleInvalidOperation("Must specify one operation: -status, -add, or -rm")
	}

	return
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

	for _, result := range results {
		fmt.Println(result)
	}
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

func handleInvalidOperation(message string) {
	fmt.Fprintf(os.Stderr, message+"\n")
	flag.Usage()
	os.Exit(1)
}
