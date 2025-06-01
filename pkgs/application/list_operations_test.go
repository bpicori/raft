package application

import (
	"bpicori/raft/pkgs/dto"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplicateRemoveCommand(t *testing.T) {
	tests := []struct {
		name          string
		removeCommand *dto.RemoveCommand
		setupHashMap  func()
		shouldExist   bool
	}{
		{
			name: "remove existing key",
			removeCommand: &dto.RemoveCommand{
				Key: "existingkey",
			},
			setupHashMap: func() {
				resetHashMap()
				setHashMapValue("existingkey", "somevalue")
			},
			shouldExist: false,
		},
		{
			name: "remove non-existing key",
			removeCommand: &dto.RemoveCommand{
				Key: "nonexistentkey",
			},
			setupHashMap: func() {
				resetHashMap()
			},
			shouldExist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupHashMap()

			replicateRemoveCommand(tt.removeCommand)

			exists := hashMapHasKey(tt.removeCommand.Key)
			assert.Equal(t, tt.shouldExist, exists, "Key existence should match expected")
		})
	}
}

func TestReplicateLpushCommand(t *testing.T) {
	tests := []struct {
		name         string
		lpushCommand *dto.LpushCommand
		setupHashMap func()
		expectedList []string
	}{
		{
			name: "push to new list",
			lpushCommand: &dto.LpushCommand{
				Key:      "newlist",
				Elements: []string{"item1", "item2"},
			},
			setupHashMap: func() {
				resetHashMap()
			},
			expectedList: []string{"item1", "item2"},
		},
		{
			name: "push to existing list",
			lpushCommand: &dto.LpushCommand{
				Key:      "existinglist",
				Elements: []string{"new1", "new2"},
			},
			setupHashMap: func() {
				resetHashMap()
				existingList := []string{"old1", "old2"}
				jsonData, _ := json.Marshal(existingList)
				setHashMapValue("existinglist", string(jsonData))
			},
			expectedList: []string{"new1", "new2", "old1", "old2"},
		},
		{
			name: "push single element",
			lpushCommand: &dto.LpushCommand{
				Key:      "singlelist",
				Elements: []string{"single"},
			},
			setupHashMap: func() {
				resetHashMap()
			},
			expectedList: []string{"single"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupHashMap()

			replicateLpushCommand(tt.lpushCommand)

			value, exists := getHashMapValue(tt.lpushCommand.Key)
			assert.True(t, exists, "Key should exist after LPUSH")

			var actualList []string
			err := json.Unmarshal([]byte(value), &actualList)
			assert.NoError(t, err, "Should be able to unmarshal list")
			assert.Equal(t, tt.expectedList, actualList, "List should match expected")
		})
	}
}

func TestReplicateLpopCommand(t *testing.T) {
	tests := []struct {
		name         string
		lpopCommand  *dto.LpopCommand
		setupHashMap func()
		expectedList []string
		shouldExist  bool
	}{
		{
			name: "pop from list with multiple elements",
			lpopCommand: &dto.LpopCommand{
				Key: "multilist",
			},
			setupHashMap: func() {
				resetHashMap()
				list := []string{"first", "second", "third"}
				jsonData, _ := json.Marshal(list)
				setHashMapValue("multilist", string(jsonData))
			},
			expectedList: []string{"second", "third"},
			shouldExist:  true,
		},
		{
			name: "pop from list with single element",
			lpopCommand: &dto.LpopCommand{
				Key: "singlelist",
			},
			setupHashMap: func() {
				resetHashMap()
				list := []string{"onlyone"}
				jsonData, _ := json.Marshal(list)
				setHashMapValue("singlelist", string(jsonData))
			},
			expectedList: []string{},
			shouldExist:  false,
		},
		{
			name: "pop from empty list",
			lpopCommand: &dto.LpopCommand{
				Key: "emptylist",
			},
			setupHashMap: func() {
				resetHashMap()
				list := []string{}
				jsonData, _ := json.Marshal(list)
				setHashMapValue("emptylist", string(jsonData))
			},
			expectedList: []string{},
			shouldExist:  true, // Empty list still exists as empty JSON array
		},
		{
			name: "pop from non-existing key",
			lpopCommand: &dto.LpopCommand{
				Key: "nonexistent",
			},
			setupHashMap: func() {
				resetHashMap()
			},
			expectedList: []string{},
			shouldExist:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupHashMap()

			replicateLpopCommand(tt.lpopCommand)

			exists := hashMapHasKey(tt.lpopCommand.Key)
			assert.Equal(t, tt.shouldExist, exists, "Key existence should match expected")

			if tt.shouldExist && len(tt.expectedList) > 0 {
				value, _ := getHashMapValue(tt.lpopCommand.Key)
				var actualList []string
				err := json.Unmarshal([]byte(value), &actualList)
				assert.NoError(t, err, "Should be able to unmarshal list")
				assert.Equal(t, tt.expectedList, actualList, "List should match expected")
			}
		})
	}
}

