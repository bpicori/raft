package core

import (
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
)

type Event[T any] struct {
	Type consts.RaftRPCType
	Data *T
}

type EventManager struct {
	VoteRequestChan  chan Event[dto.VoteRequest]
	VoteResponseChan chan Event[dto.VoteResponse]
	LogRequestChan   chan Event[dto.LogRequest]
	LogResponseChan  chan Event[dto.LogResponse]
	SetCommandChan   chan Event[dto.SetCommand]
}

func NewEventLoop() *EventManager {
	return &EventManager{
		VoteRequestChan:  make(chan Event[dto.VoteRequest]),
		VoteResponseChan: make(chan Event[dto.VoteResponse]),
		LogRequestChan:   make(chan Event[dto.LogRequest]),
		LogResponseChan:  make(chan Event[dto.LogResponse]),
		SetCommandChan:   make(chan Event[dto.SetCommand]),
	}
}

func (el *EventManager) Close() {
	close(el.VoteRequestChan)
	close(el.VoteResponseChan)
	close(el.LogRequestChan)
	close(el.LogResponseChan)
	close(el.SetCommandChan)
}
