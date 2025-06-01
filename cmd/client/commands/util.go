package commands

import (
	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/tcp"
	"flag"
	"fmt"
	"log/slog"
	"os"
)

func ClearScreen() {
	fmt.Print("\033[H\033[2J")
}

func PrintHelp() {
	fmt.Println("\nAvailable commands:")
	fmt.Println("  status                                      - Get the cluster status")
	fmt.Println("  set <key> <value>                           - Add a key to the storage")
	fmt.Println("  get <key>                                   - Get a value from the storage")
	fmt.Println("  incr <key>                                  - Increment a numeric value")
	fmt.Println("  decr <key>                                  - Decrement a numeric value")
	fmt.Println("  rm <key>                                    - Remove a key from the storage")
	fmt.Println("  lpush <key> <element> [element2] [element3] - Prepend elements to list")
	fmt.Println("  lpop <key>                                  - Remove and return left element from list")
	fmt.Println("  lindex <key> <index>                        - Get element at index in list")
	fmt.Println("  llen <key>                                  - Get length of list")
	fmt.Println("  clear/cls                                   - Clear the screen")
	fmt.Println("  help                                        - Show this help message")
	fmt.Println("  exit/quit                                   - Exit the CLI")
	fmt.Println()
}

func ShowUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s -servers=host1:port1,host2:port2,... [operation] [arguments...]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Operations:\n")
	fmt.Fprintf(os.Stderr, "  status                                      - Get the cluster status\n")
	fmt.Fprintf(os.Stderr, "  set <key> <value>                           - Add a key to the storage\n")
	fmt.Fprintf(os.Stderr, "  get <key>                                   - Get a value from the storage\n")
	fmt.Fprintf(os.Stderr, "  incr <key>                                  - Increment a numeric value\n")
	fmt.Fprintf(os.Stderr, "  decr <key>                                  - Decrement a numeric value\n")
	fmt.Fprintf(os.Stderr, "  rm <key>                                    - Remove a key from the storage\n")
	fmt.Fprintf(os.Stderr, "  lpush <key> <element> [element2] [element3] - Prepend elements to list\n")
	fmt.Fprintf(os.Stderr, "  lpop <key>                                  - Remove and return left element from list\n")
	fmt.Fprintf(os.Stderr, "  lindex <key> <index>                        - Get element at index in list\n")
	fmt.Fprintf(os.Stderr, "  llen <key>                                  - Get length of list\n")
	fmt.Fprintf(os.Stderr, "If no operation is provided, the CLI will start in interactive mode.\n")
	flag.PrintDefaults()
}

func FindLeader(cfg *config.Config) string {
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
