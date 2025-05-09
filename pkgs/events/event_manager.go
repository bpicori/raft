package events

import (
	"bpicori/raft/pkgs/dto"
)

type EventManager struct {
	VoteRequestChan  chan *dto.VoteRequest
	VoteResponseChan chan *dto.VoteResponse
	LogRequestChan   chan *dto.LogRequest
	LogResponseChan  chan *dto.LogResponse
	SetCommandChan   chan *dto.SetCommand
}

func NewEventManager() *EventManager {
	return &EventManager{
		VoteRequestChan:  make(chan *dto.VoteRequest),
		VoteResponseChan: make(chan *dto.VoteResponse),
		LogRequestChan:   make(chan *dto.LogRequest),
		LogResponseChan:  make(chan *dto.LogResponse),
		SetCommandChan:   make(chan *dto.SetCommand),
	}
}

func (el *EventManager) Close() {
	close(el.VoteRequestChan)
	close(el.VoteResponseChan)
	close(el.LogRequestChan)
	close(el.LogResponseChan)
	close(el.SetCommandChan)
}
