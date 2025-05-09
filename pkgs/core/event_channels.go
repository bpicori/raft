package core

import (
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
)

type Event[T any] struct {
	Type consts.RaftRPCType
	Data *T
}

type EventChannels struct {
	voteRequestChan  chan Event[dto.VoteRequest]
	voteResponseChan chan Event[dto.VoteResponse]
	logRequestChan   chan Event[dto.LogRequest]
	logResponseChan  chan Event[dto.LogResponse]
	setCommandChan   chan Event[dto.SetCommand]
}

func NewEventLoop() *EventChannels {
	return &EventChannels{
		voteRequestChan:  make(chan Event[dto.VoteRequest]),
		voteResponseChan: make(chan Event[dto.VoteResponse]),
		logRequestChan:   make(chan Event[dto.LogRequest]),
		logResponseChan:  make(chan Event[dto.LogResponse]),
		setCommandChan:   make(chan Event[dto.SetCommand]),
	}
}

func (el *EventChannels) Close() {
	close(el.voteRequestChan)
	close(el.voteResponseChan)
	close(el.logRequestChan)
	close(el.logResponseChan)
	close(el.setCommandChan)
}
