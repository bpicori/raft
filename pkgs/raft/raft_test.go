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

type MockStorage struct {
	mock.Mock
}

func (ms *MockStorage) PersistStateMachine(state *dto.StateMachineState) error {
	args := ms.Called(state)
	return args.Error(0)
}

func (ms *MockStorage) LoadStateMachine() (*dto.StateMachineState, error) {
	args := ms.Called()
	return args.Get(0).(*dto.StateMachineState), args.Error(1)
}

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

func TestClusterStartAsFollowers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	mockStorage := new(MockStorage)
	mockStorage.On("PersistStateMachine", mock.Anything).Return(nil)
	mockStorage.On("LoadStateMachine").Return(&dto.StateMachineState{}, nil)

	node := NewRaft(mockedEventManager(), mockStorage, mockedConfig("node0"), ctx)

	wg.Add(1)
	go func() {
		defer wg.Done()
		node.Start()
	}()

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, consts.Follower, node.currentRole)

	mockStorage.AssertCalled(t, "LoadStateMachine")

	cancel()
	fmt.Println("Waiting for all goroutines to stop...")
	wg.Wait()
	mockStorage.AssertCalled(t, "PersistStateMachine", mock.Anything)
	fmt.Println("All goroutines stopped")
}

// func TestClusterCandidateStartsElection(t *testing.T) {
// 	ctx, cancel := context.WithCancel(context.Background())
// 	wg := &sync.WaitGroup{}

// }
