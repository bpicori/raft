package core

import "bpicori/raft/pkgs/dto"

type EventType int

const (
	RequestVoteReq EventType = iota
	RequestVoteResp
	AppendEntriesReq
	AppendEntriesResp
	HeartbeatReq
	HeartbeatResp
)

type Event[T any] struct {
	Type EventType
	Data *T
}

type EventChannels struct {
	voteRequestChan    chan Event[dto.RequestVoteArgs]
	voteResponseChan   chan Event[dto.RequestVoteReply]
	appendEntriesReqCh chan Event[dto.AppendEntriesArgs]
	appendEntriesResCh chan Event[dto.AppendEntriesReply]
	heartbeatReqCh     chan Event[dto.AppendEntriesArgs]
	heartbeatRespCh    chan Event[dto.AppendEntriesReply]
	leaderElectedCh    chan Event[bool]
}

func NewEventLoop() *EventChannels {
	return &EventChannels{
		voteRequestChan:    make(chan Event[dto.RequestVoteArgs]),
		voteResponseChan:   make(chan Event[dto.RequestVoteReply]),
		appendEntriesReqCh: make(chan Event[dto.AppendEntriesArgs]),
		appendEntriesResCh: make(chan Event[dto.AppendEntriesReply]),
		heartbeatReqCh:     make(chan Event[dto.AppendEntriesArgs]),
		heartbeatRespCh:    make(chan Event[dto.AppendEntriesReply]),
		leaderElectedCh:    make(chan Event[bool]),
	}
}

func (el *EventChannels) Close() {
	close(el.voteRequestChan)
	close(el.voteResponseChan)
	close(el.appendEntriesReqCh)
	close(el.appendEntriesResCh)
	close(el.heartbeatReqCh)
	close(el.heartbeatRespCh)
	close(el.leaderElectedCh)
}
