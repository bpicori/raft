package http

import (
	"context"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/consts" // Mock this later if necessary

	"github.com/stretchr/testify/assert"
)

// MockRaftNode can be a simple struct that implements the parts of raft.Raft needed for tests
type MockRaftNode struct {
	mu   sync.Mutex
	role consts.Role
}

func (m *MockRaftNode) GetCurrentRole() consts.Role {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.role
}

func (m *MockRaftNode) SetRole(role consts.Role) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.role = role
}

// Add other methods if HttpServer starts using them

func TestNewHttpServer(t *testing.T) {
	mockRaft := &MockRaftNode{}
	ctx := context.Background()
	config := config.Config{
		HTTPPort: "8080",
	}

	server := NewHttpServer(config, mockRaft, ctx)

	assert.NotNil(t, server, "HttpServer should not be nil")
	assert.Equal(t, config.HTTPPort, server.config.HTTPPort, "Port should be initialized correctly")
	assert.Equal(t, mockRaft, server.raftNode, "RaftNode should be initialized correctly")
	assert.Equal(t, ctx, server.ctx, "Context should be initialized correctly")
	assert.Nil(t, server.server, "Initially, the http.Server should be nil")
	assert.Nil(t, server.serverCancel, "Initially, serverCancel should be nil")
}

