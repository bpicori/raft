package raft

import (
	"math/rand"
	"time"
)

type RequestResponse[Req, Resp any] struct {
	Request  Req
	Response Resp
	Done     chan struct{}
}

// randomTimeout returns a random number between 150ms and 300ms.
func randomTimeout(from int, to int) time.Duration {
	return time.Duration(rand.Intn(to-from)+from) * time.Millisecond
}
