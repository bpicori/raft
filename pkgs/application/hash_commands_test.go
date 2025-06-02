package application

import (
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/events"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// Enable test mode for faster timeouts
	SetTestMode(true)
}

// Test helper functions for hash commands
func createHsetCommandEvent(key string, fields map[string]string) events.HsetCommandEvent {
	return events.HsetCommandEvent{
		Payload: &dto.HsetCommandRequest{
			Key:    key,
			Fields: fields,
		},
		Reply: make(chan *dto.HsetCommandResponse, 1),
	}
}

func createHgetCommandEvent(key, field string) events.HgetCommandEvent {
	return events.HgetCommandEvent{
		Payload: &dto.HgetCommandRequest{
			Key:   key,
			Field: field,
		},
		Reply: make(chan *dto.HgetCommandResponse, 1),
	}
}

func createHmgetCommandEvent(key string, fields []string) events.HmgetCommandEvent {
	return events.HmgetCommandEvent{
		Payload: &dto.HmgetCommandRequest{
			Key:    key,
			Fields: fields,
		},
		Reply: make(chan *dto.HmgetCommandResponse, 1),
	}
}

func createHincrbyCommandEvent(key, field string, increment int32) events.HincrbyCommandEvent {
	return events.HincrbyCommandEvent{
		Payload: &dto.HincrbyCommandRequest{
			Key:       key,
			Field:     field,
			Increment: increment,
		},
		Reply: make(chan *dto.HincrbyCommandResponse, 1),
	}
}

func waitForHsetResponse(ch chan *dto.HsetCommandResponse, timeout time.Duration) (*dto.HsetCommandResponse, bool) {
	select {
	case response := <-ch:
		return response, true
	case <-time.After(timeout):
		return nil, false
	}
}

func waitForHgetResponse(ch chan *dto.HgetCommandResponse, timeout time.Duration) (*dto.HgetCommandResponse, bool) {
	select {
	case response := <-ch:
		return response, true
	case <-time.After(timeout):
		return nil, false
	}
}

func waitForHmgetResponse(ch chan *dto.HmgetCommandResponse, timeout time.Duration) (*dto.HmgetCommandResponse, bool) {
	select {
	case response := <-ch:
		return response, true
	case <-time.After(timeout):
		return nil, false
	}
}

func waitForHincrbyResponse(ch chan *dto.HincrbyCommandResponse, timeout time.Duration) (*dto.HincrbyCommandResponse, bool) {
	select {
	case response := <-ch:
		return response, true
	case <-time.After(timeout):
		return nil, false
	}
}

// Set up a hash in the hashMap for testing
func setupTestHash(key string, fields map[string]string) {
	jsonData, _ := json.Marshal(fields)
	hashMap.Store(key, string(jsonData))
}

func TestHset(t *testing.T) {
	tests := []struct {
		name          string
		key           string
		fields        map[string]string
		setupHashMap  func()
		expectedAdded int32
		expectError   bool
		mockSuccess   bool
	}{
		{
			name: "new hash with single field",
			key:  "hash1",
			fields: map[string]string{
				"field1": "value1",
			},
			setupHashMap: func() {
				resetHashMap()
			},
			expectedAdded: 1,
			expectError:   false,
			mockSuccess:   true,
		},
		{
			name: "new hash with multiple fields",
			key:  "hash2",
			fields: map[string]string{
				"field1": "value1",
				"field2": "value2",
				"field3": "value3",
			},
			setupHashMap: func() {
				resetHashMap()
			},
			expectedAdded: 3,
			expectError:   false,
			mockSuccess:   true,
		},
		{
			name: "update existing field",
			key:  "hash3",
			fields: map[string]string{
				"field1": "newvalue1",
			},
			setupHashMap: func() {
				resetHashMap()
				setupTestHash("hash3", map[string]string{
					"field1": "oldvalue1",
					"field2": "value2",
				})
			},
			expectedAdded: 0,
			expectError:   false,
			mockSuccess:   true,
		},
		{
			name: "add new field to existing hash",
			key:  "hash4",
			fields: map[string]string{
				"field3": "value3",
			},
			setupHashMap: func() {
				resetHashMap()
				setupTestHash("hash4", map[string]string{
					"field1": "value1",
					"field2": "value2",
				})
			},
			expectedAdded: 1,
			expectError:   false,
			mockSuccess:   true,
		},
		{
			name: "empty key should fail validation",
			key:  "",
			fields: map[string]string{
				"field1": "value1",
			},
			setupHashMap: func() {
				resetHashMap()
			},
			expectedAdded: 0,
			expectError:   true,
			mockSuccess:   false,
		},
		{
			name:   "empty fields should fail validation",
			key:    "hash5",
			fields: map[string]string{},
			setupHashMap: func() {
				resetHashMap()
			},
			expectedAdded: 0,
			expectError:   true,
			mockSuccess:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupHashMap()
			mockEM := NewMockEventManager()
			ctx, cancel := createTestContext(200 * time.Millisecond)
			defer cancel()

			mockEM.StartMockAppendLogEntryHandler(ctx)

			event := createHsetCommandEvent(tt.key, tt.fields)

			// Set up mock response
			if !tt.mockSuccess {
				mockEM.SetSimulateTimeout(true)
			}

			Hset(&mockEM.EventManager, &event)

			response, ok := waitForHsetResponse(event.Reply, 300*time.Millisecond)
			require.True(t, ok, "Should receive response")
			assert.Equal(t, tt.expectedAdded, response.Added, "Added count should match expected")

			if tt.expectError {
				assert.NotEmpty(t, response.Error, "Should have error message")
			} else if tt.mockSuccess {
				assert.Empty(t, response.Error, "Should not have error message")
			}
		})
	}
}

