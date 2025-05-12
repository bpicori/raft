package storage

import (
	"os"
	"path/filepath"
	"testing"

	"bpicori/raft/pkgs/dto"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestPersistAndLoadStateMachine(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "raft-storage-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a test state with some data
	state := &dto.StateMachineState{
		CurrentTerm:  5,
		VotedFor:     "server2",
		CommitLength: 10,
		LogEntry: []*dto.LogEntry{
			{
				Term: 3,
				Uuid: "entry1",
				Command: &dto.Command{
					Operation: "set",
					Args: &dto.Command_SetCommand{
						SetCommand: &dto.SetCommand{
							Key:   "testKey",
							Value: "testValue",
						},
					},
				},
			},
		},
	}

	// Test persisting state
	serverId := "test-server"
	err = PersistStateMachine(serverId, tempDir, state)
	assert.NoError(t, err)

	// Verify the file was created
	expectedFilePath := filepath.Join(tempDir, serverId+".pb")
	_, err = os.Stat(expectedFilePath)
	assert.NoError(t, err)

	// Verify file contents
	data, err := os.ReadFile(expectedFilePath)
	assert.NoError(t, err)

	var loadedStateDirectly dto.StateMachineState
	err = proto.Unmarshal(data, &loadedStateDirectly)
	assert.NoError(t, err)
	assert.Equal(t, state.CurrentTerm, loadedStateDirectly.CurrentTerm)
	assert.Equal(t, state.VotedFor, loadedStateDirectly.VotedFor)
	assert.Equal(t, state.CommitLength, loadedStateDirectly.CommitLength)
	assert.Equal(t, 1, len(loadedStateDirectly.LogEntry))

	// Test loading state
	loadedState, err := LoadStateMachine(serverId, tempDir)
	assert.NoError(t, err)
	assert.Equal(t, state.CurrentTerm, loadedState.CurrentTerm)
	assert.Equal(t, state.VotedFor, loadedState.VotedFor)
	assert.Equal(t, state.CommitLength, loadedState.CommitLength)
	assert.Equal(t, 1, len(loadedState.LogEntry))
	assert.Equal(t, state.LogEntry[0].Term, loadedState.LogEntry[0].Term)
	assert.Equal(t, state.LogEntry[0].Uuid, loadedState.LogEntry[0].Uuid)
	assert.Equal(t, "set", loadedState.LogEntry[0].Command.Operation)
	assert.Equal(t, "testKey", loadedState.LogEntry[0].Command.GetSetCommand().Key)
	assert.Equal(t, "testValue", loadedState.LogEntry[0].Command.GetSetCommand().Value)
}

func TestLoadStateMachineNonExistent(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "raft-storage-test-nonexistent")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Test loading a state that doesn't exist
	serverId := "nonexistent-server"
	state, err := LoadStateMachine(serverId, tempDir)
	assert.NoError(t, err)
	assert.NotNil(t, state)
	assert.Equal(t, int32(0), state.CurrentTerm)
	assert.Equal(t, "", state.VotedFor)
	assert.Equal(t, 0, len(state.LogEntry))
	assert.Equal(t, int32(0), state.CommitLength)
}

func TestPersistStateMachineError(t *testing.T) {
	// Create a test state
	state := &dto.StateMachineState{
		CurrentTerm: 1,
		VotedFor:    "server1",
	}

	// Test persisting to a non-existent directory
	err := PersistStateMachine("server1", "/nonexistent/dir", state)
	assert.Error(t, err)
}

func TestLoadStateMachineCorruptedData(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "raft-storage-test-corrupted")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a file with corrupted (non-proto) data
	serverId := "corrupted-server"
	filePath := filepath.Join(tempDir, serverId+".pb")
	corruptedData := []byte("this is not a valid protobuf message")
	err = os.WriteFile(filePath, corruptedData, 0644)
	assert.NoError(t, err)

	// Try to load the corrupted state
	_, err = LoadStateMachine(serverId, tempDir)
	assert.Error(t, err)
}

func TestPersistAndLoadMultipleStates(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "raft-storage-test-multiple")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create and persist multiple server states
	serverIds := []string{"server1", "server2", "server3"}
	states := make([]*dto.StateMachineState, len(serverIds))

	for i, serverId := range serverIds {
		states[i] = &dto.StateMachineState{
			CurrentTerm:  int32(i + 1),
			VotedFor:     "voter" + serverId,
			CommitLength: int32(i * 5),
		}
		err = PersistStateMachine(serverId, tempDir, states[i])
		assert.NoError(t, err)
	}

	// Verify each state can be loaded correctly
	for i, serverId := range serverIds {
		loadedState, err := LoadStateMachine(serverId, tempDir)
		assert.NoError(t, err)
		assert.Equal(t, states[i].CurrentTerm, loadedState.CurrentTerm)
		assert.Equal(t, states[i].VotedFor, loadedState.VotedFor)
		assert.Equal(t, states[i].CommitLength, loadedState.CommitLength)
	}
}

func TestFilePermissionErrors(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "raft-storage-test-permissions")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a test state
	state := &dto.StateMachineState{
		CurrentTerm: 1,
		VotedFor:    "server1",
	}

	// Create a read-only directory
	readOnlyDir := filepath.Join(tempDir, "readonly")
	err = os.Mkdir(readOnlyDir, 0500) // Read-only, no write permission
	assert.NoError(t, err)

	// Try to persist to the read-only directory
	err = PersistStateMachine("server1", readOnlyDir, state)
	assert.Error(t, err, "Should fail to write to read-only directory")

	// Create a file and make it read-only
	serverId := "permission-server"
	filePath := filepath.Join(tempDir, serverId+".pb")

	// First create and write with normal permissions
	err = PersistStateMachine(serverId, tempDir, state)
	assert.NoError(t, err)

	// Now make it read-only
	err = os.Chmod(filePath, 0400) // Read-only
	assert.NoError(t, err)

	// Try to overwrite the read-only file
	err = PersistStateMachine(serverId, tempDir, state)
	assert.Error(t, err, "Should fail to overwrite read-only file")
}
