package main

import (
	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/core"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/logger"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"

	"google.golang.org/protobuf/proto"
)

func init() {
	logger.LogSetup()
}

func main() {
	command, cfg := parseConfigFromFlags()

	switch command {
	case "leader":
		getLeader(cfg)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		flag.Usage()
		os.Exit(1)
	}
}

func parseConfigFromFlags() (string, *config.Config) {
	command := flag.String("command", "", "The command to execute (leader)")
	servers := flag.String("servers", "", "Comma-separated list of servers in format host:port")
	flag.Parse()

	if *command == "" || *servers == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Parse servers directly from command line
	serverList := strings.Split(*servers, ",")
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

	cfg := &config.Config{
		Servers: serverConfigs,
	}

	return *command, cfg
}

func getLeader(cfg *config.Config) {
	results := make([]*dto.ClusterState, 0)

	for _, server := range cfg.Servers {
		clusterStateReq := &dto.RaftRPC{
			Type: core.ClusterStateType.String(),
		}
		clusterStateResp := &dto.ClusterState{}

		err := sendReceiveRPC(server.Addr, clusterStateReq, clusterStateResp)
		if err != nil {
			slog.Error("Error in ClusterState RPC", "server", server.Addr, "error", err)
			os.Exit(1)
		}

		results = append(results, clusterStateResp)
	}

	// print results
	for _, result := range results {
		fmt.Println(result)
	}
}

func openConnection(addr string) net.Conn {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		slog.Error("Error getting connection", "error", err)
		os.Exit(1)
	}

	return conn
}

func sendReceiveRPC(addr string, req proto.Message, resp proto.Message) error {
	conn := openConnection(addr)
	defer conn.Close()

	data, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	_, err = conn.Write(data)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	slog.Debug("Sent RPC request", "remote_addr", conn.RemoteAddr().String(), "type", fmt.Sprintf("%T", req))

	buffer := make([]byte, 4096) // Consider making buffer size configurable or dynamic
	n, err := conn.Read(buffer)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	if n == 0 {
		return errors.New("received empty response")
	}

	err = proto.Unmarshal(buffer[:n], resp)
	if err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	slog.Debug("Received RPC response", "remote_addr", conn.RemoteAddr().String(), "type", fmt.Sprintf("%T", resp))
	return nil
}
