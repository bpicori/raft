package raft

import (
	"fmt"
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
	HttpPort           string
	TimeoutMin         int
	TimeoutMax         int
	Heartbeat          int
}

func LoadConfig() (Config, error) {
	serversStr := os.Getenv("RAFT_SERVERS")
	currentSrv := os.Getenv("CURRENT_SERVER")
	persistentFilePath := os.Getenv("PERSISTENT_FILE_PATH")
	httpPort := os.Getenv("HTTP_PORT")
	timeoutMin := os.Getenv("TIMEOUT_MIN")
	timeoutMax := os.Getenv("TIMEOUT_MAX")
	heartbeat := os.Getenv("HEARTBEAT")

	if serversStr == "" || currentSrv == "" || httpPort == "" || timeoutMin == "" || timeoutMax == "" || heartbeat == "" {
		return Config{}, fmt.Errorf("RAFT_SERVERS, CURRENT_SERVER, HTTP_PORT, TIMEOUT_MIN, TIMEOUT_MAX, HEARTBEAT environment variables must be all set")
	}

	if persistentFilePath == "" {
		persistentFilePath = "./ignore"
	}

	servers := make(map[string]ServerConfig)
	var selfID string
	var selfServer ServerConfig

	for _, addr := range strings.Split(serversStr, ",") {
		host, port, err := net.SplitHostPort(addr)

		if err != nil {
			return Config{}, fmt.Errorf("invalid address format: %s", addr)
		}

		id := fmt.Sprintf("%s:%s", host, port)
		server := ServerConfig{ID: id, Addr: addr}
		servers[id] = server

		if addr == currentSrv {
			selfID = id
			selfServer = server
		}

	}

	if selfID == "" {
		return Config{}, fmt.Errorf("current server %s not found in RAFT_SERVERS", currentSrv)
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
		SelfID:             selfID,
		SelfServer:         selfServer,
		PersistentFilePath: persistentFilePath,
		HttpPort:           httpPort,
		TimeoutMin:         timeoutMinInt,
		TimeoutMax:         timeoutMaxInt,
		Heartbeat:          heartbeatInt,
	}, nil
}
