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

func TestSadd(t *testing.T) {
	tests := []struct {
		name          string
		key           string
		members       []string
		setupHashMap  func()
		mockSuccess   bool
		expectedAdded int32
		expectError   bool
	}{
		{
			name:    "add members to new set",
			key:     "newset",
			members: []string{"member1", "member2", "member3"},
			setupHashMap: func() {
				resetHashMap()
			},
			mockSuccess:   true,
			expectedAdded: 3,
			expectError:   false,
		},
		{
			name:    "add members to existing set",
			key:     "existingset",
			members: []string{"member3", "member4"},
			setupHashMap: func() {
				resetHashMap()
				existingSet := []string{"member1", "member2"}
				jsonData, _ := json.Marshal(existingSet)
				setHashMapValue("existingset", string(jsonData))
			},
			mockSuccess:   true,
			expectedAdded: 2,
			expectError:   false,
		},
		{
			name:    "add duplicate members",
			key:     "dupset",
			members: []string{"member1", "member2", "member1"},
			setupHashMap: func() {
				resetHashMap()
				existingSet := []string{"member1"}
				jsonData, _ := json.Marshal(existingSet)
				setHashMapValue("dupset", string(jsonData))
			},
			mockSuccess:   true,
			expectedAdded: 1, // Only member2 is new
			expectError:   false,
		},
		{
			name:    "add with timeout",
			key:     "timeoutset",
			members: []string{"member1"},
			setupHashMap: func() {
				resetHashMap()
			},
			mockSuccess:   false,
			expectedAdded: 0,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupHashMap()
			mockEM := NewMockEventManager()
			ctx, cancel := createTestContext(5 * time.Second)
			defer cancel()

			mockEM.StartMockAppendLogEntryHandler(ctx)

			event := createSaddCommandEvent(tt.key, tt.members)

			if !tt.mockSuccess {
				mockEM.SetSimulateTimeout(true)
			}

			Sadd(&mockEM.EventManager, &event)

			response, ok := waitForSaddResponse(event.Reply, 6*time.Second)
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

func TestSrem(t *testing.T) {
	tests := []struct {
		name            string
		key             string
		members         []string
		setupHashMap    func()
		mockSuccess     bool
		expectedRemoved int32
		expectError     bool
	}{
		{
			name:    "remove members from existing set",
			key:     "existingset",
			members: []string{"member1", "member3"},
			setupHashMap: func() {
				resetHashMap()
				existingSet := []string{"member1", "member2", "member3"}
				jsonData, _ := json.Marshal(existingSet)
				setHashMapValue("existingset", string(jsonData))
			},
			mockSuccess:     true,
			expectedRemoved: 2,
			expectError:     false,
		},
		{
			name:    "remove non-existing members",
			key:     "existingset",
			members: []string{"member4", "member5"},
			setupHashMap: func() {
				resetHashMap()
				existingSet := []string{"member1", "member2"}
				jsonData, _ := json.Marshal(existingSet)
				setHashMapValue("existingset", string(jsonData))
			},
			mockSuccess:     true,
			expectedRemoved: 0,
			expectError:     false,
		},
		{
			name:    "remove from non-existing set",
			key:     "nonexistentset",
			members: []string{"member1"},
			setupHashMap: func() {
				resetHashMap()
			},
			mockSuccess:     true,
			expectedRemoved: 0,
			expectError:     false,
		},
		{
			name:    "remove with timeout",
			key:     "timeoutset",
			members: []string{"member1"},
			setupHashMap: func() {
				resetHashMap()
			},
			mockSuccess:     false,
			expectedRemoved: 0,
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupHashMap()
			mockEM := NewMockEventManager()
			ctx, cancel := createTestContext(5 * time.Second)
			defer cancel()

			mockEM.StartMockAppendLogEntryHandler(ctx)

			event := createSremCommandEvent(tt.key, tt.members)

			if !tt.mockSuccess {
				mockEM.SetSimulateTimeout(true)
			}

			Srem(&mockEM.EventManager, &event)

			response, ok := waitForSremResponse(event.Reply, 6*time.Second)
			require.True(t, ok, "Should receive response")
			assert.Equal(t, tt.expectedRemoved, response.Removed, "Removed count should match expected")

			if tt.expectError {
				assert.NotEmpty(t, response.Error, "Should have error message")
			} else if tt.mockSuccess {
				assert.Empty(t, response.Error, "Should not have error message")
			}
		})
	}
}

func TestSismember(t *testing.T) {
	tests := []struct {
		name           string
		key            string
		member         string
		setupHashMap   func()
		expectedResult int32
	}{
		{
			name:   "member exists in set",
			key:    "existingset",
			member: "member2",
			setupHashMap: func() {
				resetHashMap()
				existingSet := []string{"member1", "member2", "member3"}
				jsonData, _ := json.Marshal(existingSet)
				setHashMapValue("existingset", string(jsonData))
			},
			expectedResult: 1,
		},
		{
			name:   "member does not exist in set",
			key:    "existingset",
			member: "member4",
			setupHashMap: func() {
				resetHashMap()
				existingSet := []string{"member1", "member2", "member3"}
				jsonData, _ := json.Marshal(existingSet)
				setHashMapValue("existingset", string(jsonData))
			},
			expectedResult: 0,
		},
		{
			name:   "key does not exist",
			key:    "nonexistentset",
			member: "member1",
			setupHashMap: func() {
				resetHashMap()
			},
			expectedResult: 0,
		},
		{
			name:   "empty set",
			key:    "emptyset",
			member: "member1",
			setupHashMap: func() {
				resetHashMap()
				emptySet := []string{}
				jsonData, _ := json.Marshal(emptySet)
				setHashMapValue("emptyset", string(jsonData))
			},
			expectedResult: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupHashMap()
			mockEM := NewMockEventManager()
			event := createSismemberCommandEvent(tt.key, tt.member)

			Sismember(&mockEM.EventManager, &event)

			response, ok := waitForSismemberResponse(event.Reply, 100*time.Millisecond)
			require.True(t, ok, "Should receive response")
			assert.Equal(t, tt.expectedResult, response.IsMember, "IsMember should match expected")
			assert.Empty(t, response.Error, "Should not have error")
		})
	}
}

// Helper functions for creating test events
func createSaddCommandEvent(key string, members []string) events.SaddCommandEvent {
	return events.SaddCommandEvent{
		Payload: &dto.SaddCommandRequest{
			Key:     key,
			Members: members,
		},
		Reply: make(chan *dto.SaddCommandResponse, 1),
	}
}

func createSremCommandEvent(key string, members []string) events.SremCommandEvent {
	return events.SremCommandEvent{
		Payload: &dto.SremCommandRequest{
			Key:     key,
			Members: members,
		},
		Reply: make(chan *dto.SremCommandResponse, 1),
	}
}

func createSismemberCommandEvent(key string, member string) events.SismemberCommandEvent {
	return events.SismemberCommandEvent{
		Payload: &dto.SismemberCommandRequest{
			Key:    key,
			Member: member,
		},
		Reply: make(chan *dto.SismemberCommandResponse, 1),
	}
}

// Helper functions for waiting for responses
func waitForSaddResponse(replyChan chan *dto.SaddCommandResponse, timeout time.Duration) (*dto.SaddCommandResponse, bool) {
	select {
	case response := <-replyChan:
		return response, true
	case <-time.After(timeout):
		return nil, false
	}
}

func waitForSremResponse(replyChan chan *dto.SremCommandResponse, timeout time.Duration) (*dto.SremCommandResponse, bool) {
	select {
	case response := <-replyChan:
		return response, true
	case <-time.After(timeout):
		return nil, false
	}
}

func waitForSismemberResponse(replyChan chan *dto.SismemberCommandResponse, timeout time.Duration) (*dto.SismemberCommandResponse, bool) {
	select {
	case response := <-replyChan:
		return response, true
	case <-time.After(timeout):
		return nil, false
	}
}

func TestSinter(t *testing.T) {
	tests := []struct {
		name            string
		keys            []string
		setupHashMap    func()
		expectedMembers []string
	}{
		{
			name: "intersection of two sets",
			keys: []string{"set1", "set2"},
			setupHashMap: func() {
				resetHashMap()
				set1 := []string{"member1", "member2", "member3"}
				set2 := []string{"member2", "member3", "member4"}
				jsonData1, _ := json.Marshal(set1)
				jsonData2, _ := json.Marshal(set2)
				setHashMapValue("set1", string(jsonData1))
				setHashMapValue("set2", string(jsonData2))
			},
			expectedMembers: []string{"member2", "member3"},
		},
		{
			name: "intersection with empty set",
			keys: []string{"set1", "emptyset"},
			setupHashMap: func() {
				resetHashMap()
				set1 := []string{"member1", "member2"}
				emptySet := []string{}
				jsonData1, _ := json.Marshal(set1)
				jsonData2, _ := json.Marshal(emptySet)
				setHashMapValue("set1", string(jsonData1))
				setHashMapValue("emptyset", string(jsonData2))
			},
			expectedMembers: []string{},
		},
		{
			name: "intersection with non-existing set",
			keys: []string{"set1", "nonexistent"},
			setupHashMap: func() {
				resetHashMap()
				set1 := []string{"member1", "member2"}
				jsonData1, _ := json.Marshal(set1)
				setHashMapValue("set1", string(jsonData1))
			},
			expectedMembers: []string{},
		},
		{
			name: "intersection of three sets",
			keys: []string{"set1", "set2", "set3"},
			setupHashMap: func() {
				resetHashMap()
				set1 := []string{"member1", "member2", "member3"}
				set2 := []string{"member2", "member3", "member4"}
				set3 := []string{"member2", "member5"}
				jsonData1, _ := json.Marshal(set1)
				jsonData2, _ := json.Marshal(set2)
				jsonData3, _ := json.Marshal(set3)
				setHashMapValue("set1", string(jsonData1))
				setHashMapValue("set2", string(jsonData2))
				setHashMapValue("set3", string(jsonData3))
			},
			expectedMembers: []string{"member2"},
		},
		{
			name: "no intersection",
			keys: []string{"set1", "set2"},
			setupHashMap: func() {
				resetHashMap()
				set1 := []string{"member1", "member2"}
				set2 := []string{"member3", "member4"}
				jsonData1, _ := json.Marshal(set1)
				jsonData2, _ := json.Marshal(set2)
				setHashMapValue("set1", string(jsonData1))
				setHashMapValue("set2", string(jsonData2))
			},
			expectedMembers: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupHashMap()
			mockEM := NewMockEventManager()
			event := createSinterCommandEvent(tt.keys)

			Sinter(&mockEM.EventManager, &event)

			response, ok := waitForSinterResponse(event.Reply, 100*time.Millisecond)
			require.True(t, ok, "Should receive response")
			assert.ElementsMatch(t, tt.expectedMembers, response.Members, "Members should match expected")
			assert.Empty(t, response.Error, "Should not have error")
		})
	}
}