func TestLoadAndConvertToInt32(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		setupHashMap func()
		expectedVal  int32
		expectError  bool
	}{
		{
			name: "existing numeric value",
			key:  "numkey",
			setupHashMap: func() {
				resetHashMap()
				setHashMapValue("numkey", "42")
			},
			expectedVal: 42,
			expectError: false,
		},
		{
			name: "non-existing key",
			key:  "nonexistent",
			setupHashMap: func() {
				resetHashMap()
			},
			expectedVal: 0,
			expectError: false,
		},
		{
			name: "zero value",
			key:  "zerokey",
			setupHashMap: func() {
				resetHashMap()
				setHashMapValue("zerokey", "0")
			},
			expectedVal: 0,
			expectError: false,
		},
		{
			name: "negative value",
			key:  "negkey",
			setupHashMap: func() {
				resetHashMap()
				setHashMapValue("negkey", "-10")
			},
			expectedVal: -10,
			expectError: false,
		},
		{
			name: "invalid numeric value",
			key:  "invalidkey",
			setupHashMap: func() {
				resetHashMap()
				setHashMapValue("invalidkey", "notanumber")
			},
			expectedVal: 0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupHashMap()

			result, err := loadAndConvertToInt32(tt.key)

			if tt.expectError {
				assert.Error(t, err, "Should return error for invalid numeric value")
			} else {
				assert.NoError(t, err, "Should not return error")
				assert.Equal(t, tt.expectedVal, result, "Value should match expected")
			}
		})
	}
}

func TestLpop(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		setupHashMap func()
		mockSuccess  bool
		expectedElem string
		expectError  bool
	}{
		{
			name: "pop from list with elements",
			key:  "testlist",
			setupHashMap: func() {
				resetHashMap()
				list := []string{"first", "second", "third"}
				jsonData, _ := json.Marshal(list)
				setHashMapValue("testlist", string(jsonData))
			},
			mockSuccess:  true,
			expectedElem: "first",
			expectError:  false,
		},
		{
			name: "pop from empty list",
			key:  "emptylist",
			setupHashMap: func() {
				resetHashMap()
				list := []string{}
				jsonData, _ := json.Marshal(list)
				setHashMapValue("emptylist", string(jsonData))
			},
			mockSuccess:  true,
			expectedElem: "",
			expectError:  false,
		},
		{
			name: "pop from non-existing key",
			key:  "nonexistent",
			setupHashMap: func() {
				resetHashMap()
			},
			mockSuccess:  true,
			expectedElem: "",
			expectError:  false,
		},
		{
			name: "pop with timeout",
			key:  "timeoutlist",
			setupHashMap: func() {
				resetHashMap()
				list := []string{"item"}
				jsonData, _ := json.Marshal(list)
				setHashMapValue("timeoutlist", string(jsonData))
			},
			mockSuccess:  false,
			expectedElem: "",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupHashMap()
			mockEM := NewMockEventManager()
			ctx, cancel := createTestContext(200 * time.Millisecond)
			defer cancel()

			mockEM.StartMockAppendLogEntryHandler(ctx)

			event := createLpopCommandEvent(tt.key)

			if !tt.mockSuccess {
				mockEM.SetSimulateTimeout(true)
			}

			Lpop(&mockEM.EventManager, &event)

			response, ok := waitForResponse(event.Reply, 300*time.Millisecond)
			require.True(t, ok, "Should receive response")
			assert.Equal(t, tt.expectedElem, response.Element, "Element should match expected")

			if tt.expectError {
				assert.NotEmpty(t, response.Error, "Should have error message")
			} else if tt.mockSuccess {
				assert.Empty(t, response.Error, "Should not have error message")
			}
		})
	}
}

