package raft

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
	Data T
}

type EventChannels struct {
	requestVoteReqCh   chan Event[RequestVoteArgs]
	requestVoteRespCh  chan Event[RequestVoteReply]
	appendEntriesReqCh chan Event[AppendEntriesArgs]
	appendEntriesResCh chan Event[AppendEntriesReply]
	heartbeatReqCh     chan Event[AppendEntriesArgs]
	heartbeatRespCh    chan Event[AppendEntriesReply]
	leaderElectedCh    chan Event[bool]
}

func NewEventLoop() *EventChannels {
	return &EventChannels{
		requestVoteReqCh:   make(chan Event[RequestVoteArgs]),
		requestVoteRespCh:  make(chan Event[RequestVoteReply]),
		appendEntriesReqCh: make(chan Event[AppendEntriesArgs]),
		appendEntriesResCh: make(chan Event[AppendEntriesReply]),
		heartbeatReqCh:     make(chan Event[AppendEntriesArgs]),
		heartbeatRespCh:    make(chan Event[AppendEntriesReply]),
		leaderElectedCh:    make(chan Event[bool]),
	}
}

func (el *EventChannels) Close() {
	close(el.requestVoteReqCh)
	close(el.requestVoteRespCh)
	close(el.appendEntriesReqCh)
	close(el.appendEntriesResCh)
	close(el.heartbeatReqCh)
	close(el.heartbeatRespCh)
	close(el.leaderElectedCh)
}
