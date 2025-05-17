package raft

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/events"

	"github.com/stretchr/testify/assert"
)

func mockedEventManager() *events.EventManager {
	return &events.EventManager{
		ElectionTimer:      NewMockTimer(),
		HeartbeatTimer:     NewMockTimer(),
		VoteRequestChan:    make(chan *dto.VoteRequest),
		VoteResponseChan:   make(chan *dto.VoteResponse),
		LogRequestChan:     make(chan *dto.LogRequest),
		LogResponseChan:    make(chan *dto.LogResponse),
		NodeStatusChan:     make(chan events.NodeStatusEvent),
		AppendLogEntryChan: make(chan events.AppendLogEntryEvent),
	}
}

func mockedConfig(nodeID string) config.Config {
	return config.Config{
		Servers:            make(map[string]config.ServerConfig),
		SelfID:             nodeID,
		SelfServer:         config.ServerConfig{ID: nodeID, Addr: "localhost:8080"},
		PersistentFilePath: fmt.Sprintf("./test_raft_state_%s.json", nodeID),
		TimeoutMin:         2000,
		TimeoutMax:         3000,
		Heartbeat:          1000,
	}
}

func mockedRaft(nodeID string, context context.Context, wg *sync.WaitGroup) *Raft {
	return NewRaft(mockedEventManager(), mockedConfig(nodeID), context, wg)
}

func TestClusterStartAsFollowers(t *testing.T) {
	// test timeout
	go func() {
		time.Sleep(5 * time.Second)
		panic("test timeout")
	}()

	// Create a context and wait group
	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	// Create a cluster of 3 nodes
	nodes := make(map[string]*Raft)
	for i := 0; i < 3; i++ {
		nodes[fmt.Sprintf("node%d", i)] = mockedRaft(fmt.Sprintf("node%d", i), ctx, wg)
	}

	// Start all nodes
	for _, node := range nodes {
		go node.Start()
	}

	// expect all nodes to be followers
	for _, node := range nodes {
		assert.Equal(t, consts.Follower, node.currentRole)
	}

	cancel()
	wg.Wait()
}
