package main

import (
	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/core"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/logger"
	"flag"
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"os"
	"strings"

	"google.golang.org/protobuf/proto"
)

func init() {
	logger.LogSetup()
}

func main() {
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

	switch *command {
	case "leader":
		getLeader(cfg)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", *command)
		flag.Usage()
		os.Exit(1)
	}
}

func getLeader(cfg *config.Config) {
	randomServer := pickRandomServer(cfg.Servers)

	conn, err := net.Dial("tcp", randomServer.Addr)
	if err != nil {
		slog.Error("Error getting connection", "error", err)
		os.Exit(1)
	}

	defer conn.Close()

	// send ClusterState RPC
	clusterStateReq := &dto.RaftRPC{
		Type: core.ClusterStateType.String(),
	}

	data, err := proto.Marshal(clusterStateReq)
	if err != nil {
		slog.Error("Error marshaling cluster state", "error", err)
		os.Exit(1)
	}

	_, err = conn.Write(data)
	if err != nil {
		slog.Error("Error sending cluster state", "error", err)
		os.Exit(1)
	}

	slog.Info("Sent ClusterState RPC", "server", randomServer.Addr)

	// receive ClusterState RaftRPC
	clusterStateResp := &dto.ClusterState{}

	data = make([]byte, 1024)
	n, err := conn.Read(data)
	if err != nil {
		slog.Error("Error reading data", "error", err)
		os.Exit(1)
	}

	err = proto.Unmarshal(data[:n], clusterStateResp)
	if err != nil {
		slog.Error("Error unmarshaling data", "error", err)
		os.Exit(1)
	}

	slog.Info("Received ClusterState RPC", "leader", clusterStateResp.Leader)
}

func pickRandomServer(servers map[string]config.ServerConfig) config.ServerConfig {
	keys := make([]string, 0, len(servers))
	for key := range servers {
		keys = append(keys, key)
	}

	randomIndex := rand.Intn(len(keys))
	randomKey := keys[randomIndex]

	return servers[randomKey]
}
