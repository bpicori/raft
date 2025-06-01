package application

import (
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/events"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeys(t *testing.T) {
	tests := []struct {
		name         string
		setupHashMap func()
		expectedKeys []string
	}{
		{
			name: "empty store should return empty list",
			setupHashMap: func() {
				resetHashMap()
			},
			expectedKeys: []string{},
		},
		{
			name: "single key should return one key",
			setupHashMap: func() {
				resetHashMap()
				setHashMapValue("key1", "value1")
			},
			expectedKeys: []string{"key1"},
		},
		{
			name: "multiple keys should return all keys",
			setupHashMap: func() {
				resetHashMap()
				setHashMapValue("key1", "value1")
				setHashMapValue("key2", "value2")
				setHashMapValue("key3", "value3")
			},
			expectedKeys: []string{"key1", "key2", "key3"},
		},
		{
			name: "mixed data types should return all keys",
			setupHashMap: func() {
				resetHashMap()
				setHashMapValue("string_key", "string_value")
				setHashMapValue("number_key", "42")
				setHashMapValue("list_key", `["item1", "item2"]`)
			},
			expectedKeys: []string{"string_key", "number_key", "list_key"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupHashMap()
			mockEM := NewMockEventManager()
			event := createKeysCommandEvent()

			Keys(&mockEM.EventManager, &event)

			response, ok := waitForKeysResponse(event.Reply, 100*time.Millisecond)
			require.True(t, ok, "Should receive response within timeout")
			assert.Empty(t, response.Error, "Should not have error")

			// Since sync.Map iteration order is not guaranteed, we need to check
			// that all expected keys are present, regardless of order
			assert.ElementsMatch(t, tt.expectedKeys, response.Keys, "Should return all expected keys")
		})
	}
}


// Helper function to create a KEYS command event
func createKeysCommandEvent() events.KeysCommandEvent {
	return events.KeysCommandEvent{
		Payload: &dto.KeysCommandRequest{},
		Reply:   make(chan *dto.KeysCommandResponse, 1),
	}
}

// Helper function to wait for KEYS response with timeout
func waitForKeysResponse(replyChan chan *dto.KeysCommandResponse, timeout time.Duration) (*dto.KeysCommandResponse, bool) {
	select {
	case response := <-replyChan:
		return response, true
	case <-time.After(timeout):
		return nil, false
	}
}
