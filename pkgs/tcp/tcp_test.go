package tcp

import (
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/events"
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestStartAndShutdown(t *testing.T) {
	addr := "127.0.0.1:9001"
	eventManager := events.NewEventManager()
	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	wg.Add(1)

	go Start(addr, eventManager, ctx, wg)
	time.Sleep(100 * time.Millisecond) // Allow server to start

	conn, err := net.Dial("tcp", addr)
	assert.NoError(t, err)
	assert.NotNil(t, conn)
	conn.Close()

	cancel()  // Trigger shutdown
	wg.Wait() // Wait for shutdown to complete

	// Server should be closed, new connections should fail
	_, err = net.Dial("tcp", addr)
	assert.Error(t, err)
}

func TestSendAsyncRPC_ConnectionError(t *testing.T) {
	// Using a port that's (hopefully) not in use
	addr := "127.0.0.1:65432"

	rpc := &dto.RaftRPC{
		Type: consts.VoteRequest.String(),
		Args: &dto.RaftRPC_VoteRequest{
			VoteRequest: &dto.VoteRequest{},
		},
	}

	err := SendAsyncRPC(addr, rpc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error getting connection")
}

func TestSendAsyncRPC(t *testing.T) {
	addr := "127.0.0.1:9002"
	eventManager := events.NewEventManager()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := &sync.WaitGroup{}
	wg.Add(1)

	go Start(addr, eventManager, ctx, wg)
	time.Sleep(100 * time.Millisecond) // Allow server to start

	voteRequestReceived := make(chan bool, 1)
	go func() {
		select {
		case <-eventManager.VoteRequestChan:
			voteRequestReceived <- true
		case <-time.After(500 * time.Millisecond):
			voteRequestReceived <- false
		}
	}()

	voteRequest := &dto.VoteRequest{
		Term:         1,
		CandidateId:  "node1",
		LastLogIndex: 0,
		LastLogTerm:  0,
	}

	rpc := &dto.RaftRPC{
		Type: consts.VoteRequest.String(),
		Args: &dto.RaftRPC_VoteRequest{
			VoteRequest: voteRequest,
		},
	}

	err := SendAsyncRPC(addr, rpc)
	assert.NoError(t, err)

	received := <-voteRequestReceived
	assert.True(t, received, "VoteRequest should be received by the server")
}

func TestSendSyncRPC(t *testing.T) {
	addr := "127.0.0.1:9003"
	eventManager := events.NewEventManager()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := &sync.WaitGroup{}
	wg.Add(1)

	go Start(addr, eventManager, ctx, wg)
	time.Sleep(100 * time.Millisecond) // Allow server to start

	// Handle NodeStatus requests in a goroutine
	go func() {
		select {
		case event := <-eventManager.NodeStatusChan:
			response := &dto.NodeStatusResponse{
				NodeId:        "node1",
				CurrentTerm:   5,
				CurrentRole:   "leader",
				CurrentLeader: "node1",
			}
			event.Reply <- response
		case <-time.After(2 * time.Second):
			t.Error("Timeout waiting for NodeStatus request")
		}
	}()

	// Create and send a NodeStatus request
	request := &dto.RaftRPC{
		Type: consts.NodeStatus.String(),
		Args: &dto.RaftRPC_NodeStatusRequest{
			NodeStatusRequest: &dto.NodeStatusRequest{},
		},
	}

	response, err := SendSyncRPC(addr, request)
	assert.NoError(t, err)
	assert.NotNil(t, response)

	statusResponse := response.GetNodeStatusResponse()
	assert.NotNil(t, statusResponse)
	assert.Equal(t, "leader", statusResponse.CurrentRole)
	assert.Equal(t, int32(5), statusResponse.CurrentTerm)
}

func TestSendSyncRPC_ConnectionError(t *testing.T) {
	// Using a port that's (hopefully) not in use
	addr := "127.0.0.1:65433"

	request := &dto.RaftRPC{
		Type: consts.NodeStatus.String(),
		Args: &dto.RaftRPC_NodeStatusRequest{
			NodeStatusRequest: &dto.NodeStatusRequest{},
		},
	}

	_, err := SendSyncRPC(addr, request)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error getting connection")
}

func TestHandleConnection_InvalidMessageType(t *testing.T) {
	eventManager := events.NewEventManager()

	// Create a pair of connected net.Conn objects
	client, server := net.Pipe()
	defer client.Close()

	// Create an RPC with invalid type
	rpc := &dto.RaftRPC{
		Type: "InvalidType",
	}

	// Marshal and send the message
	data, err := proto.Marshal(rpc)
	assert.NoError(t, err)

	done := make(chan bool)
	go func() {
		HandleConnection(server, eventManager)
		done <- true
	}()

	_, err = client.Write(data)
	assert.NoError(t, err)

	// Connection should be closed by HandleConnection
	select {
	case <-done:
		// This is the expected path - connection should be closed
	case <-time.After(500 * time.Millisecond):
		t.Error("HandleConnection did not complete in time")
	}
}

func TestHandleConnection_VoteRequest(t *testing.T) {
	eventManager := events.NewEventManager()

	// Create a pair of connected net.Conn objects
	client, server := net.Pipe()
	defer client.Close()

	// Create a VoteRequest message
	voteRequest := &dto.VoteRequest{
		Term:         2,
		CandidateId:  "node2",
		LastLogIndex: 5,
		LastLogTerm:  1,
	}

	rpc := &dto.RaftRPC{
		Type: consts.VoteRequest.String(),
		Args: &dto.RaftRPC_VoteRequest{
			VoteRequest: voteRequest,
		},
	}

	// Marshal and send the message
	data, err := proto.Marshal(rpc)
	assert.NoError(t, err)

	voteRequestReceived := make(chan bool, 1)
	go func() {
		select {
		case req := <-eventManager.VoteRequestChan:
			assert.Equal(t, voteRequest.Term, req.Term)
			assert.Equal(t, voteRequest.CandidateId, req.CandidateId)
			assert.Equal(t, voteRequest.LastLogIndex, req.LastLogIndex)
			assert.Equal(t, voteRequest.LastLogTerm, req.LastLogTerm)
			voteRequestReceived <- true
		case <-time.After(500 * time.Millisecond):
			voteRequestReceived <- false
		}
	}()

	go HandleConnection(server, eventManager)

	_, err = client.Write(data)
	assert.NoError(t, err)

	received := <-voteRequestReceived
	assert.True(t, received, "VoteRequest should be received and processed")
}

func TestHandleConnection_VoteResponse(t *testing.T) {
	eventManager := events.NewEventManager()

	// Create a pair of connected net.Conn objects
	client, server := net.Pipe()
	defer client.Close()

	// Create a VoteResponse message
	voteResponse := &dto.VoteResponse{
		Term:        3,
		NodeId:      "node3",
		VoteGranted: true,
	}

	rpc := &dto.RaftRPC{
		Type: consts.VoteResponse.String(),
		Args: &dto.RaftRPC_VoteResponse{
			VoteResponse: voteResponse,
		},
	}

	// Marshal and send the message
	data, err := proto.Marshal(rpc)
	assert.NoError(t, err)

	voteResponseReceived := make(chan bool, 1)
	go func() {
		select {
		case resp := <-eventManager.VoteResponseChan:
			assert.Equal(t, voteResponse.Term, resp.Term)
			assert.Equal(t, voteResponse.NodeId, resp.NodeId)
			assert.Equal(t, voteResponse.VoteGranted, resp.VoteGranted)
			voteResponseReceived <- true
		case <-time.After(500 * time.Millisecond):
			voteResponseReceived <- false
		}
	}()

	go HandleConnection(server, eventManager)

	_, err = client.Write(data)
	assert.NoError(t, err)

	received := <-voteResponseReceived
	assert.True(t, received, "VoteResponse should be received and processed")
}

func TestHandleConnection_LogRequest(t *testing.T) {
	eventManager := events.NewEventManager()

	// Create a pair of connected net.Conn objects
	client, server := net.Pipe()
	defer client.Close()

	// Create a LogRequest message
	logRequest := &dto.LogRequest{
		Term:         4,
		LeaderId:     "leader1",
		PrefixLength: 10,
		PrefixTerm:   2,
		LeaderCommit: 8,
	}

	rpc := &dto.RaftRPC{
		Type: consts.LogRequest.String(),
		Args: &dto.RaftRPC_LogRequest{
			LogRequest: logRequest,
		},
	}

	// Marshal and send the message
	data, err := proto.Marshal(rpc)
	assert.NoError(t, err)

	logRequestReceived := make(chan bool, 1)
	go func() {
		select {
		case req := <-eventManager.LogRequestChan:
			assert.Equal(t, logRequest.Term, req.Term)
			assert.Equal(t, logRequest.LeaderId, req.LeaderId)
			assert.Equal(t, logRequest.PrefixLength, req.PrefixLength)
			assert.Equal(t, logRequest.PrefixTerm, req.PrefixTerm)
			assert.Equal(t, logRequest.LeaderCommit, req.LeaderCommit)
			logRequestReceived <- true
		case <-time.After(500 * time.Millisecond):
			logRequestReceived <- false
		}
	}()

	go HandleConnection(server, eventManager)

	_, err = client.Write(data)
	assert.NoError(t, err)

	received := <-logRequestReceived
	assert.True(t, received, "LogRequest should be received and processed")
}

func TestHandleConnection_LogResponse(t *testing.T) {
	eventManager := events.NewEventManager()

	// Create a pair of connected net.Conn objects
	client, server := net.Pipe()
	defer client.Close()

	// Create a LogResponse message
	logResponse := &dto.LogResponse{
		Term:       2,
		FollowerId: "follower1",
		Ack:        5,
		Success:    true,
	}

	rpc := &dto.RaftRPC{
		Type: consts.LogResponse.String(),
		Args: &dto.RaftRPC_LogResponse{
			LogResponse: logResponse,
		},
	}

	// Marshal and send the message
	data, err := proto.Marshal(rpc)
	assert.NoError(t, err)

	logResponseReceived := make(chan bool, 1)
	go func() {
		select {
		case resp := <-eventManager.LogResponseChan:
			assert.Equal(t, logResponse.Term, resp.Term)
			assert.Equal(t, logResponse.FollowerId, resp.FollowerId)
			assert.Equal(t, logResponse.Ack, resp.Ack)
			assert.Equal(t, logResponse.Success, resp.Success)
			logResponseReceived <- true
		case <-time.After(500 * time.Millisecond):
			logResponseReceived <- false
		}
	}()

	go HandleConnection(server, eventManager)

	_, err = client.Write(data)
	assert.NoError(t, err)

	received := <-logResponseReceived
	assert.True(t, received, "LogResponse should be received and processed")
}

func TestHandleConnection_SetCommand(t *testing.T) {
	eventManager := events.NewEventManager()

	// Create a pair of connected net.Conn objects
	client, server := net.Pipe()
	defer client.Close()

	// Create a SetCommand message
	setCommandRequest := &dto.SetCommandRequest{
		Key:   "testKey",
		Value: "testValue",
	}

	rpc := &dto.RaftRPC{
		Type: consts.SetCommand.String(),
		Args: &dto.RaftRPC_SetCommandRequest{
			SetCommandRequest: setCommandRequest,
		},
	}

	// Marshal and send the message
	data, err := proto.Marshal(rpc)
	assert.NoError(t, err)

	// Handle the SetCommand in a goroutine
	go func() {
		select {
		case event := <-eventManager.SetCommandRequestChan:
			assert.Equal(t, setCommandRequest.Key, event.Payload.Key)
			assert.Equal(t, setCommandRequest.Value, event.Payload.Value)

			// Send a success response
			event.Reply <- &dto.OkResponse{
				Ok: true,
			}
		case <-time.After(500 * time.Millisecond):
			t.Error("Timeout waiting for SetCommand")
		}
	}()

	go HandleConnection(server, eventManager)

	_, err = client.Write(data)
	assert.NoError(t, err)

	// Read the response
	buffer := make([]byte, 4096)
	n, err := client.Read(buffer)
	assert.NoError(t, err)

	var response dto.RaftRPC
	err = proto.Unmarshal(buffer[:n], &response)
	assert.NoError(t, err)

	okResponse := response.GetOkResponse()
	assert.NotNil(t, okResponse)
	assert.True(t, okResponse.Ok)
}

func TestHandleConnection_GetCommand(t *testing.T) {
	eventManager := events.NewEventManager()

	// Create a pair of connected net.Conn objects
	client, server := net.Pipe()
	defer client.Close()

	// Create a GetCommand message
	getCommandRequest := &dto.GetCommandRequest{
		Key: "testKey",
	}

	rpc := &dto.RaftRPC{
		Type: consts.GetCommand.String(),
		Args: &dto.RaftRPC_GetCommandRequest{
			GetCommandRequest: getCommandRequest,
		},
	}

	// Marshal and send the message
	data, err := proto.Marshal(rpc)
	assert.NoError(t, err)

	// Handle the GetCommand in a goroutine
	go func() {
		select {
		case event := <-eventManager.GetCommandRequestChan:
			assert.Equal(t, getCommandRequest.Key, event.Payload.Key)

			// Send a response
			event.Reply <- &dto.GetCommandResponse{
				Value: "testValue",
			}
		case <-time.After(500 * time.Millisecond):
			t.Error("Timeout waiting for GetCommand")
		}
	}()

	go HandleConnection(server, eventManager)

	_, err = client.Write(data)
	assert.NoError(t, err)

	// Read the response
	buffer := make([]byte, 4096)
	n, err := client.Read(buffer)
	assert.NoError(t, err)

	var response dto.RaftRPC
	err = proto.Unmarshal(buffer[:n], &response)
	assert.NoError(t, err)

	getResponse := response.GetGetCommandResponse()
	assert.NotNil(t, getResponse)
	assert.Equal(t, "testValue", getResponse.Value)
}

func TestHandleConnection_IncrCommand(t *testing.T) {
	eventManager := events.NewEventManager()

	// Create a pair of connected net.Conn objects
	client, server := net.Pipe()
	defer client.Close()

	// Create an IncrCommand message
	incrCommandRequest := &dto.IncrCommandRequest{
		Key: "counterKey",
	}

	rpc := &dto.RaftRPC{
		Type: consts.IncrCommand.String(),
		Args: &dto.RaftRPC_IncrCommandRequest{
			IncrCommandRequest: incrCommandRequest,
		},
	}

	// Marshal and send the message
	data, err := proto.Marshal(rpc)
	assert.NoError(t, err)

	// Handle the IncrCommand in a goroutine
	go func() {
		select {
		case event := <-eventManager.IncrCommandRequestChan:
			assert.Equal(t, incrCommandRequest.Key, event.Payload.Key)

			// Send a response
			event.Reply <- &dto.IncrCommandResponse{
				Value: 42,
			}
		case <-time.After(500 * time.Millisecond):
			t.Error("Timeout waiting for IncrCommand")
		}
	}()

	go HandleConnection(server, eventManager)

	_, err = client.Write(data)
	assert.NoError(t, err)

	// Read the response
	buffer := make([]byte, 4096)
	n, err := client.Read(buffer)
	assert.NoError(t, err)

	var response dto.RaftRPC
	err = proto.Unmarshal(buffer[:n], &response)
	assert.NoError(t, err)

	incrResponse := response.GetIncrCommandResponse()
	assert.NotNil(t, incrResponse)
	assert.Equal(t, int32(42), incrResponse.Value)
}

func TestHandleConnection_DecrCommand(t *testing.T) {
	eventManager := events.NewEventManager()

	// Create a pair of connected net.Conn objects
	client, server := net.Pipe()
	defer client.Close()

	// Create a DecrCommand message
	decrCommandRequest := &dto.DecrCommandRequest{
		Key: "counterKey",
	}

	rpc := &dto.RaftRPC{
		Type: consts.DecrCommand.String(),
		Args: &dto.RaftRPC_DecrCommandRequest{
			DecrCommandRequest: decrCommandRequest,
		},
	}

	// Marshal and send the message
	data, err := proto.Marshal(rpc)
	assert.NoError(t, err)

	// Handle the DecrCommand in a goroutine
	go func() {
		select {
		case event := <-eventManager.DecrCommandRequestChan:
			assert.Equal(t, decrCommandRequest.Key, event.Payload.Key)

			// Send a response
			event.Reply <- &dto.DecrCommandResponse{
				Value: 41,
			}
		case <-time.After(500 * time.Millisecond):
			t.Error("Timeout waiting for DecrCommand")
		}
	}()

	go HandleConnection(server, eventManager)

	_, err = client.Write(data)
	assert.NoError(t, err)

	// Read the response
	buffer := make([]byte, 4096)
	n, err := client.Read(buffer)
	assert.NoError(t, err)

	var response dto.RaftRPC
	err = proto.Unmarshal(buffer[:n], &response)
	assert.NoError(t, err)

	decrResponse := response.GetDecrCommandResponse()
	assert.NotNil(t, decrResponse)
	assert.Equal(t, int32(41), decrResponse.Value)
}

func TestHandleConnection_RemoveCommand(t *testing.T) {
	eventManager := events.NewEventManager()

	// Create a pair of connected net.Conn objects
	client, server := net.Pipe()
	defer client.Close()

	// Create a RemoveCommand message
	removeCommandRequest := &dto.RemoveCommandRequest{
		Key: "testKey",
	}

	rpc := &dto.RaftRPC{
		Type: consts.RemoveCommand.String(),
		Args: &dto.RaftRPC_RemoveCommandRequest{
			RemoveCommandRequest: removeCommandRequest,
		},
	}

	// Marshal and send the message
	data, err := proto.Marshal(rpc)
	assert.NoError(t, err)

	// Handle the RemoveCommand in a goroutine
	go func() {
		select {
		case event := <-eventManager.RemoveCommandRequestChan:
			assert.Equal(t, removeCommandRequest.Key, event.Payload.Key)

			// Send a success response
			event.Reply <- &dto.OkResponse{
				Ok: true,
			}
		case <-time.After(500 * time.Millisecond):
			t.Error("Timeout waiting for RemoveCommand")
		}
	}()

	go HandleConnection(server, eventManager)

	_, err = client.Write(data)
	assert.NoError(t, err)

	// Read the response
	buffer := make([]byte, 4096)
	n, err := client.Read(buffer)
	assert.NoError(t, err)

	var response dto.RaftRPC
	err = proto.Unmarshal(buffer[:n], &response)
	assert.NoError(t, err)

	okResponse := response.GetOkResponse()
	assert.NotNil(t, okResponse)
	assert.True(t, okResponse.Ok)
}
