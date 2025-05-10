package events

import (
	"bpicori/raft/pkgs/dto"
	"time"
)

type EventManager struct {
	VoteRequestChan  chan *dto.VoteRequest
	VoteResponseChan chan *dto.VoteResponse
	LogRequestChan   chan *dto.LogRequest
	LogResponseChan  chan *dto.LogResponse
	SetCommandChan   chan *dto.SetCommandRequest
	NodeStatusChan   chan chan *dto.NodeStatusResponse

	ElectionTimer  *time.Timer
	HeartbeatTimer *time.Timer
}

func NewEventManager() *EventManager {
	// Initialize timers in a stopped state or with a placeholder duration.
	// They will be reset with actual durations when needed.
	electionTimer := time.NewTimer(1 * time.Second)
	if !electionTimer.Stop() {
		<-electionTimer.C // Ensure channel is drained if Stop returns false
	}
	heartbeatTimer := time.NewTimer(1 * time.Second)
	if !heartbeatTimer.Stop() {
		<-heartbeatTimer.C // Ensure channel is drained if Stop returns false
	}

	return &EventManager{
		VoteRequestChan:  make(chan *dto.VoteRequest),
		VoteResponseChan: make(chan *dto.VoteResponse),
		LogRequestChan:   make(chan *dto.LogRequest),
		LogResponseChan:  make(chan *dto.LogResponse),
		SetCommandChan:   make(chan *dto.SetCommandRequest),
		NodeStatusChan:   make(chan chan *dto.NodeStatusResponse),
		ElectionTimer:    electionTimer,
		HeartbeatTimer:   heartbeatTimer,
	}
}

func (el *EventManager) ResetElectionTimer(d time.Duration) {
	if !el.ElectionTimer.Stop() {
		select {
		case <-el.ElectionTimer.C:
		default:
		}
	}
	el.ElectionTimer.Reset(d)
}

func (el *EventManager) StopElectionTimer() bool {
	return el.ElectionTimer.Stop()
}

func (el *EventManager) ElectionTimerChan() <-chan time.Time {
	return el.ElectionTimer.C
}

func (el *EventManager) ResetHeartbeatTimer(d time.Duration) {
	if !el.HeartbeatTimer.Stop() {
		select {
		case <-el.HeartbeatTimer.C:
		default:
		}
	}
	el.HeartbeatTimer.Reset(d)
}

func (el *EventManager) StopHeartbeatTimer() bool {
	return el.HeartbeatTimer.Stop()
}

func (el *EventManager) HeartbeatTimerChan() <-chan time.Time {
	return el.HeartbeatTimer.C
}

func (el *EventManager) Close() {
	close(el.VoteRequestChan)
	close(el.VoteResponseChan)
	close(el.LogRequestChan)
	close(el.LogResponseChan)
	close(el.SetCommandChan)

	if el.ElectionTimer != nil {
		el.ElectionTimer.Stop()
	}
	if el.HeartbeatTimer != nil {
		el.HeartbeatTimer.Stop()
	}
}
