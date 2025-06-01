package application

import (
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {
	// Enable test mode for faster timeouts
	SetTestMode(true)
}

// Test helper to reset global hashMap state
func resetHashMap() {
	hashMap = sync.Map{}
}

// Test helper to set a value in hashMap
func setHashMapValue(key, value string) {
	hashMap.Store(key, value)
}

// Test helper to get a value from hashMap
func getHashMapValue(key string) (string, bool) {
	value, ok := hashMap.Load(key)
	if !ok {
		return "", false
	}
	return value.(string), true
}

// Test helper to check if key exists in hashMap
func hashMapHasKey(key string) bool {
	_, ok := hashMap.Load(key)
	return ok
}

func TestReplicateLogEntries(t *testing.T) {
	tests := []struct {
		name         string
		logEntries   []*dto.LogEntry
		commitLength int32
		setupHashMap func()
		expectedKeys []string
		expectedVals []string
	}{
		{
			name:         "empty log entries",
			logEntries:   []*dto.LogEntry{},
			commitLength: 0,
			setupHashMap: func() { resetHashMap() },
			expectedKeys: []string{},
			expectedVals: []string{},
		},
		{
			name: "single SET command",
			logEntries: []*dto.LogEntry{
				{
					Term: 1,
					Uuid: "test-uuid-1",
					Command: &dto.Command{
						Operation: consts.SetOp.String(),
						Args: &dto.Command_SetCommand{
							SetCommand: &dto.SetCommand{
								Key:   "key1",
								Value: "value1",
							},
						},
					},
				},
			},
			commitLength: 1,
			setupHashMap: func() { resetHashMap() },
			expectedKeys: []string{"key1"},
			expectedVals: []string{"value1"},
		},
		{
			name: "multiple commands",
			logEntries: []*dto.LogEntry{
				{
					Term: 1,
					Uuid: "test-uuid-1",
					Command: &dto.Command{
						Operation: consts.SetOp.String(),
						Args: &dto.Command_SetCommand{
							SetCommand: &dto.SetCommand{
								Key:   "key1",
								Value: "value1",
							},
						},
					},
				},
				{
					Term: 1,
					Uuid: "test-uuid-2",
					Command: &dto.Command{
						Operation: consts.IncrementOp.String(),
						Args: &dto.Command_IncrCommand{
							IncrCommand: &dto.IncrCommand{
								Key: "counter",
							},
						},
					},
				},
			},
			commitLength: 2,
			setupHashMap: func() {
				resetHashMap()
				setHashMapValue("counter", "5")
			},
			expectedKeys: []string{"key1", "counter"},
			expectedVals: []string{"value1", "6"},
		},
		{
			name: "commit length less than log entries",
			logEntries: []*dto.LogEntry{
				{
					Term: 1,
					Uuid: "test-uuid-1",
					Command: &dto.Command{
						Operation: consts.SetOp.String(),
						Args: &dto.Command_SetCommand{
							SetCommand: &dto.SetCommand{
								Key:   "key1",
								Value: "value1",
							},
						},
					},
				},
				{
					Term: 1,
					Uuid: "test-uuid-2",
					Command: &dto.Command{
						Operation: consts.SetOp.String(),
						Args: &dto.Command_SetCommand{
							SetCommand: &dto.SetCommand{
								Key:   "key2",
								Value: "value2",
							},
						},
					},
				},
			},
			commitLength: 1,
			setupHashMap: func() { resetHashMap() },
			expectedKeys: []string{"key1"},
			expectedVals: []string{"value1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupHashMap()

			replicateLogEntries(tt.logEntries, tt.commitLength)

			for i, key := range tt.expectedKeys {
				value, exists := getHashMapValue(key)
				assert.True(t, exists, "Key %s should exist in hashMap", key)
				assert.Equal(t, tt.expectedVals[i], value, "Value for key %s should match", key)
			}
		})
	}
}

func TestReplicateLogEntry(t *testing.T) {
	tests := []struct {
		name         string
		logEntry     *dto.LogEntry
		setupHashMap func()
		expectedKey  string
		expectedVal  string
		shouldExist  bool
	}{
		{
			name: "SET command",
			logEntry: &dto.LogEntry{
				Term: 1,
				Uuid: "test-uuid",
				Command: &dto.Command{
					Operation: consts.SetOp.String(),
					Args: &dto.Command_SetCommand{
						SetCommand: &dto.SetCommand{
							Key:   "testkey",
							Value: "testvalue",
						},
					},
				},
			},
			setupHashMap: func() { resetHashMap() },
			expectedKey:  "testkey",
			expectedVal:  "testvalue",
			shouldExist:  true,
		},
		{
			name: "INCREMENT command",
			logEntry: &dto.LogEntry{
				Term: 1,
				Uuid: "test-uuid",
				Command: &dto.Command{
					Operation: consts.IncrementOp.String(),
					Args: &dto.Command_IncrCommand{
						IncrCommand: &dto.IncrCommand{
							Key: "counter",
						},
					},
				},
			},
			setupHashMap: func() {
				resetHashMap()
				setHashMapValue("counter", "10")
			},
			expectedKey: "counter",
			expectedVal: "11",
			shouldExist: true,
		},
		{
			name: "DECREMENT command",
			logEntry: &dto.LogEntry{
				Term: 1,
				Uuid: "test-uuid",
				Command: &dto.Command{
					Operation: consts.DecrementOp.String(),
					Args: &dto.Command_DecrCommand{
						DecrCommand: &dto.DecrCommand{
							Key: "counter",
						},
					},
				},
			},
			setupHashMap: func() {
				resetHashMap()
				setHashMapValue("counter", "5")
			},
			expectedKey: "counter",
			expectedVal: "4",
			shouldExist: true,
		},
		{
			name: "DELETE command",
			logEntry: &dto.LogEntry{
				Term: 1,
				Uuid: "test-uuid",
				Command: &dto.Command{
					Operation: consts.DeleteOp.String(),
					Args: &dto.Command_RemoveCommand{
						RemoveCommand: &dto.RemoveCommand{
							Key: "keyToDelete",
						},
					},
				},
			},
			setupHashMap: func() {
				resetHashMap()
				setHashMapValue("keyToDelete", "somevalue")
			},
			expectedKey: "keyToDelete",
			expectedVal: "",
			shouldExist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupHashMap()

			replicateLogEntry(tt.logEntry)

			if tt.shouldExist {
				value, exists := getHashMapValue(tt.expectedKey)
				assert.True(t, exists, "Key %s should exist in hashMap", tt.expectedKey)
				assert.Equal(t, tt.expectedVal, value, "Value for key %s should match", tt.expectedKey)
			} else {
				exists := hashMapHasKey(tt.expectedKey)
				assert.False(t, exists, "Key %s should not exist in hashMap", tt.expectedKey)
			}
		})
	}
}

