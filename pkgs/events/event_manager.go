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
	ElectionTimer  Timer
	HeartbeatTimer Timer

	/* Application */
	SetCommandRequestChan       chan SetCommandEvent
	GetCommandRequestChan       chan GetCommandEvent
	SyncCommandRequestChan      chan SyncCommandEvent
	IncrCommandRequestChan      chan IncrCommandEvent
	DecrCommandRequestChan      chan DecrCommandEvent
	RemoveCommandRequestChan    chan RemoveCommandEvent
	LpushCommandRequestChan     chan LpushCommandEvent
	LpopCommandRequestChan      chan LpopCommandEvent
	LindexCommandRequestChan    chan LindexCommandEvent
	LlenCommandRequestChan      chan LlenCommandEvent
	KeysCommandRequestChan      chan KeysCommandEvent
	SaddCommandRequestChan      chan SaddCommandEvent
	SremCommandRequestChan      chan SremCommandEvent
	SismemberCommandRequestChan chan SismemberCommandEvent
	SinterCommandRequestChan    chan SinterCommandEvent
	ScardCommandRequestChan     chan ScardCommandEvent
	HsetCommandRequestChan      chan HsetCommandEvent
	HgetCommandRequestChan      chan HgetCommandEvent
	HmgetCommandRequestChan     chan HmgetCommandEvent
	HincrbyCommandRequestChan   chan HincrbyCommandEvent
}

type AppendLogEntryEvent struct {
	Command *dto.Command
	Uuid    string
	Reply   chan bool
}

func NewEventManager() *EventManager {
	electionTimer := NewRealTimer(1 * time.Second)
	heartbeatTimer := NewRealTimer(1 * time.Second)

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
		SetCommandRequestChan:       make(chan SetCommandEvent),
		GetCommandRequestChan:       make(chan GetCommandEvent),
		SyncCommandRequestChan:      make(chan SyncCommandEvent),
		IncrCommandRequestChan:      make(chan IncrCommandEvent),
		DecrCommandRequestChan:      make(chan DecrCommandEvent),
		RemoveCommandRequestChan:    make(chan RemoveCommandEvent),
		LpushCommandRequestChan:     make(chan LpushCommandEvent),
		LpopCommandRequestChan:      make(chan LpopCommandEvent),
		LindexCommandRequestChan:    make(chan LindexCommandEvent),
		LlenCommandRequestChan:      make(chan LlenCommandEvent),
		KeysCommandRequestChan:      make(chan KeysCommandEvent),
		SaddCommandRequestChan:      make(chan SaddCommandEvent),
		SremCommandRequestChan:      make(chan SremCommandEvent),
		SismemberCommandRequestChan: make(chan SismemberCommandEvent),
		SinterCommandRequestChan:    make(chan SinterCommandEvent),
		ScardCommandRequestChan:     make(chan ScardCommandEvent),
		HsetCommandRequestChan:      make(chan HsetCommandEvent),
		HgetCommandRequestChan:      make(chan HgetCommandEvent),
		HmgetCommandRequestChan:     make(chan HmgetCommandEvent),
		HincrbyCommandRequestChan:   make(chan HincrbyCommandEvent),
	}
}

func (el *EventManager) ResetElectionTimer(d time.Duration) {
	if !el.ElectionTimer.Stop() {
		select {
		case <-el.ElectionTimer.C():
		default:
		}
	}
	el.ElectionTimer.Reset(d)
}

func (el *EventManager) StopElectionTimer() bool {
	return el.ElectionTimer.Stop()
}

func (el *EventManager) ElectionTimerChan() <-chan time.Time {
	return el.ElectionTimer.C()
}

func (el *EventManager) ResetHeartbeatTimer(d time.Duration) {
	if !el.HeartbeatTimer.Stop() {
		select {
		case <-el.HeartbeatTimer.C():
		default:
		}
	}
	el.HeartbeatTimer.Reset(d)
}

func (el *EventManager) StopHeartbeatTimer() bool {
	return el.HeartbeatTimer.Stop()
}

func (el *EventManager) HeartbeatTimerChan() <-chan time.Time {
	return el.HeartbeatTimer.C()
}
