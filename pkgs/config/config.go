package config

import (
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
)

type ServerConfig struct {
	ID   string
	Addr string
}

type Config struct {
	Servers            map[string]ServerConfig
	SelfID             string
	SelfServer         ServerConfig
	PersistentFilePath string
	TimeoutMin         int
	TimeoutMax         int
	Heartbeat          int
}

func LoadConfig(isClient bool) (Config, error) {
	var (
		raftServers    string
		currentServer  string
		persistentPath string
		timeoutMin     string
		timeoutMax     string
		heartbeat      string
	)

	flag.StringVar(&raftServers, "servers", "", "Comma-separated list of Raft server addresses, e.g. localhost:8080,localhost:8081")
	flag.StringVar(&currentServer, "current", "", "Address of the current server")
	flag.StringVar(&persistentPath, "persistent-path", "./ignore", "Path to store persistent state")
	flag.StringVar(&timeoutMin, "timeout-min", "", "Minimum timeout for election")
	flag.StringVar(&timeoutMax, "timeout-max", "", "Maximum timeout for election")
	flag.StringVar(&heartbeat, "heartbeat", "", "Heartbeat interval")
	flag.Parse()

	// If not provided try to get from environment variables
	if raftServers == "" {
		raftServers = os.Getenv("RAFT_SERVERS")
	}
	if currentServer == "" {
		currentServer = os.Getenv("CURRENT_SERVER")
	}
	if persistentPath == "" {
		persistentPath = os.Getenv("PERSISTENT_FILE_PATH")
	}
	if timeoutMin == "" {
		timeoutMin = os.Getenv("TIMEOUT_MIN")
	}
	if timeoutMax == "" {
		timeoutMax = os.Getenv("TIMEOUT_MAX")
	}
	if heartbeat == "" {
		heartbeat = os.Getenv("HEARTBEAT")
	}

	if isClient {
		if raftServers == "" {
			return Config{}, fmt.Errorf("missing required flags: servers")
		}

		servers, err := parseServers(raftServers)
		if err != nil {
			return Config{}, err
		}

		return Config{Servers: servers}, nil

	}

	if raftServers == "" || currentServer == "" || persistentPath == "" || timeoutMin == "" || timeoutMax == "" || heartbeat == "" {
		missingFlags := []string{}
		if raftServers == "" {
			missingFlags = append(missingFlags, "servers")
		}
		if currentServer == "" {
			missingFlags = append(missingFlags, "current")
		}
		if persistentPath == "" {
			missingFlags = append(missingFlags, "persistent-path")
		}
		if timeoutMin == "" {
			missingFlags = append(missingFlags, "timeout-min")
		}
		if timeoutMax == "" {
			missingFlags = append(missingFlags, "timeout-max")
		}
		if heartbeat == "" {
			missingFlags = append(missingFlags, "heartbeat")
		}
		slog.Error("Missing required flags", "flags", missingFlags)

		return Config{}, fmt.Errorf("missing required flags: %v", missingFlags)
	}

	servers, err := parseServers(raftServers)
	if err != nil {
		return Config{}, err
	}

	selfServer := servers[currentServer]
	if selfServer == (ServerConfig{}) {
		return Config{}, fmt.Errorf("current server %s not found in RAFT_SERVERS", currentServer)
	}

	heartbeatInt, err := strconv.Atoi(heartbeat)
	if err != nil {
		return Config{}, fmt.Errorf("HEARTBEAT must be an integer")
	}

	timeoutMinInt, err := strconv.Atoi(timeoutMin)
	if err != nil {
		return Config{}, fmt.Errorf("TIMEOUT_MIN must be an integer")
	}

	timeoutMaxInt, err := strconv.Atoi(timeoutMax)
	if err != nil {
		return Config{}, fmt.Errorf("TIMEOUT_MAX must be an integer")
	}

	return Config{
		Servers:            servers,
		SelfID:             currentServer,
		SelfServer:         selfServer,
		PersistentFilePath: persistentPath,
		TimeoutMin:         timeoutMinInt,
		TimeoutMax:         timeoutMaxInt,
		Heartbeat:          heartbeatInt,
	}, nil
}

func parseServers(raftServers string) (map[string]ServerConfig, error) {
	servers := make(map[string]ServerConfig)

	for _, addr := range strings.Split(raftServers, ",") {
		host, port, err := net.SplitHostPort(addr)

		if err != nil {
			return servers, fmt.Errorf("invalid address format: %s", addr)
		}

		id := fmt.Sprintf("%s:%s", host, port)
		server := ServerConfig{ID: id, Addr: addr}
		servers[id] = server
	}

	return servers, nil
}
