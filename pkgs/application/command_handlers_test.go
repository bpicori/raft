package application

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		setupHashMap func()
		expectedVal  string
	}{
		{
			name: "get existing key",
			key:  "existingkey",
			setupHashMap: func() {
				resetHashMap()
				setHashMapValue("existingkey", "testvalue")
			},
			expectedVal: "testvalue",
		},
		{
			name: "get non-existing key",
			key:  "nonexistent",
			setupHashMap: func() {
				resetHashMap()
			},
			expectedVal: "",
		},
		{
			name: "get empty value",
			key:  "emptykey",
			setupHashMap: func() {
				resetHashMap()
				setHashMapValue("emptykey", "")
			},
			expectedVal: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupHashMap()
			mockEM := NewMockEventManager()
			event := createGetCommandEvent(tt.key)

			Get(&mockEM.EventManager, &event)

			response, ok := waitForResponse(event.Reply, 100*time.Millisecond)
			require.True(t, ok, "Should receive response")
			assert.Equal(t, tt.expectedVal, response.Value, "Value should match expected")
		})
	}
}

func TestSet(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		value       string
		mockSuccess bool
		expectedOk  bool
	}{
		{
			name:        "successful set",
			key:         "testkey",
			value:       "testvalue",
			mockSuccess: true,
			expectedOk:  true,
		},
		{
			name:        "failed set (timeout)",
			key:         "testkey",
			value:       "testvalue",
			mockSuccess: false,
			expectedOk:  false,
		},
		{
			name:        "set empty value (validation error)",
			key:         "emptykey",
			value:       "",
			mockSuccess: true,
			expectedOk:  false, // Should fail validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetHashMap()
			mockEM := NewMockEventManager()
			ctx, cancel := createTestContext(5 * time.Second)
			defer cancel()

			mockEM.StartMockAppendLogEntryHandler(ctx)

			event := createSetCommandEvent(tt.key, tt.value)

			// Set up mock response
			if !tt.mockSuccess {
				// Simulate timeout by not sending any response
				mockEM.SetSimulateTimeout(true)
			}

			Set(&mockEM.EventManager, &event)

			response, ok := waitForResponse(event.Reply, 6*time.Second)
			require.True(t, ok, "Should receive response")
			assert.Equal(t, tt.expectedOk, response.Ok, "Response Ok should match expected")

			if tt.expectedOk {
				// Check that value was stored in hashMap
				value, exists := getHashMapValue(tt.key)
				assert.True(t, exists, "Key should exist in hashMap")
				assert.Equal(t, tt.value, value, "Value should match")
			}
		})
	}
}

func TestIncr(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		setupHashMap func()
		mockSuccess  bool
		expectedVal  int32
		expectError  bool
	}{
		{
			name: "increment existing numeric value",
			key:  "counter",
			setupHashMap: func() {
				resetHashMap()
				setHashMapValue("counter", "5")
			},
			mockSuccess: true,
			expectedVal: 6,
			expectError: false,
		},
		{
			name: "increment non-existing key",
			key:  "newcounter",
			setupHashMap: func() {
				resetHashMap()
			},
			mockSuccess: true,
			expectedVal: 1,
			expectError: false,
		},
		{
			name: "increment with timeout",
			key:  "timeoutkey",
			setupHashMap: func() {
				resetHashMap()
				setHashMapValue("timeoutkey", "10")
			},
			mockSuccess: false,
			expectedVal: 0,
			expectError: false, // Timeout doesn't return error, just zero value
		},
		{
			name: "increment invalid value",
			key:  "invalidkey",
			setupHashMap: func() {
				resetHashMap()
				setHashMapValue("invalidkey", "notanumber")
			},
			mockSuccess: true,
			expectedVal: 0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupHashMap()
			mockEM := NewMockEventManager()
			ctx, cancel := createTestContext(5 * time.Second)
			defer cancel()

			mockEM.StartMockAppendLogEntryHandler(ctx)

			event := createIncrCommandEvent(tt.key)

			// Set up mock response
			if !tt.mockSuccess {
				mockEM.SetSimulateTimeout(true)
			}

			Incr(&mockEM.EventManager, &event)

			response, ok := waitForResponse(event.Reply, 6*time.Second)
			require.True(t, ok, "Should receive response")
			assert.Equal(t, tt.expectedVal, response.Value, "Value should match expected")

			if tt.expectError {
				assert.NotEmpty(t, response.Error, "Should have error message")
			} else if tt.mockSuccess {
				assert.Empty(t, response.Error, "Should not have error message")
			}
		})
	}
}

func TestDecr(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		setupHashMap func()
		mockSuccess  bool
		expectedVal  int32
		expectError  bool
	}{
		{
			name: "decrement existing numeric value",
			key:  "counter",
			setupHashMap: func() {
				resetHashMap()
				setHashMapValue("counter", "5")
			},
			mockSuccess: true,
			expectedVal: 4,
			expectError: false,
		},
		{
			name: "decrement non-existing key",
			key:  "newcounter",
			setupHashMap: func() {
				resetHashMap()
			},
			mockSuccess: true,
			expectedVal: -1,
			expectError: false,
		},
		{
			name: "decrement with timeout",
			key:  "timeoutkey",
			setupHashMap: func() {
				resetHashMap()
				setHashMapValue("timeoutkey", "10")
			},
			mockSuccess: false,
			expectedVal: 0,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupHashMap()
			mockEM := NewMockEventManager()
			ctx, cancel := createTestContext(5 * time.Second)
			defer cancel()

			mockEM.StartMockAppendLogEntryHandler(ctx)

			event := createDecrCommandEvent(tt.key)

			if !tt.mockSuccess {
				mockEM.SetSimulateTimeout(true)
			}

			Decr(&mockEM.EventManager, &event)

			response, ok := waitForResponse(event.Reply, 6*time.Second)
			require.True(t, ok, "Should receive response")
			assert.Equal(t, tt.expectedVal, response.Value, "Value should match expected")

			if tt.expectError {
				assert.NotEmpty(t, response.Error, "Should have error message")
			} else if tt.mockSuccess {
				assert.Empty(t, response.Error, "Should not have error message")
			}
		})
	}
}

func TestRemove(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		setupHashMap func()
		mockSuccess  bool
		expectedOk   bool
	}{
		{
			name: "remove existing key",
			key:  "existingkey",
			setupHashMap: func() {
				resetHashMap()
				setHashMapValue("existingkey", "somevalue")
			},
			mockSuccess: true,
			expectedOk:  true,
		},
		{
			name: "remove non-existing key",
			key:  "nonexistent",
			setupHashMap: func() {
				resetHashMap()
			},
			mockSuccess: true,
			expectedOk:  true,
		},
		{
			name: "remove with timeout",
			key:  "timeoutkey",
			setupHashMap: func() {
				resetHashMap()
				setHashMapValue("timeoutkey", "value")
			},
			mockSuccess: false,
			expectedOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupHashMap()
			mockEM := NewMockEventManager()
			ctx, cancel := createTestContext(5 * time.Second)
			defer cancel()

			mockEM.StartMockAppendLogEntryHandler(ctx)

			event := createRemoveCommandEvent(tt.key)

			if !tt.mockSuccess {
				mockEM.SetSimulateTimeout(true)
			}

			Remove(&mockEM.EventManager, &event)

			response, ok := waitForResponse(event.Reply, 6*time.Second)
			require.True(t, ok, "Should receive response")
			assert.Equal(t, tt.expectedOk, response.Ok, "Response Ok should match expected")
		})
	}
}