func TestLindex(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		index        int32
		setupHashMap func()
		expectedElem string
		expectError  bool
	}{
		{
			name:  "get element at valid positive index",
			key:   "testlist",
			index: 1,
			setupHashMap: func() {
				resetHashMap()
				list := []string{"first", "second", "third"}
				jsonData, _ := json.Marshal(list)
				setHashMapValue("testlist", string(jsonData))
			},
			expectedElem: "second",
			expectError:  false,
		},
		{
			name:  "get element at valid negative index",
			key:   "testlist",
			index: -1,
			setupHashMap: func() {
				resetHashMap()
				list := []string{"first", "second", "third"}
				jsonData, _ := json.Marshal(list)
				setHashMapValue("testlist", string(jsonData))
			},
			expectedElem: "third",
			expectError:  false,
		},
		{
			name:  "get element at out of bounds index",
			key:   "testlist",
			index: 10,
			setupHashMap: func() {
				resetHashMap()
				list := []string{"first", "second"}
				jsonData, _ := json.Marshal(list)
				setHashMapValue("testlist", string(jsonData))
			},
			expectedElem: "(nil)",
			expectError:  false,
		},
		{
			name:  "get element from empty list",
			key:   "emptylist",
			index: 0,
			setupHashMap: func() {
				resetHashMap()
				list := []string{}
				jsonData, _ := json.Marshal(list)
				setHashMapValue("emptylist", string(jsonData))
			},
			expectedElem: "(nil)",
			expectError:  false,
		},
		{
			name:  "get element from non-existing key",
			key:   "nonexistent",
			index: 0,
			setupHashMap: func() {
				resetHashMap()
			},
			expectedElem: "(nil)",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupHashMap()
			mockEM := NewMockEventManager()
			event := createLindexCommandEvent(tt.key, tt.index)

			Lindex(&mockEM.EventManager, &event)

			response, ok := waitForResponse(event.Reply, 100*time.Millisecond)
			require.True(t, ok, "Should receive response")
			assert.Equal(t, tt.expectedElem, response.Element, "Element should match expected")

			if tt.expectError {
				assert.NotEmpty(t, response.Error, "Should have error message")
			} else {
				assert.Empty(t, response.Error, "Should not have error message")
			}
		})
	}
}

func TestLpush(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		elements     []string
		setupHashMap func()
		mockSuccess  bool
		expectedLen  int32
		expectError  bool
	}{
		{
			name:     "push to new list",
			key:      "newlist",
			elements: []string{"item1", "item2"},
			setupHashMap: func() {
				resetHashMap()
			},
			mockSuccess: true,
			expectedLen: 2,
			expectError: false,
		},
		{
			name:     "push to existing list",
			key:      "existinglist",
			elements: []string{"new1"},
			setupHashMap: func() {
				resetHashMap()
				existingList := []string{"old1", "old2"}
				jsonData, _ := json.Marshal(existingList)
				setHashMapValue("existinglist", string(jsonData))
			},
			mockSuccess: true,
			expectedLen: 3,
			expectError: false,
		},
		{
			name:     "push with timeout",
			key:      "timeoutlist",
			elements: []string{"item"},
			setupHashMap: func() {
				resetHashMap()
			},
			mockSuccess: false,
			expectedLen: 0,
			expectError: true,
		},
		{
			name:     "push empty elements",
			key:      "emptylist",
			elements: []string{},
			setupHashMap: func() {
				resetHashMap()
			},
			mockSuccess: true,
			expectedLen: 0,
			expectError: true, // Should fail validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupHashMap()
			mockEM := NewMockEventManager()
			ctx, cancel := createTestContext(200 * time.Millisecond)
			defer cancel()

			mockEM.StartMockAppendLogEntryHandler(ctx)

			event := createLpushCommandEvent(tt.key, tt.elements)

			if !tt.mockSuccess {
				mockEM.SetSimulateTimeout(true)
			}

			Lpush(&mockEM.EventManager, &event)

			response, ok := waitForResponse(event.Reply, 300*time.Millisecond)
			require.True(t, ok, "Should receive response")
			assert.Equal(t, tt.expectedLen, response.Length, "Length should match expected")

			if tt.expectError {
				assert.NotEmpty(t, response.Error, "Should have error message")
			} else if tt.mockSuccess {
				assert.Empty(t, response.Error, "Should not have error message")
			}
		})
	}
}
