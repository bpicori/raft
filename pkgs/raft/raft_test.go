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
	"github.com/stretchr/testify/mock"
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

func mockedSingleConfig(nodeID string) config.Config {
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

func mockedMultiConfig(nodeID string, servers map[string]config.ServerConfig) config.Config {
	return config.Config{
		Servers:            servers,
		SelfID:             nodeID,
		SelfServer:         config.ServerConfig{ID: nodeID, Addr: servers[nodeID].Addr},
		PersistentFilePath: fmt.Sprintf("./test_raft_state_%s.json", nodeID),
		TimeoutMin:         2000,
		TimeoutMax:         3000,
	}
}

func mockedStorage() *MockStorage {
	mockStorage := new(MockStorage)
	mockStorage.On("PersistStateMachine", mock.Anything).Return(nil)
	mockStorage.On("LoadStateMachine").Return(&dto.StateMachineState{}, nil)
	return mockStorage
}

func TestClusterStartAsFollowers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	node := NewRaft(mockedEventManager(), mockedStorage(), mockedSingleConfig("node0"), ctx)
	wg.Add(1)
	go func() {
		defer wg.Done()
		node.Start()
	}()
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, consts.Follower, node.currentRole)

	cancel()
	wg.Wait()
}

func TestClusterCandidateStartsElection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	multiConfig := mockedMultiConfig("node0", map[string]config.ServerConfig{
		"node0": {ID: "node0", Addr: "localhost:8080"},
		"node1": {ID: "node1", Addr: "localhost:8081"},
		"node2": {ID: "node2", Addr: "localhost:8082"},
	})

	node := NewRaft(mockedEventManager(), mockedStorage(), multiConfig, ctx)

	wg.Add(1)
	go func() {
		defer wg.Done()
		node.Start()
	}()

	time.Sleep(100 * time.Millisecond)

	electionTimeChan := node.eventManager.ElectionTimer.(*MockTimer).CChan
	electionTimeChan <- time.Now()

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, consts.Candidate, node.GetCurrentRole())

	cancel()
	wg.Wait()
}

func TestClusterLeaderElection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	multiConfig := mockedMultiConfig("node0", map[string]config.ServerConfig{
		"node0": {ID: "node0", Addr: "localhost:8080"},
		"node1": {ID: "node1", Addr: "localhost:8081"},
		"node2": {ID: "node2", Addr: "localhost:8082"},
	})

	node := NewRaft(mockedEventManager(), mockedStorage(), multiConfig, ctx)
	wg.Add(1)
	go func() {
		defer wg.Done()
		node.Start()
	}()

	time.Sleep(100 * time.Millisecond)

	// start election
	electionTimeChan := node.eventManager.ElectionTimer.(*MockTimer).CChan
	electionTimeChan <- time.Now()

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, consts.Candidate, node.GetCurrentRole())

	// receive vote response from node2
	node.eventManager.VoteResponseChan <- &dto.VoteResponse{
		NodeId:      "node1",
		Term:        1,
		VoteGranted: true,
	}

	// receive vote response from node2
	node.eventManager.VoteResponseChan <- &dto.VoteResponse{
		NodeId:      "node2",
		Term:        1,
		VoteGranted: true,
	}

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, consts.Leader, node.GetCurrentRole())

	cancel()
	wg.Wait()
}
