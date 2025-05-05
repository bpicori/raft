package main

import (
	"bpicori/raft/pkgs/config"
	"math/rand"
)

func pickRandomServer(servers map[string]config.ServerConfig) config.ServerConfig {
	keys := make([]string, 0, len(servers))
	for key := range servers {
		keys = append(keys, key)
	}

	randomIndex := rand.Intn(len(keys))
	randomKey := keys[randomIndex]

	return servers[randomKey]
}