func TestHget(t *testing.T) {
	tests := []struct {
		name          string
		key           string
		field         string
		setupHashMap  func()
		expectedValue string
		expectError   bool
	}{
		{
			name:  "get existing field",
			key:   "hash1",
			field: "field1",
			setupHashMap: func() {
				resetHashMap()
				setupTestHash("hash1", map[string]string{
					"field1": "value1",
					"field2": "value2",
				})
			},
			expectedValue: "value1",
			expectError:   false,
		},
		{
			name:  "get non-existing field",
			key:   "hash1",
			field: "field3",
			setupHashMap: func() {
				resetHashMap()
				setupTestHash("hash1", map[string]string{
					"field1": "value1",
					"field2": "value2",
				})
			},
			expectedValue: "(nil)",
			expectError:   false,
		},
		{
			name:  "get from non-existing hash",
			key:   "nonexistent",
			field: "field1",
			setupHashMap: func() {
				resetHashMap()
			},
			expectedValue: "(nil)",
			expectError:   false,
		},
		{
			name:  "empty key should fail validation",
			key:   "",
			field: "field1",
			setupHashMap: func() {
				resetHashMap()
			},
			expectedValue: "",
			expectError:   true,
		},
		{
			name:  "empty field should fail validation",
			key:   "hash1",
			field: "",
			setupHashMap: func() {
				resetHashMap()
			},
			expectedValue: "",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupHashMap()
			mockEM := NewMockEventManager()
			event := createHgetCommandEvent(tt.key, tt.field)

			Hget(&mockEM.EventManager, &event)

			response, ok := waitForHgetResponse(event.Reply, 100*time.Millisecond)
			require.True(t, ok, "Should receive response within timeout")
			assert.Equal(t, tt.expectedValue, response.Value, "Value should match expected")

			if tt.expectError {
				assert.NotEmpty(t, response.Error, "Should have error message")
			} else {
				assert.Empty(t, response.Error, "Should not have error message")
			}
		})
	}
}

func TestHmget(t *testing.T) {
	tests := []struct {
		name           string
		key            string
		fields         []string
		setupHashMap   func()
		expectedValues []string
		expectError    bool
	}{
		{
			name:   "get multiple existing fields",
			key:    "hash1",
			fields: []string{"field1", "field2"},
			setupHashMap: func() {
				resetHashMap()
				setupTestHash("hash1", map[string]string{
					"field1": "value1",
					"field2": "value2",
					"field3": "value3",
				})
			},
			expectedValues: []string{"value1", "value2"},
			expectError:    false,
		},
		{
			name:   "get mix of existing and non-existing fields",
			key:    "hash1",
			fields: []string{"field1", "field4", "field2"},
			setupHashMap: func() {
				resetHashMap()
				setupTestHash("hash1", map[string]string{
					"field1": "value1",
					"field2": "value2",
				})
			},
			expectedValues: []string{"value1", "(nil)", "value2"},
			expectError:    false,
		},
		{
			name:   "get from non-existing hash",
			key:    "nonexistent",
			fields: []string{"field1", "field2"},
			setupHashMap: func() {
				resetHashMap()
			},
			expectedValues: []string{"(nil)", "(nil)"},
			expectError:    false,
		},
		{
			name:   "empty key should fail validation",
			key:    "",
			fields: []string{"field1"},
			setupHashMap: func() {
				resetHashMap()
			},
			expectedValues: []string{},
			expectError:    true,
		},
		{
			name:   "empty fields should fail validation",
			key:    "hash1",
			fields: []string{},
			setupHashMap: func() {
				resetHashMap()
			},
			expectedValues: []string{},
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupHashMap()
			mockEM := NewMockEventManager()
			event := createHmgetCommandEvent(tt.key, tt.fields)

			Hmget(&mockEM.EventManager, &event)

			response, ok := waitForHmgetResponse(event.Reply, 100*time.Millisecond)
			require.True(t, ok, "Should receive response within timeout")
			assert.Equal(t, tt.expectedValues, response.Values, "Values should match expected")

			if tt.expectError {
				assert.NotEmpty(t, response.Error, "Should have error message")
			} else {
				assert.Empty(t, response.Error, "Should not have error message")
			}
		})
	}
}
