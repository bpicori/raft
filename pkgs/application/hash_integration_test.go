package application

import (
	"bpicori/raft/pkgs/dto"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashIntegration(t *testing.T) {
	// Reset hashMap for clean test
	resetHashMap()

	// Test 1: HSET - Create a new hash with multiple fields
	hsetCommand1 := &dto.HsetCommand{
		Key: "user:1",
		Fields: map[string]string{
			"name":  "John Doe",
			"email": "john@example.com",
			"age":   "30",
		},
	}
	replicateHsetCommand(hsetCommand1)

	// Verify the hash was created
	value, exists := hashMap.Load("user:1")
	require.True(t, exists, "Hash should exist")

	var hash map[string]string
	err := json.Unmarshal([]byte(value.(string)), &hash)
	require.NoError(t, err, "Should be able to unmarshal hash")

	expectedHash := map[string]string{
		"name":  "John Doe",
		"email": "john@example.com",
		"age":   "30",
	}
	assert.Equal(t, expectedHash, hash, "Hash should match expected")

	// Test 2: HGET - Get individual fields
	tests := []struct {
		field    string
		expected string
	}{
		{"name", "John Doe"},
		{"email", "john@example.com"},
		{"age", "30"},
		{"nonexistent", "(nil)"},
	}

	for _, tt := range tests {
		t.Run("HGET_"+tt.field, func(t *testing.T) {
			// Simulate HGET operation
			var currentHash map[string]string
			if existingValue, exists := hashMap.Load("user:1"); exists {
				if existingStr, ok := existingValue.(string); ok {
					json.Unmarshal([]byte(existingStr), &currentHash)
				}
			}

			var result string
			if currentHash != nil {
				if value, exists := currentHash[tt.field]; exists {
					result = value
				} else {
					result = "(nil)"
				}
			} else {
				result = "(nil)"
			}

			assert.Equal(t, tt.expected, result, "HGET result should match expected")
		})
	}

	// Test 3: HMGET - Get multiple fields
	t.Run("HMGET_multiple_fields", func(t *testing.T) {
		fields := []string{"name", "age", "nonexistent", "email"}
		expectedValues := []string{"John Doe", "30", "(nil)", "john@example.com"}

		// Simulate HMGET operation
		var currentHash map[string]string
		if existingValue, exists := hashMap.Load("user:1"); exists {
			if existingStr, ok := existingValue.(string); ok {
				json.Unmarshal([]byte(existingStr), &currentHash)
			}
		}

		values := make([]string, len(fields))
		for i, field := range fields {
			if currentHash != nil {
				if value, exists := currentHash[field]; exists {
					values[i] = value
				} else {
					values[i] = "(nil)"
				}
			} else {
				values[i] = "(nil)"
			}
		}

		assert.Equal(t, expectedValues, values, "HMGET values should match expected")
	})

	// Test 4: HINCRBY - Increment age field
	t.Run("HINCRBY_increment_age", func(t *testing.T) {
		hincrbyCommand := &dto.HincrbyCommand{
			Key:       "user:1",
			Field:     "age",
			Increment: 5,
		}
		replicateHincrbyCommand(hincrbyCommand)

		// Verify the field was incremented
		value, exists := hashMap.Load("user:1")
		require.True(t, exists, "Hash should exist")

		var hash map[string]string
		err := json.Unmarshal([]byte(value.(string)), &hash)
		require.NoError(t, err, "Should be able to unmarshal hash")

		assert.Equal(t, "35", hash["age"], "Age should be incremented to 35")
		assert.Equal(t, "John Doe", hash["name"], "Name should remain unchanged")
		assert.Equal(t, "john@example.com", hash["email"], "Email should remain unchanged")
	})

	// Test 5: HINCRBY - Create new numeric field
	t.Run("HINCRBY_new_field", func(t *testing.T) {
		hincrbyCommand := &dto.HincrbyCommand{
			Key:       "user:1",
			Field:     "score",
			Increment: 100,
		}
		replicateHincrbyCommand(hincrbyCommand)

		// Verify the new field was created
		value, exists := hashMap.Load("user:1")
		require.True(t, exists, "Hash should exist")

		var hash map[string]string
		err := json.Unmarshal([]byte(value.(string)), &hash)
		require.NoError(t, err, "Should be able to unmarshal hash")

		assert.Equal(t, "100", hash["score"], "Score should be set to 100")
		assert.Equal(t, "35", hash["age"], "Age should remain 35")
	})

	// Test 6: HSET - Update existing fields and add new ones
	t.Run("HSET_update_and_add", func(t *testing.T) {
		hsetCommand2 := &dto.HsetCommand{
			Key: "user:1",
			Fields: map[string]string{
				"email":    "john.doe@example.com", // Update existing
				"phone":    "+1234567890",          // Add new
				"location": "New York",             // Add new
			},
		}
		replicateHsetCommand(hsetCommand2)

		// Verify the hash was updated
		value, exists := hashMap.Load("user:1")
		require.True(t, exists, "Hash should exist")

		var hash map[string]string
		err := json.Unmarshal([]byte(value.(string)), &hash)
		require.NoError(t, err, "Should be able to unmarshal hash")

		expectedHash := map[string]string{
			"name":     "John Doe",
			"email":    "john.doe@example.com", // Updated
			"age":      "35",
			"score":    "100",
			"phone":    "+1234567890", // New
			"location": "New York",    // New
		}
		assert.Equal(t, expectedHash, hash, "Hash should match expected after update")
	})

	// Test 7: HINCRBY with negative increment
	t.Run("HINCRBY_negative_increment", func(t *testing.T) {
		hincrbyCommand := &dto.HincrbyCommand{
			Key:       "user:1",
			Field:     "score",
			Increment: -25,
		}
		replicateHincrbyCommand(hincrbyCommand)

		// Verify the field was decremented
		value, exists := hashMap.Load("user:1")
		require.True(t, exists, "Hash should exist")

		var hash map[string]string
		err := json.Unmarshal([]byte(value.(string)), &hash)
		require.NoError(t, err, "Should be able to unmarshal hash")

		assert.Equal(t, "75", hash["score"], "Score should be decremented to 75")
	})

	// Test 8: Operations on non-existent hash
	t.Run("operations_on_nonexistent_hash", func(t *testing.T) {
		// HGET on non-existent hash
		var currentHash map[string]string
		if existingValue, exists := hashMap.Load("nonexistent"); exists {
			if existingStr, ok := existingValue.(string); ok {
				json.Unmarshal([]byte(existingStr), &currentHash)
			}
		}

		var result string
		if currentHash != nil {
			if value, exists := currentHash["field1"]; exists {
				result = value
			} else {
				result = "(nil)"
			}
		} else {
			result = "(nil)"
		}
		assert.Equal(t, "(nil)", result, "HGET on non-existent hash should return (nil)")

		// HINCRBY on non-existent hash should create it
		hincrbyCommand := &dto.HincrbyCommand{
			Key:       "counter:1",
			Field:     "visits",
			Increment: 1,
		}
		replicateHincrbyCommand(hincrbyCommand)

		value, exists := hashMap.Load("counter:1")
		require.True(t, exists, "Hash should be created")

		var hash map[string]string
		err := json.Unmarshal([]byte(value.(string)), &hash)
		require.NoError(t, err, "Should be able to unmarshal hash")

		assert.Equal(t, "1", hash["visits"], "Visits should be set to 1")
	})
}
