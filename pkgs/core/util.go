package core
import (
	"log"
	"math/rand"
	"time"
)

var (
	WarningLogger *log.Logger
	InfoLogger    *log.Logger
	ErrorLogger   *log.Logger
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

func roleToString(role Role) string {
	switch role {
	case Follower:
		return "Follower"
	case Candidate:
		return "Candidate"
	case Leader:
		return "Leader"
	default:
		return "Unknown"
	}
}