func TestReplicateSetCommand(t *testing.T) {
	tests := []struct {
		name        string
		setCommand  *dto.SetCommand
		expectedKey string
		expectedVal string
	}{
		{
			name: "basic set",
			setCommand: &dto.SetCommand{
				Key:   "key1",
				Value: "value1",
			},
			expectedKey: "key1",
			expectedVal: "value1",
		},
		{
			name: "overwrite existing key",
			setCommand: &dto.SetCommand{
				Key:   "existing",
				Value: "newvalue",
			},
			expectedKey: "existing",
			expectedVal: "newvalue",
		},
		{
			name: "empty value",
			setCommand: &dto.SetCommand{
				Key:   "emptykey",
				Value: "",
			},
			expectedKey: "emptykey",
			expectedVal: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetHashMap()
			if tt.name == "overwrite existing key" {
				setHashMapValue("existing", "oldvalue")
			}

			replicateSetCommand(tt.setCommand)

			value, exists := getHashMapValue(tt.expectedKey)
			assert.True(t, exists, "Key should exist after SET")
			assert.Equal(t, tt.expectedVal, value, "Value should match")
		})
	}
}

func TestReplicateIncrCommand(t *testing.T) {
	tests := []struct {
		name         string
		incrCommand  *dto.IncrCommand
		setupHashMap func()
		expectedVal  string
	}{
		{
			name: "increment existing numeric value",
			incrCommand: &dto.IncrCommand{
				Key: "counter",
			},
			setupHashMap: func() {
				resetHashMap()
				setHashMapValue("counter", "5")
			},
			expectedVal: "6",
		},
		{
			name: "increment non-existing key",
			incrCommand: &dto.IncrCommand{
				Key: "newcounter",
			},
			setupHashMap: func() {
				resetHashMap()
			},
			expectedVal: "1",
		},
		{
			name: "increment zero value",
			incrCommand: &dto.IncrCommand{
				Key: "zerocounter",
			},
			setupHashMap: func() {
				resetHashMap()
				setHashMapValue("zerocounter", "0")
			},
			expectedVal: "1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupHashMap()

			replicateIncrCommand(tt.incrCommand)

			value, exists := getHashMapValue(tt.incrCommand.Key)
			assert.True(t, exists, "Key should exist after INCR")
			assert.Equal(t, tt.expectedVal, value, "Value should be incremented")
		})
	}
}

func TestReplicateDecrCommand(t *testing.T) {
	tests := []struct {
		name         string
		decrCommand  *dto.DecrCommand
		setupHashMap func()
		expectedVal  string
	}{
		{
			name: "decrement existing numeric value",
			decrCommand: &dto.DecrCommand{
				Key: "counter",
			},
			setupHashMap: func() {
				resetHashMap()
				setHashMapValue("counter", "5")
			},
			expectedVal: "4",
		},
		{
			name: "decrement non-existing key",
			decrCommand: &dto.DecrCommand{
				Key: "newcounter",
			},
			setupHashMap: func() {
				resetHashMap()
			},
			expectedVal: "-1",
		},
		{
			name: "decrement zero value",
			decrCommand: &dto.DecrCommand{
				Key: "zerocounter",
			},
			setupHashMap: func() {
				resetHashMap()
				setHashMapValue("zerocounter", "0")
			},
			expectedVal: "-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupHashMap()

			replicateDecrCommand(tt.decrCommand)

			value, exists := getHashMapValue(tt.decrCommand.Key)
			assert.True(t, exists, "Key should exist after DECR")
			assert.Equal(t, tt.expectedVal, value, "Value should be decremented")
		})
	}
}