func TestHttpServer_Lifecycle_LeaderStarts_FollowerStops(t *testing.T) {
	// Use a different port for this test to avoid conflicts
	testPort := "8081"
	mockRaftNode := &MockRaftNode{}
	mockRaftNode.SetRole(consts.Follower) // Start as Follower

	// Main context for the HttpServer
	mainCtx, mainCancel := context.WithCancel(context.Background())
	defer mainCancel() // Ensure cleanup

	config := config.Config{
		HTTPPort: testPort,
	}
	httpServer := NewHttpServer(config, mockRaftNode, mainCtx)

	// Start the HttpServer's lifecycle manager in a goroutine
	go httpServer.Start()

	// Helper to check server availability
	checkServer := func(shouldBeUp bool, expectedStatus int, expectedBodySubstring string) {
		client := http.Client{Timeout: 200 * time.Millisecond} // Short timeout for check
		url := "http://localhost:" + testPort + "/"
		resp, err := client.Get(url)

		if shouldBeUp {
			assert.NoError(t, err, "Server should be up, but GET failed")
			if err == nil {
				defer resp.Body.Close()
				assert.Equal(t, expectedStatus, resp.StatusCode, "Unexpected status code")
				bodyBytes, _ := io.ReadAll(resp.Body)
				assert.Contains(t, string(bodyBytes), expectedBodySubstring, "Unexpected response body")
			}
		} else {
			assert.Error(t, err, "Server should be down, but GET succeeded or took too long")
		}
	}

	// 1. Initially Follower, server should not be running
	t.Log("Checking server state: Initial (Follower)")
	time.Sleep(INTERVAL + 200*time.Millisecond) // Wait for at least one cycle + buffer
	checkServer(false, 0, "")

	// 2. Transition to Leader, server should start
	t.Log("Checking server state: Transition to Leader")
	mockRaftNode.SetRole(consts.Leader)
	// Wait for the server to potentially start (INTERVAL for check + some time for startup)
	// Asserting server is up can take a few tries if startup is not immediate
	assert.Eventually(t, func() bool {
		client := http.Client{Timeout: 200 * time.Millisecond}
		resp, err := client.Get("http://localhost:" + testPort + "/")
		if err != nil {
			return false
		}
		resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 3*INTERVAL, 300*time.Millisecond, "Server did not start after becoming Leader")
	checkServer(true, http.StatusOK, "Raft Leader answering")

	// 3. Transition back to Follower, server should stop
	t.Log("Checking server state: Transition back to Follower")
	mockRaftNode.SetRole(consts.Follower)
	// Wait for the server to potentially stop
	assert.Eventually(t, func() bool {
		client := http.Client{Timeout: 200 * time.Millisecond}
		_, err := client.Get("http://localhost:" + testPort + "/")
		return err != nil // Expect an error when server is down
	}, SHUTDOWN_TIMEOUT+INTERVAL, 300*time.Millisecond, "Server did not stop after becoming Follower")
	checkServer(false, 0, "")

	// 4. Cancel main context, ensure lifecycle manager exits (already deferred)
	t.Log("Cancelling main context for server shutdown")
	mainCancel()
	// Give it a moment to process the cancellation if Start() loop is active
	time.Sleep(100 * time.Millisecond)
}

func TestHttpServer_Lifecycle_MainContextCancelStopsServer(t *testing.T) {
	testPort := "8082" // Different port
	mockRaftNode := &MockRaftNode{}
	mockRaftNode.SetRole(consts.Leader) // Start as Leader

	mainCtx, mainCancel := context.WithCancel(context.Background())
	// No defer mainCancel() here, we will call it explicitly to test shutdown

	config := config.Config{
		HTTPPort: testPort,
	}
	httpServer := NewHttpServer(config, mockRaftNode, mainCtx)
	go httpServer.Start()

	// Wait for server to start
	assert.Eventually(t, func() bool {
		client := http.Client{Timeout: 200 * time.Millisecond}
		resp, err := client.Get("http://localhost:" + testPort + "/")
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 3*INTERVAL, 300*time.Millisecond, "Server did not start when leader")

	// Check it's actually serving the leader response
	client := http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get("http://localhost:" + testPort + "/")
	assert.NoError(t, err)
	if err == nil {
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		bodyBytes, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(bodyBytes), "Raft Leader answering")
		resp.Body.Close()
	}

	// Cancel the main context
	t.Log("Cancelling main context to initiate server shutdown")
	mainCancel()

	// Server should shut down
	// The shutdown timeout in http.go is SHUTDOWN_TIMEOUT (6s) for the lifecycle manager
	// plus 5s for the serverInstance.Shutdown itself. So, wait a bit longer.
	assert.Eventually(t, func() bool {
		client := http.Client{Timeout: 200 * time.Millisecond}
		_, errGet := client.Get("http://localhost:" + testPort + "/")
		return errGet != nil
	}, SHUTDOWN_TIMEOUT+1*time.Second, 300*time.Millisecond, "Server did not stop after main context cancellation")

	// Final check to confirm it's down
	_, err = client.Get("http://localhost:" + testPort + "/")
	assert.Error(t, err, "Server should be down after context cancellation")
}

func TestHttpServer_HandleRoot_RoleChangeMidRequest(t *testing.T) {
	testPort := "8083" // Different port
	mockRaftNode := &MockRaftNode{}
	mockRaftNode.SetRole(consts.Leader) // Start as Leader

	mainCtx, mainCancel := context.WithCancel(context.Background())
	defer mainCancel() // Ensure cleanup eventually

	config := config.Config{
		HTTPPort: testPort,
	}
	httpServer := NewHttpServer(config, mockRaftNode, mainCtx)
	go httpServer.Start()

	// 1. Wait for server to start as Leader
	assert.Eventually(t, func() bool {
		client := http.Client{Timeout: 200 * time.Millisecond}
		resp, err := client.Get("http://localhost:" + testPort + "/")
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 3*INTERVAL, 300*time.Millisecond, "Server did not start when leader")

	// Confirm Leader response
	client := http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get("http://localhost:" + testPort + "/")
	assert.NoError(t, err)
	if assert.NotNil(t, resp) {
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		bodyBytes, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(bodyBytes), "Raft Leader answering")
		resp.Body.Close()
	}

	// 2. Change role to Follower
	t.Log("Changing role to Follower while server is running")
	mockRaftNode.SetRole(consts.Follower)

	// 3. Immediately make a request. Server is still running but handler should deny.
	// The HttpServer.Start() loop will take up to INTERVAL to notice the role change and stop the server.
	// Before that, requests to the still-running server should be handled by handleRoot seeing Follower role.

	// Give a very short moment for SetRole to take effect, but not enough for the server to stop via lifecycle
	time.Sleep(50 * time.Millisecond)

	resp, err = client.Get("http://localhost:" + testPort + "/")
	assert.NoError(t, err, "Expected request to go through to the still-running server")
	if assert.NotNil(t, resp) {
		assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode, "Expected 503 from handler due to Follower role")
		bodyBytes, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(bodyBytes), "Not a Raft Leader", "Expected 'Not a Raft Leader' body")
		resp.Body.Close()
	}

	// 4. Eventually, the server should stop due to the lifecycle manager reacting to Follower role
	t.Log("Waiting for server to fully stop after role change to Follower")
	assert.Eventually(t, func() bool {
		clientTest := http.Client{Timeout: 200 * time.Millisecond}
		_, errGet := clientTest.Get("http://localhost:" + testPort + "/")
		return errGet != nil // Expect an error when server is down
	}, SHUTDOWN_TIMEOUT+INTERVAL, 300*time.Millisecond, "Server did not stop after becoming Follower")

}
