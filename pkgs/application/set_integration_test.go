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

// TestSetOperationsIntegration tests the complete workflow of set operations
func TestSetOperationsIntegration(t *testing.T) {
	// Reset hashMap for clean test
	resetHashMap()
	mockEM := NewMockEventManager()
	ctx, cancel := createTestContext(1 * time.Second)
	defer cancel()

	mockEM.StartMockAppendLogEntryHandler(ctx)

	// Test 1: SADD - Add members to a new set
	saddEvent := createSaddCommandEvent("myset", []string{"member1", "member2", "member3"})
	Sadd(&mockEM.EventManager, &saddEvent)

	saddResponse, ok := waitForSaddResponse(saddEvent.Reply, 200*time.Millisecond)
	require.True(t, ok, "Should receive SADD response")
	assert.Equal(t, int32(3), saddResponse.Added, "Should add 3 members")
	assert.Empty(t, saddResponse.Error, "Should not have error")

	// Test 2: SCARD - Check cardinality
	scardEvent := createScardCommandEvent("myset")
	Scard(&mockEM.EventManager, &scardEvent)

	scardResponse, ok := waitForScardResponse(scardEvent.Reply, 100*time.Millisecond)
	require.True(t, ok, "Should receive SCARD response")
	assert.Equal(t, int32(3), scardResponse.Cardinality, "Should have 3 members")
	assert.Empty(t, scardResponse.Error, "Should not have error")

	// Test 3: SISMEMBER - Check if member exists
	sismemberEvent := createSismemberCommandEvent("myset", "member2")
	Sismember(&mockEM.EventManager, &sismemberEvent)

	sismemberResponse, ok := waitForSismemberResponse(sismemberEvent.Reply, 100*time.Millisecond)
	require.True(t, ok, "Should receive SISMEMBER response")
	assert.Equal(t, int32(1), sismemberResponse.IsMember, "member2 should exist")
	assert.Empty(t, sismemberResponse.Error, "Should not have error")

	// Test 4: SISMEMBER - Check if non-existing member exists
	sismemberEvent2 := createSismemberCommandEvent("myset", "member4")
	Sismember(&mockEM.EventManager, &sismemberEvent2)

	sismemberResponse2, ok := waitForSismemberResponse(sismemberEvent2.Reply, 100*time.Millisecond)
	require.True(t, ok, "Should receive SISMEMBER response")
	assert.Equal(t, int32(0), sismemberResponse2.IsMember, "member4 should not exist")
	assert.Empty(t, sismemberResponse2.Error, "Should not have error")

	// Test 5: SADD - Add duplicate and new members
	saddEvent2 := createSaddCommandEvent("myset", []string{"member2", "member4", "member5"})
	Sadd(&mockEM.EventManager, &saddEvent2)

	saddResponse2, ok := waitForSaddResponse(saddEvent2.Reply, 200*time.Millisecond)
	require.True(t, ok, "Should receive SADD response")
	assert.Equal(t, int32(2), saddResponse2.Added, "Should add 2 new members (member4, member5)")
	assert.Empty(t, saddResponse2.Error, "Should not have error")

	// Test 6: SCARD - Check updated cardinality
	scardEvent2 := createScardCommandEvent("myset")
	Scard(&mockEM.EventManager, &scardEvent2)

	scardResponse2, ok := waitForScardResponse(scardEvent2.Reply, 100*time.Millisecond)
	require.True(t, ok, "Should receive SCARD response")
	assert.Equal(t, int32(5), scardResponse2.Cardinality, "Should have 5 members")
	assert.Empty(t, scardResponse2.Error, "Should not have error")

	// Test 7: SREM - Remove some members
	sremEvent := createSremCommandEvent("myset", []string{"member1", "member3", "member6"})
	Srem(&mockEM.EventManager, &sremEvent)

	sremResponse, ok := waitForSremResponse(sremEvent.Reply, 200*time.Millisecond)
	require.True(t, ok, "Should receive SREM response")
	assert.Equal(t, int32(2), sremResponse.Removed, "Should remove 2 members (member1, member3)")
	assert.Empty(t, sremResponse.Error, "Should not have error")

	// Test 8: SCARD - Check cardinality after removal
	scardEvent3 := createScardCommandEvent("myset")
	Scard(&mockEM.EventManager, &scardEvent3)

	scardResponse3, ok := waitForScardResponse(scardEvent3.Reply, 100*time.Millisecond)
	require.True(t, ok, "Should receive SCARD response")
	assert.Equal(t, int32(3), scardResponse3.Cardinality, "Should have 3 members")
	assert.Empty(t, scardResponse3.Error, "Should not have error")
}

