package core

import "bpicori/raft/pkgs/dto"

type Event[T any] struct {
	Type RaftRPCType
	Data *T
}

type EventChannels struct {
	voteRequestChan    chan Event[dto.VoteRequest]
	voteResponseChan   chan Event[dto.VoteResponse]
	appendEntriesReqCh chan Event[dto.AppendEntriesArgs]
	appendEntriesResCh chan Event[dto.AppendEntriesReply]
	heartbeatReqCh     chan Event[dto.AppendEntriesArgs]
	heartbeatRespCh    chan Event[dto.AppendEntriesReply]
	leaderElectedCh    chan Event[bool]
}

func NewEventLoop() *EventChannels {
	return &EventChannels{
		voteRequestChan:    make(chan Event[dto.VoteRequest]),
		voteResponseChan:   make(chan Event[dto.VoteResponse]),
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