func TestScard(t *testing.T) {
	tests := []struct {
		name                string
		key                 string
		setupHashMap        func()
		expectedCardinality int32
	}{
		{
			name: "cardinality of existing set",
			key:  "existingset",
			setupHashMap: func() {
				resetHashMap()
				existingSet := []string{"member1", "member2", "member3"}
				jsonData, _ := json.Marshal(existingSet)
				setHashMapValue("existingset", string(jsonData))
			},
			expectedCardinality: 3,
		},
		{
			name: "cardinality of empty set",
			key:  "emptyset",
			setupHashMap: func() {
				resetHashMap()
				emptySet := []string{}
				jsonData, _ := json.Marshal(emptySet)
				setHashMapValue("emptyset", string(jsonData))
			},
			expectedCardinality: 0,
		},
		{
			name: "cardinality of non-existing set",
			key:  "nonexistentset",
			setupHashMap: func() {
				resetHashMap()
			},
			expectedCardinality: 0,
		},
		{
			name: "cardinality of single member set",
			key:  "singleset",
			setupHashMap: func() {
				resetHashMap()
				singleSet := []string{"member1"}
				jsonData, _ := json.Marshal(singleSet)
				setHashMapValue("singleset", string(jsonData))
			},
			expectedCardinality: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupHashMap()
			mockEM := NewMockEventManager()
			event := createScardCommandEvent(tt.key)

			Scard(&mockEM.EventManager, &event)

			response, ok := waitForScardResponse(event.Reply, 100*time.Millisecond)
			require.True(t, ok, "Should receive response")
			assert.Equal(t, tt.expectedCardinality, response.Cardinality, "Cardinality should match expected")
			assert.Empty(t, response.Error, "Should not have error")
		})
	}
}

// Additional helper functions
func createSinterCommandEvent(keys []string) events.SinterCommandEvent {
	return events.SinterCommandEvent{
		Payload: &dto.SinterCommandRequest{
			Keys: keys,
		},
		Reply: make(chan *dto.SinterCommandResponse, 1),
	}
}

func createScardCommandEvent(key string) events.ScardCommandEvent {
	return events.ScardCommandEvent{
		Payload: &dto.ScardCommandRequest{
			Key: key,
		},
		Reply: make(chan *dto.ScardCommandResponse, 1),
	}
}

func waitForSinterResponse(replyChan chan *dto.SinterCommandResponse, timeout time.Duration) (*dto.SinterCommandResponse, bool) {
	select {
	case response := <-replyChan:
		return response, true
	case <-time.After(timeout):
		return nil, false
	}
}

func waitForScardResponse(replyChan chan *dto.ScardCommandResponse, timeout time.Duration) (*dto.ScardCommandResponse, bool) {
	select {
	case response := <-replyChan:
		return response, true
	case <-time.After(timeout):
		return nil, false
	}
}