// TestSetIntersectionIntegration tests SINTER with multiple sets
func TestSetIntersectionIntegration(t *testing.T) {
	// Reset hashMap for clean test
	resetHashMap()
	mockEM := NewMockEventManager()
	ctx, cancel := createTestContext(1 * time.Second)
	defer cancel()

	mockEM.StartMockAppendLogEntryHandler(ctx)

	// Create first set
	saddEvent1 := createSaddCommandEvent("set1", []string{"a", "b", "c", "d"})
	Sadd(&mockEM.EventManager, &saddEvent1)
	saddResponse1, ok := waitForSaddResponse(saddEvent1.Reply, 200*time.Millisecond)
	require.True(t, ok, "Should receive SADD response for set1")
	assert.Equal(t, int32(4), saddResponse1.Added, "Should add 4 members to set1")

	// Create second set
	saddEvent2 := createSaddCommandEvent("set2", []string{"b", "c", "e", "f"})
	Sadd(&mockEM.EventManager, &saddEvent2)
	saddResponse2, ok := waitForSaddResponse(saddEvent2.Reply, 200*time.Millisecond)
	require.True(t, ok, "Should receive SADD response for set2")
	assert.Equal(t, int32(4), saddResponse2.Added, "Should add 4 members to set2")

	// Create third set
	saddEvent3 := createSaddCommandEvent("set3", []string{"c", "d", "g", "h"})
	Sadd(&mockEM.EventManager, &saddEvent3)
	saddResponse3, ok := waitForSaddResponse(saddEvent3.Reply, 200*time.Millisecond)
	require.True(t, ok, "Should receive SADD response for set3")
	assert.Equal(t, int32(4), saddResponse3.Added, "Should add 4 members to set3")

	// Test intersection of set1 and set2 (should be: b, c)
	sinterEvent1 := createSinterCommandEvent([]string{"set1", "set2"})
	Sinter(&mockEM.EventManager, &sinterEvent1)

	sinterResponse1, ok := waitForSinterResponse(sinterEvent1.Reply, 100*time.Millisecond)
	require.True(t, ok, "Should receive SINTER response")
	assert.ElementsMatch(t, []string{"b", "c"}, sinterResponse1.Members, "Intersection should be b, c")
	assert.Empty(t, sinterResponse1.Error, "Should not have error")

	// Test intersection of all three sets (should be: c)
	sinterEvent2 := createSinterCommandEvent([]string{"set1", "set2", "set3"})
	Sinter(&mockEM.EventManager, &sinterEvent2)

	sinterResponse2, ok := waitForSinterResponse(sinterEvent2.Reply, 100*time.Millisecond)
	require.True(t, ok, "Should receive SINTER response")
	assert.ElementsMatch(t, []string{"c"}, sinterResponse2.Members, "Intersection should be c")
	assert.Empty(t, sinterResponse2.Error, "Should not have error")

	// Test intersection with non-existing set (should be empty)
	sinterEvent3 := createSinterCommandEvent([]string{"set1", "nonexistent"})
	Sinter(&mockEM.EventManager, &sinterEvent3)

	sinterResponse3, ok := waitForSinterResponse(sinterEvent3.Reply, 100*time.Millisecond)
	require.True(t, ok, "Should receive SINTER response")
	assert.Empty(t, sinterResponse3.Members, "Intersection with non-existing set should be empty")
	assert.Empty(t, sinterResponse3.Error, "Should not have error")
}

