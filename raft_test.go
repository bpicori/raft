package raft

import (
	"os"
	"reflect"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name           string
		raftServers    string
		currentServer  string
		expectedConfig Config
		expectError    bool
	}{
		{
			name:          "Valid configuration",
			raftServers:   "localhost:8080,localhost:8081,localhost:8082",
			currentServer: "localhost:8080",
			expectedConfig: Config{
				Servers: map[string]ServerConfig{
					"localhost:8080": {ID: "localhost:8080", Addr: "localhost:8080"},
					"localhost:8081": {ID: "localhost:8081", Addr: "localhost:8081"},
					"localhost:8082": {ID: "localhost:8082", Addr: "localhost:8082"},
				},
				SelfID:     "localhost:8080",
				SelfServer: ServerConfig{ID: "localhost:8080", Addr: "localhost:8080"},
			},
			expectError: false,
		},
		{
			name:          "Missing RAFT_SERVERS",
			raftServers:   "",
			currentServer: "localhost:8080",
			expectError:   true,
		},
		{
			name:          "Missing CURRENT_SERVER",
			raftServers:   "localhost:8080,localhost:8081,localhost:8082",
			currentServer: "",
			expectError:   true,
		},
		{
			name:          "Invalid server in RAFT_SERVERS",
			raftServers:   "localhost:8080,invalid-address,localhost:8082",
			currentServer: "localhost:8080",
			expectError:   true,
		},
		{
			name:          "Current server not in RAFT_SERVERS",
			raftServers:   "localhost:8080,localhost:8081,localhost:8082",
			currentServer: "localhost:8083",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			os.Setenv("RAFT_SERVERS", tt.raftServers)
			os.Setenv("CURRENT_SERVER", tt.currentServer)

			// Run LoadConfig
			config, err := LoadConfig()

			// Check for expected error
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected an error, but got none")
				}
				return
			}

			// Check for unexpected error
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Compare the result with expected config
			if !reflect.DeepEqual(config, tt.expectedConfig) {
				t.Errorf("Config mismatch.\nExpected: %+v\nGot: %+v", tt.expectedConfig, config)
			}
		})
	}
}
