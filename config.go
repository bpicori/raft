package raft

import (
	"fmt"
	"net"
	"os"
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
}

func LoadConfig() (Config, error) {
	serversStr := os.Getenv("RAFT_SERVERS")
	currentSrv := os.Getenv("CURRENT_SERVER")
	persistentFilePath := os.Getenv("PERSISTENT_FILE_PATH")
	httpPort := os.Getenv("HTTP_PORT")

	if serversStr == "" || currentSrv == "" || httpPort == "" {
		return Config{}, fmt.Errorf("RAFT_SERVERS, CURRENT_SERVER and HTTP_PORT environment variables must be all set")
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

	return Config{
		Servers:            servers,
		SelfID:             selfID,
		SelfServer:         selfServer,
		PersistentFilePath: persistentFilePath,
		HttpPort:           httpPort,
	}, nil
}