// TestSetValidationErrors tests error handling for invalid inputs
func TestSetValidationErrors(t *testing.T) {
	resetHashMap()
	mockEM := NewMockEventManager()

	// Test SADD with empty key
	saddEvent := events.SaddCommandEvent{
		Payload: &dto.SaddCommandRequest{
			Key:     "",
			Members: []string{"member1"},
		},
		Reply: make(chan *dto.SaddCommandResponse, 1),
	}
	Sadd(&mockEM.EventManager, &saddEvent)

	saddResponse, ok := waitForSaddResponse(saddEvent.Reply, 100*time.Millisecond)
	require.True(t, ok, "Should receive SADD response")
	assert.Equal(t, int32(0), saddResponse.Added, "Should not add any members")
	assert.NotEmpty(t, saddResponse.Error, "Should have error for empty key")

	// Test SADD with no members
	saddEvent2 := events.SaddCommandEvent{
		Payload: &dto.SaddCommandRequest{
			Key:     "myset",
			Members: []string{},
		},
		Reply: make(chan *dto.SaddCommandResponse, 1),
	}
	Sadd(&mockEM.EventManager, &saddEvent2)

	saddResponse2, ok := waitForSaddResponse(saddEvent2.Reply, 100*time.Millisecond)
	require.True(t, ok, "Should receive SADD response")
	assert.Equal(t, int32(0), saddResponse2.Added, "Should not add any members")
	assert.NotEmpty(t, saddResponse2.Error, "Should have error for no members")

	// Test SISMEMBER with empty key
	sismemberEvent := events.SismemberCommandEvent{
		Payload: &dto.SismemberCommandRequest{
			Key:    "",
			Member: "member1",
		},
		Reply: make(chan *dto.SismemberCommandResponse, 1),
	}
	Sismember(&mockEM.EventManager, &sismemberEvent)

	sismemberResponse, ok := waitForSismemberResponse(sismemberEvent.Reply, 100*time.Millisecond)
	require.True(t, ok, "Should receive SISMEMBER response")
	assert.Equal(t, int32(0), sismemberResponse.IsMember, "Should return 0")
	assert.NotEmpty(t, sismemberResponse.Error, "Should have error for empty key")

	// Test SINTER with empty keys
	sinterEvent := events.SinterCommandEvent{
		Payload: &dto.SinterCommandRequest{
			Keys: []string{},
		},
		Reply: make(chan *dto.SinterCommandResponse, 1),
	}
	Sinter(&mockEM.EventManager, &sinterEvent)

	sinterResponse, ok := waitForSinterResponse(sinterEvent.Reply, 100*time.Millisecond)
	require.True(t, ok, "Should receive SINTER response")
	assert.Empty(t, sinterResponse.Members, "Should return empty members")
	assert.NotEmpty(t, sinterResponse.Error, "Should have error for no keys")
}

// TestSetReplicationFunctions tests the replication functions directly
func TestSetReplicationFunctions(t *testing.T) {
	resetHashMap()

	// Test SADD replication
	saddCommand := &dto.SaddCommand{
		Key:     "testset",
		Members: []string{"member1", "member2", "member3"},
	}
	replicateSaddCommand(saddCommand)

	// Verify set was created
	value, exists := hashMap.Load("testset")
	require.True(t, exists, "Set should exist after replication")

	var members []string
	err := json.Unmarshal([]byte(value.(string)), &members)
	require.NoError(t, err, "Should unmarshal successfully")
	assert.ElementsMatch(t, []string{"member1", "member2", "member3"}, members, "Members should match")

	// Test SREM replication
	sremCommand := &dto.SremCommand{
		Key:     "testset",
		Members: []string{"member2"},
	}
	replicateSremCommand(sremCommand)

	// Verify member was removed
	value, exists = hashMap.Load("testset")
	require.True(t, exists, "Set should still exist after removal")

	err = json.Unmarshal([]byte(value.(string)), &members)
	require.NoError(t, err, "Should unmarshal successfully")
	assert.ElementsMatch(t, []string{"member1", "member3"}, members, "member2 should be removed")
}
