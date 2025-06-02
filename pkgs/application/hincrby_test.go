package application

import (
	"bpicori/raft/pkgs/dto"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHincrby(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		field        string
		increment    int32
		setupHashMap func()
		expectedVal  int32
		expectError  bool
		mockSuccess  bool
	}{
		{
			name:      "increment new field in new hash",
			key:       "hash1",
			field:     "counter",
			increment: 5,
			setupHashMap: func() {
				resetHashMap()
			},
			expectedVal: 5,
			expectError: false,
			mockSuccess: true,
		},
		{
			name:      "increment existing integer field",
			key:       "hash2",
			field:     "counter",
			increment: 3,
			setupHashMap: func() {
				resetHashMap()
				setupTestHash("hash2", map[string]string{
					"counter": "10",
					"name":    "test",
				})
			},
			expectedVal: 13,
			expectError: false,
			mockSuccess: true,
		},
		{
			name:      "decrement with negative increment",
			key:       "hash3",
			field:     "counter",
			increment: -2,
			setupHashMap: func() {
				resetHashMap()
				setupTestHash("hash3", map[string]string{
					"counter": "5",
				})
			},
			expectedVal: 3,
			expectError: false,
			mockSuccess: true,
		},
		{
			name:      "increment non-integer field should fail",
			key:       "hash4",
			field:     "name",
			increment: 1,
			setupHashMap: func() {
				resetHashMap()
				setupTestHash("hash4", map[string]string{
					"name": "notanumber",
				})
			},
			expectedVal: 0,
			expectError: true,
			mockSuccess: false,
		},
		{
			name:      "increment zero value",
			key:       "hash5",
			field:     "counter",
			increment: 0,
			setupHashMap: func() {
				resetHashMap()
				setupTestHash("hash5", map[string]string{
					"counter": "42",
				})
			},
			expectedVal: 42,
			expectError: false,
			mockSuccess: true,
		},
		{
			name:      "increment large positive number",
			key:       "hash6",
			field:     "counter",
			increment: 1000,
			setupHashMap: func() {
				resetHashMap()
				setupTestHash("hash6", map[string]string{
					"counter": "2000",
				})
			},
			expectedVal: 3000,
			expectError: false,
			mockSuccess: true,
		},
		{
			name:      "increment large negative number",
			key:       "hash7",
			field:     "counter",
			increment: -500,
			setupHashMap: func() {
				resetHashMap()
				setupTestHash("hash7", map[string]string{
					"counter": "100",
				})
			},
			expectedVal: -400,
			expectError: false,
			mockSuccess: true,
		},
		{
			name:      "empty key should fail validation",
			key:       "",
			field:     "counter",
			increment: 1,
			setupHashMap: func() {
				resetHashMap()
			},
			expectedVal: 0,
			expectError: true,
			mockSuccess: false,
		},
		{
			name:      "empty field should fail validation",
			key:       "hash8",
			field:     "",
			increment: 1,
			setupHashMap: func() {
				resetHashMap()
			},
			expectedVal: 0,
			expectError: true,
			mockSuccess: false,
		},
		{
			name:      "timeout should return error",
			key:       "hash9",
			field:     "counter",
			increment: 1,
			setupHashMap: func() {
				resetHashMap()
			},
			expectedVal: 0,
			expectError: true,
			mockSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupHashMap()
			mockEM := NewMockEventManager()
			ctx, cancel := createTestContext(200 * time.Millisecond)
			defer cancel()

			mockEM.StartMockAppendLogEntryHandler(ctx)

			event := createHincrbyCommandEvent(tt.key, tt.field, tt.increment)

			// Set up mock response
			if !tt.mockSuccess {
				mockEM.SetSimulateTimeout(true)
			}

			Hincrby(&mockEM.EventManager, &event)

			response, ok := waitForHincrbyResponse(event.Reply, 300*time.Millisecond)
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

func TestReplicateHsetCommand(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		fields       map[string]string
		setupHashMap func()
		expectedHash map[string]string
	}{
		{
			name: "replicate to new hash",
			key:  "hash1",
			fields: map[string]string{
				"field1": "value1",
				"field2": "value2",
			},
			setupHashMap: func() {
				resetHashMap()
			},
			expectedHash: map[string]string{
				"field1": "value1",
				"field2": "value2",
			},
		},
		{
			name: "replicate to existing hash",
			key:  "hash2",
			fields: map[string]string{
				"field2": "newvalue2",
				"field3": "value3",
			},
			setupHashMap: func() {
				resetHashMap()
				setupTestHash("hash2", map[string]string{
					"field1": "value1",
					"field2": "oldvalue2",
				})
			},
			expectedHash: map[string]string{
				"field1": "value1",
				"field2": "newvalue2",
				"field3": "value3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupHashMap()

			hsetCommand := &dto.HsetCommand{
				Key:    tt.key,
				Fields: tt.fields,
			}

			replicateHsetCommand(hsetCommand)

			// Verify the hash was updated correctly
			value, exists := hashMap.Load(tt.key)
			require.True(t, exists, "Hash should exist")

			var actualHash map[string]string
			err := json.Unmarshal([]byte(value.(string)), &actualHash)
			require.NoError(t, err, "Should be able to unmarshal hash")

			assert.Equal(t, tt.expectedHash, actualHash, "Hash should match expected")
		})
	}
}

func TestReplicateHincrbyCommand(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		field        string
		increment    int32
		setupHashMap func()
		expectedVal  string
	}{
		{
			name:      "replicate increment to new hash",
			key:       "hash1",
			field:     "counter",
			increment: 5,
			setupHashMap: func() {
				resetHashMap()
			},
			expectedVal: "5",
		},
		{
			name:      "replicate increment to existing field",
			key:       "hash2",
			field:     "counter",
			increment: 3,
			setupHashMap: func() {
				resetHashMap()
				setupTestHash("hash2", map[string]string{
					"counter": "10",
					"name":    "test",
				})
			},
			expectedVal: "13",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupHashMap()

			hincrbyCommand := &dto.HincrbyCommand{
				Key:       tt.key,
				Field:     tt.field,
				Increment: tt.increment,
			}

			replicateHincrbyCommand(hincrbyCommand)

			// Verify the field was updated correctly
			value, exists := hashMap.Load(tt.key)
			require.True(t, exists, "Hash should exist")

			var actualHash map[string]string
			err := json.Unmarshal([]byte(value.(string)), &actualHash)
			require.NoError(t, err, "Should be able to unmarshal hash")

			assert.Equal(t, tt.expectedVal, actualHash[tt.field], "Field value should match expected")
		})
	}
}
