package events

import (
	"bpicori/raft/pkgs/dto"
	"time"
)

type EventManager struct {
	/* Raft */
	VoteRequestChan    chan *dto.VoteRequest
	VoteResponseChan   chan *dto.VoteResponse
	LogRequestChan     chan *dto.LogRequest
	LogResponseChan    chan *dto.LogResponse
	NodeStatusChan     chan NodeStatusEvent
	AppendLogEntryChan chan AppendLogEntryEvent
	/* Raft Timers */
	ElectionTimer  *time.Timer
	HeartbeatTimer *time.Timer

	/* Application */
	SetCommandRequestChan  chan SetCommandEvent
	GetCommandRequestChan  chan GetCommandEvent
	SyncCommandRequestChan chan SyncCommandEvent
}

type SetCommandEvent struct {
	Payload *dto.SetCommandRequest
	Reply   chan *dto.OkResponse
}

type GetCommandEvent struct {
	Payload *dto.GetCommandRequest
	Reply   chan *dto.GetCommandResponse
}

type SyncCommandEvent struct {
	LogEntry *dto.LogEntry
}

type NodeStatusEvent struct {
	Reply chan *dto.NodeStatusResponse
}

type AppendLogEntryEvent struct {
	Command *dto.Command
	Uuid    string
	Reply   chan bool
}

func NewEventManager() *EventManager {
	electionTimer := time.NewTimer(1 * time.Second)
	if !electionTimer.Stop() {
		<-electionTimer.C
	}
	heartbeatTimer := time.NewTimer(1 * time.Second)
	if !heartbeatTimer.Stop() {
		<-heartbeatTimer.C
	}

	return &EventManager{
		VoteRequestChan:    make(chan *dto.VoteRequest),
		VoteResponseChan:   make(chan *dto.VoteResponse),
		LogRequestChan:     make(chan *dto.LogRequest),
		LogResponseChan:    make(chan *dto.LogResponse),
		NodeStatusChan:     make(chan NodeStatusEvent),
		AppendLogEntryChan: make(chan AppendLogEntryEvent),
		ElectionTimer:      electionTimer,
		HeartbeatTimer:     heartbeatTimer,

		/* Application */
		SetCommandRequestChan:  make(chan SetCommandEvent),
		GetCommandRequestChan:  make(chan GetCommandEvent),
		SyncCommandRequestChan: make(chan SyncCommandEvent),
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
	close(el.NodeStatusChan)
	close(el.AppendLogEntryChan)
	close(el.SetCommandRequestChan)
	close(el.GetCommandRequestChan)
	close(el.SyncCommandRequestChan)
	if el.ElectionTimer != nil {
		el.ElectionTimer.Stop()
	}
	if el.HeartbeatTimer != nil {
		el.HeartbeatTimer.Stop()
	}
}
