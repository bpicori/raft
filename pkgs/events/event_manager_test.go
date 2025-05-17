package events

import (
	"bpicori/raft/pkgs/dto"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewEventManager(t *testing.T) {
	em := NewEventManager()

	// Verify all channels are initialized
	assert.NotNil(t, em.VoteRequestChan)
	assert.NotNil(t, em.VoteResponseChan)
	assert.NotNil(t, em.LogRequestChan)
	assert.NotNil(t, em.LogResponseChan)
	assert.NotNil(t, em.NodeStatusChan)
	assert.NotNil(t, em.AppendLogEntryChan)
	assert.NotNil(t, em.SetCommandRequestChan)
	assert.NotNil(t, em.GetCommandRequestChan)
	assert.NotNil(t, em.SyncCommandRequestChan)
	assert.NotNil(t, em.IncrCommandRequestChan)
	assert.NotNil(t, em.DecrCommandRequestChan)
	assert.NotNil(t, em.RemoveCommandRequestChan)

	// Verify timers are initialized and stopped
	assert.NotNil(t, em.ElectionTimer)
	assert.NotNil(t, em.HeartbeatTimer)

	// Check if timer channels are properly initialized
	assert.NotNil(t, em.ElectionTimerChan())
	assert.NotNil(t, em.HeartbeatTimerChan())
}

func TestElectionTimer(t *testing.T) {
	em := NewEventManager()

	// Test ResetElectionTimer
	testDuration := 20 * time.Millisecond
	em.ResetElectionTimer(testDuration)

	// The timer should fire after the specified duration
	select {
	case <-em.ElectionTimerChan():
		// Timer fired as expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Election timer did not fire in expected time")
	}

	// Test StopElectionTimer
	em.ResetElectionTimer(200 * time.Millisecond)
	time.Sleep(10 * time.Millisecond) // Give some time for the timer to be running

	// Timer should be running, so Stop should return true
	stopped := em.StopElectionTimer()
	if !stopped {
		// It's possible the timer fired before we could stop it
		// Try again with a longer duration to be sure
		em.ResetElectionTimer(500 * time.Millisecond)
		time.Sleep(10 * time.Millisecond)
		stopped = em.StopElectionTimer()
		assert.True(t, stopped, "Expected to be able to stop the election timer")
	}

	// After stopping, the timer shouldn't fire
	timerFired := false
	select {
	case <-em.ElectionTimerChan():
		timerFired = true
	case <-time.After(300 * time.Millisecond):
		// This is expected - timer is stopped
	}
	assert.False(t, timerFired, "Timer should not have fired after being stopped")
}

func TestHeartbeatTimer(t *testing.T) {
	em := NewEventManager()

	// Test ResetHeartbeatTimer
	testDuration := 20 * time.Millisecond
	em.ResetHeartbeatTimer(testDuration)

	// The timer should fire after the specified duration
	select {
	case <-em.HeartbeatTimerChan():
		// Timer fired as expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Heartbeat timer did not fire in expected time")
	}

	// Test StopHeartbeatTimer
	em.ResetHeartbeatTimer(200 * time.Millisecond)
	time.Sleep(10 * time.Millisecond) // Give some time for the timer to be running

	// Timer should be running, so Stop should return true
	stopped := em.StopHeartbeatTimer()
	if !stopped {
		// It's possible the timer fired before we could stop it
		// Try again with a longer duration to be sure
		em.ResetHeartbeatTimer(500 * time.Millisecond)
		time.Sleep(10 * time.Millisecond)
		stopped = em.StopHeartbeatTimer()
		assert.True(t, stopped, "Expected to be able to stop the heartbeat timer")
	}

	// After stopping, the timer shouldn't fire
	timerFired := false
	select {
	case <-em.HeartbeatTimerChan():
		timerFired = true
	case <-time.After(300 * time.Millisecond):
		// This is expected - timer is stopped
	}
	assert.False(t, timerFired, "Timer should not have fired after being stopped")
}

func TestDoubleResetTimer(t *testing.T) {
	em := NewEventManager()

	// Test double reset election timer
	em.ResetElectionTimer(100 * time.Millisecond)
	time.Sleep(10 * time.Millisecond)
	em.ResetElectionTimer(200 * time.Millisecond)

	// Timer should fire after the new duration, not the old one
	timerFiredEarly := false
	select {
	case <-em.ElectionTimerChan():
		timerFiredEarly = true
	case <-time.After(150 * time.Millisecond):
		// This is expected - timer should not fire at the first duration
	}
	assert.False(t, timerFiredEarly, "Timer should not have fired at the first duration")

	// Now it should fire at the new duration
	select {
	case <-em.ElectionTimerChan():
		// Timer fired at the new duration as expected
	case <-time.After(200 * time.Millisecond):
		t.Error("Election timer did not fire at the new duration")
	}

	// Test double reset heartbeat timer
	em.ResetHeartbeatTimer(100 * time.Millisecond)
	time.Sleep(10 * time.Millisecond)
	em.ResetHeartbeatTimer(200 * time.Millisecond)

	// Timer should fire after the new duration, not the old one
	timerFiredEarly = false
	select {
	case <-em.HeartbeatTimerChan():
		timerFiredEarly = true
	case <-time.After(150 * time.Millisecond):
		// This is expected - timer should not fire at the first duration
	}
	assert.False(t, timerFiredEarly, "Timer should not have fired at the first duration")

	// Now it should fire at the new duration
	select {
	case <-em.HeartbeatTimerChan():
		// Timer fired at the new duration as expected
	case <-time.After(200 * time.Millisecond):
		t.Error("Heartbeat timer did not fire at the new duration")
	}
}

func TestEventChannels(t *testing.T) {
	em := NewEventManager()

	// Test a representative Raft consensus channel (VoteRequestChan)
	go func() {
		voteReq := &dto.VoteRequest{
			NodeId:      "test-node",
			Term:        1,
			CandidateId: "test-candidate",
		}
		em.VoteRequestChan <- voteReq
	}()
	select {
	case req := <-em.VoteRequestChan:
		assert.Equal(t, "test-node", req.NodeId)
		assert.Equal(t, int32(1), req.Term)
		assert.Equal(t, "test-candidate", req.CandidateId)
	case <-time.After(100 * time.Millisecond):
		t.Error("Timed out waiting for vote request")
	}

	// Test a representative application command channel (SetCommandRequestChan)
	go func() {
		setEvent := SetCommandEvent{
			Payload: &dto.SetCommandRequest{
				Key:   "test-key",
				Value: "test-value",
			},
			Reply: make(chan *dto.GenericResponse),
		}
		em.SetCommandRequestChan <- setEvent
	}()
	select {
	case event := <-em.SetCommandRequestChan:
		assert.Equal(t, "test-key", event.Payload.Key)
		assert.Equal(t, "test-value", event.Payload.Value)
		assert.NotNil(t, event.Reply)
	case <-time.After(100 * time.Millisecond):
		t.Error("Timed out waiting for set command event")
	}

	// Test a representative event with custom structure (AppendLogEntryChan)
	go func() {
		appendEvent := AppendLogEntryEvent{
			Command: &dto.Command{
				Operation: "test",
			},
			Uuid:  "test-uuid",
			Reply: make(chan bool),
		}
		em.AppendLogEntryChan <- appendEvent
	}()
	select {
	case event := <-em.AppendLogEntryChan:
		assert.Equal(t, "test", event.Command.Operation)
		assert.Equal(t, "test-uuid", event.Uuid)
		assert.NotNil(t, event.Reply)
	case <-time.After(100 * time.Millisecond):
		t.Error("Timed out waiting for append log entry event")
	}
}
