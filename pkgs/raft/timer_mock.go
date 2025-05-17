package raft

import "time"

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