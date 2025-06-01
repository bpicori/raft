package application

import (
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/events"
	"context"
	"time"
)

// MockEventManager provides a mock implementation of EventManager for testing
type MockEventManager struct {
	events.EventManager
	// Additional fields for controlling mock behavior
	AppendLogEntryResponses map[string]bool // UUID -> response
	AppendLogEntryDelay     time.Duration
	DefaultResponse         bool // Default response when UUID not found
	SimulateTimeout         bool // If true, don't send any response
}

// NewMockEventManager creates a new mock event manager
func NewMockEventManager() *MockEventManager {
	return &MockEventManager{
		EventManager: events.EventManager{
			SetCommandRequestChan:    make(chan events.SetCommandEvent, 10),
			GetCommandRequestChan:    make(chan events.GetCommandEvent, 10),
			IncrCommandRequestChan:   make(chan events.IncrCommandEvent, 10),
			DecrCommandRequestChan:   make(chan events.DecrCommandEvent, 10),
			RemoveCommandRequestChan: make(chan events.RemoveCommandEvent, 10),
			LpushCommandRequestChan:  make(chan events.LpushCommandEvent, 10),
			LpopCommandRequestChan:   make(chan events.LpopCommandEvent, 10),
			LindexCommandRequestChan: make(chan events.LindexCommandEvent, 10),
			SyncCommandRequestChan:   make(chan events.SyncCommandEvent, 10),
			AppendLogEntryChan:       make(chan events.AppendLogEntryEvent, 10),
		},
		AppendLogEntryResponses: make(map[string]bool),
		AppendLogEntryDelay:     0,
		DefaultResponse:         true,
		SimulateTimeout:         false,
	}
}

// SetAppendLogEntryResponse sets the response for a specific UUID
func (m *MockEventManager) SetAppendLogEntryResponse(uuid string, success bool) {
	m.AppendLogEntryResponses[uuid] = success
}

// SetAppendLogEntryDelay sets the delay before responding to append log entry requests
func (m *MockEventManager) SetAppendLogEntryDelay(delay time.Duration) {
	m.AppendLogEntryDelay = delay
}

// SetSimulateTimeout sets whether to simulate timeout (no response)
func (m *MockEventManager) SetSimulateTimeout(timeout bool) {
	m.SimulateTimeout = timeout
}

// StartMockAppendLogEntryHandler starts a goroutine that handles append log entry events
func (m *MockEventManager) StartMockAppendLogEntryHandler(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-m.AppendLogEntryChan:
				go func(e events.AppendLogEntryEvent) {
					if m.SimulateTimeout {
						// Don't send any response to simulate timeout
						return
					}

					// Check if we have a specific response for this UUID
					if response, exists := m.AppendLogEntryResponses[e.Uuid]; exists {
						if response {
							if m.AppendLogEntryDelay > 0 {
								time.Sleep(m.AppendLogEntryDelay)
							}
							e.Reply <- true
						}
						// If response is false, we don't send anything (simulates timeout)
					} else {
						// Default behavior
						if m.DefaultResponse {
							if m.AppendLogEntryDelay > 0 {
								time.Sleep(m.AppendLogEntryDelay)
							}
							e.Reply <- true
						}
					}
				}(event)
			}
		}
	}()
}

// Helper function to create a context with timeout for testing
func createTestContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// Helper function to create test command events
func createSetCommandEvent(key, value string) events.SetCommandEvent {
	return events.SetCommandEvent{
		Payload: &dto.SetCommandRequest{
			Key:   key,
			Value: value,
		},
		Reply: make(chan *dto.GenericResponse, 1),
	}
}

func createGetCommandEvent(key string) events.GetCommandEvent {
	return events.GetCommandEvent{
		Payload: &dto.GetCommandRequest{
			Key: key,
		},
		Reply: make(chan *dto.GetCommandResponse, 1),
	}
}

func createIncrCommandEvent(key string) events.IncrCommandEvent {
	return events.IncrCommandEvent{
		Payload: &dto.IncrCommandRequest{
			Key: key,
		},
		Reply: make(chan *dto.IncrCommandResponse, 1),
	}
}

func createDecrCommandEvent(key string) events.DecrCommandEvent {
	return events.DecrCommandEvent{
		Payload: &dto.DecrCommandRequest{
			Key: key,
		},
		Reply: make(chan *dto.DecrCommandResponse, 1),
	}
}

func createRemoveCommandEvent(key string) events.RemoveCommandEvent {
	return events.RemoveCommandEvent{
		Payload: &dto.RemoveCommandRequest{
			Key: key,
		},
		Reply: make(chan *dto.GenericResponse, 1),
	}
}

func createLpushCommandEvent(key string, elements []string) events.LpushCommandEvent {
	return events.LpushCommandEvent{
		Payload: &dto.LpushCommandRequest{
			Key:      key,
			Elements: elements,
		},
		Reply: make(chan *dto.LpushCommandResponse, 1),
	}
}

func createLpopCommandEvent(key string) events.LpopCommandEvent {
	return events.LpopCommandEvent{
		Payload: &dto.LpopCommandRequest{
			Key: key,
		},
		Reply: make(chan *dto.LpopCommandResponse, 1),
	}
}

func createLindexCommandEvent(key string, index int32) events.LindexCommandEvent {
	return events.LindexCommandEvent{
		Payload: &dto.LindexCommandRequest{
			Key:   key,
			Index: index,
		},
		Reply: make(chan *dto.LindexCommandResponse, 1),
	}
}

// Helper to wait for response with timeout
func waitForResponse[T any](ch <-chan T, timeout time.Duration) (T, bool) {
	select {
	case response := <-ch:
		return response, true
	case <-time.After(timeout):
		var zero T
		return zero, false
	}
}
