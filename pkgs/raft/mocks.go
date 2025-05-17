package raft

import (
	"bpicori/raft/pkgs/dto"
	"time"

	"github.com/stretchr/testify/mock"
)

type MockTimer struct {
	CChan chan time.Time
}

func NewMockTimer() *MockTimer {
	return &MockTimer{CChan: make(chan time.Time)}
}

func (mt *MockTimer) Stop() bool {
	return true
}

func (mt *MockTimer) Reset(d time.Duration) bool {
	return true
}

func (mt *MockTimer) C() <-chan time.Time {
	return mt.CChan
}

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
