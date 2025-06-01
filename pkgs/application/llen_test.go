package application

import (
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/events"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

func TestLlen(t *testing.T) {
	// Save original hashMap and restore after test
	defer func() { hashMap = sync.Map{} }()

	tests := []struct {
		name           string
		key            string
		setupData      interface{} // data to store in hashMap before test
		expectedLength int32
		expectedError  string
	}{
		{
			name:           "empty key should return error",
			key:            "",
			setupData:      nil,
			expectedLength: 0,
			expectedError:  "key is required",
		},
		{
			name:           "non-existent key should return 0",
			key:            "nonexistent",
			setupData:      nil,
			expectedLength: 0,
			expectedError:  "",
		},
		{
			name:           "empty list should return 0",
			key:            "empty_list",
			setupData:      []string{},
			expectedLength: 0,
			expectedError:  "",
		},
		{
			name:           "list with one element should return 1",
			key:            "single_element",
			setupData:      []string{"element1"},
			expectedLength: 1,
			expectedError:  "",
		},
		{
			name:           "list with multiple elements should return correct length",
			key:            "multi_element",
			setupData:      []string{"element1", "element2", "element3", "element4"},
			expectedLength: 4,
			expectedError:  "",
		},
		{
			name:           "non-list value should return 0",
			key:            "not_a_list",
			setupData:      "just a string",
			expectedLength: 0,
			expectedError:  "",
		},
		{
			name:           "invalid JSON should return 0",
			key:            "invalid_json",
			setupData:      "invalid json string",
			expectedLength: 0,
			expectedError:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset hashMap for each test
			hashMap = sync.Map{}

			// Setup test data if provided
			if tt.setupData != nil {
				if list, ok := tt.setupData.([]string); ok {
					jsonData, _ := json.Marshal(list)
					hashMap.Store(tt.key, string(jsonData))
				} else {
					// Store non-list data directly
					hashMap.Store(tt.key, tt.setupData)
				}
			}

			// Create event manager (not used in LLEN but required)
			eventManager := &events.EventManager{}

			// Create LLEN command event
			llenEvent := &events.LlenCommandEvent{
				Payload: &dto.LlenCommandRequest{
					Key: tt.key,
				},
				Reply: make(chan *dto.LlenCommandResponse, 1),
			}

			// Execute LLEN command
			Llen(eventManager, llenEvent)

			// Get response
			select {
			case response := <-llenEvent.Reply:
				if response.Length != tt.expectedLength {
					t.Errorf("Expected length %d, got %d", tt.expectedLength, response.Length)
				}
				if response.Error != tt.expectedError {
					t.Errorf("Expected error '%s', got '%s'", tt.expectedError, response.Error)
				}
			case <-time.After(1 * time.Second):
				t.Fatal("Timeout waiting for response")
			}
		})
	}
}

func TestLlenWithNilPayload(t *testing.T) {
	// Save original hashMap and restore after test
	defer func() { hashMap = sync.Map{} }()

	// Reset hashMap
	hashMap = sync.Map{}

	// Create event manager
	eventManager := &events.EventManager{}

	// Create LLEN command event with nil payload
	llenEvent := &events.LlenCommandEvent{
		Payload: nil,
		Reply:   make(chan *dto.LlenCommandResponse, 1),
	}

	// Execute LLEN command
	Llen(eventManager, llenEvent)

	// Get response
	select {
	case response := <-llenEvent.Reply:
		if response.Length != 0 {
			t.Errorf("Expected length 0, got %d", response.Length)
		}
		if response.Error == "" {
			t.Error("Expected error for nil payload, got empty string")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for response")
	}
}

func TestValidateLlenCommand(t *testing.T) {
	tests := []struct {
		name        string
		payload     *dto.LlenCommandRequest
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid command should pass",
			payload: &dto.LlenCommandRequest{
				Key: "valid_key",
			},
			expectError: false,
		},
		{
			name: "empty key should fail",
			payload: &dto.LlenCommandRequest{
				Key: "",
			},
			expectError: true,
			errorMsg:    "key is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &events.LlenCommandEvent{
				Payload: tt.payload,
			}

			err := validateLlenCommand(event)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestLlenIntegrationWithOtherListCommands tests LLEN with other list operations
func TestLlenIntegrationWithOtherListCommands(t *testing.T) {
	// Save original hashMap and restore after test
	defer func() { hashMap = sync.Map{} }()

	// Reset hashMap
	hashMap = sync.Map{}

	key := "test_list"

	// Helper function to get list length
	getLlen := func() int32 {
		llenEvent := &events.LlenCommandEvent{
			Payload: &dto.LlenCommandRequest{Key: key},
			Reply:   make(chan *dto.LlenCommandResponse, 1),
		}
		Llen(&events.EventManager{}, llenEvent)
		response := <-llenEvent.Reply
		return response.Length
	}

	// Initially, list should not exist (length 0)
	if length := getLlen(); length != 0 {
		t.Errorf("Expected initial length 0, got %d", length)
	}

	// Manually add some elements to simulate LPUSH
	testList := []string{"element1", "element2", "element3"}
	jsonData, _ := json.Marshal(testList)
	hashMap.Store(key, string(jsonData))

	// Check length after adding elements
	if length := getLlen(); length != 3 {
		t.Errorf("Expected length 3 after adding elements, got %d", length)
	}

	// Simulate removing one element (like LPOP)
	testList = testList[1:] // Remove first element
	jsonData, _ = json.Marshal(testList)
	hashMap.Store(key, string(jsonData))

	// Check length after removing element
	if length := getLlen(); length != 2 {
		t.Errorf("Expected length 2 after removing element, got %d", length)
	}

	// Remove all elements
	hashMap.Delete(key)

	// Check length after deleting key
	if length := getLlen(); length != 0 {
		t.Errorf("Expected length 0 after deleting key, got %d", length)
	}
}
