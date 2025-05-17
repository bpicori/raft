package events

import "time"

type Timer interface {
	Stop() bool
	Reset(d time.Duration) bool
	C() <-chan time.Time
}

type RealTimer struct {
	timer *time.Timer
}

func NewRealTimer(d time.Duration) *RealTimer {
	return &RealTimer{timer: time.NewTimer(d)}
}

func (rt *RealTimer) Stop() bool {
	return rt.timer.Stop()
}

func (rt *RealTimer) Reset(d time.Duration) bool {
	return rt.timer.Reset(d)
}

func (rt *RealTimer) C() <-chan time.Time {
	return rt.timer.C
}
