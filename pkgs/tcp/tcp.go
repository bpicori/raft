package tcp

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/events"

	"google.golang.org/protobuf/proto"
)

func HandleConnection(conn net.Conn, eventManager *events.EventManager) {
	defer conn.Close()

	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	if err != nil {
		slog.Error("Error reading from connection", "remote_addr", conn.RemoteAddr(), "error", err)
		return
	}

	var rpc dto.RaftRPC
	if err := proto.Unmarshal(buffer[:n], &rpc); err != nil {
		slog.Error("Error unmarshaling protobuf message", "remote_addr", conn.RemoteAddr(), "error", err, "buffer", buffer[:n])
		return
	}

	rpcType, err := consts.MapStringToRPCType(rpc.Type)
	if err != nil {
		slog.Error("Received unknown RPC type", "type", rpc.Type, "remote_addr", conn.RemoteAddr(), "error", err)
		return
	}

	switch rpcType {
	case consts.VoteRequest:
		if args := rpc.GetVoteRequest(); args != nil {
			eventManager.VoteRequestChan <- *args
		} else {
			slog.Warn("Received VoteRequest with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case consts.VoteResponse:
		if args := rpc.GetVoteResponse(); args != nil {
			eventManager.VoteResponseChan <- *args
		} else {
			slog.Warn("Received VoteResponse with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case consts.LogRequest:
		if args := rpc.GetLogRequest(); args != nil {
			eventManager.LogRequestChan <- *args
		} else {
			slog.Warn("Received LogRequest with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case consts.LogResponse:
		if args := rpc.GetLogResponse(); args != nil {
			eventManager.LogResponseChan <- *args
		} else {
			slog.Warn("Received LogResponse with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	// case consts.NodeStatus:
	// 	nodeStatus := &dto.NodeStatus{
	// 		NodeId:        params.NodeID,
	// 		CurrentTerm:   int32(params.CurrentTerm), // Added int32 conversion
	// 		VotedFor:      params.VotedFor,
	// 		CurrentRole:   consts.MapRoleToString(params.CurrentRole), // Ensure params.CurrentRole type matches MapRoleToString input
	// 		CurrentLeader: params.CurrentLeader,
	// 		CommitLength:  int32(params.CommitLength), // Added int32 conversion
	// 		LogEntries:    params.LogEntries,
	// 	}

	// 	rpcResponse := &dto.RaftRPC{
	// 		Type: consts.NodeStatus.String(),
	// 		Args: &dto.RaftRPC_NodeStatus{NodeStatus: nodeStatus},
	// 	}

	// 	data, err := proto.Marshal(rpcResponse)
	// 	if err != nil {
	// 		slog.Error("Error marshaling RaftRPC response for NodeStatus", "error", err, "remote_addr", conn.RemoteAddr())
	// 		return
	// 	}

	// 	_, err = conn.Write(data)
	// 	if err != nil {
	// 		slog.Error("Error sending cluster state response", "error", err, "remote_addr", conn.RemoteAddr())
	// 	}
	case consts.SetCommand:
		if args := rpc.GetSetCommand(); args != nil {
			eventManager.SetCommandChan <- *args

			okResponse := &dto.RaftRPC{
				Type: consts.OkResponse.String(),
				Args: &dto.RaftRPC_OkResponse{
					OkResponse: &dto.OkResponse{
						Ok: true,
					},
				},
			}

			data, err := proto.Marshal(okResponse)
			if err != nil {
				slog.Error("Error marshaling ok response", "error", err, "remote_addr", conn.RemoteAddr())
				return
			}

			_, err = conn.Write(data)
			if err != nil {
				slog.Error("Error sending ok response", "error", err, "remote_addr", conn.RemoteAddr())
			}
		} else {
			slog.Warn("Received SetCommand with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	default:
		slog.Error("Unhandled RaftRPCType enum value in switch", "rpcType", rpcType, "remote_addr", conn.RemoteAddr())
	}
}

// RunTCPServerParams holds parameters for RunTCPServer.
type RunTCPServerParams struct {
	Wg           *sync.WaitGroup
	Ctx          context.Context
	ListenAddr   string
	EventManager *events.EventManager
}

// RunTCPServer starts and manages the TCP server lifecycle.
func RunTCPServer(params RunTCPServerParams) {
	if params.Wg != nil {
		defer params.Wg.Done()
	}

	listener, err := net.Listen("tcp", params.ListenAddr)
	if err != nil {
		slog.Error("[TCP_SERVER] Error starting TCP server", "error", err, "address", params.ListenAddr)
		panic(fmt.Errorf("cannot start tcp server on %s: %w", params.ListenAddr, err))
	}
	defer listener.Close()

	go func() {
		<-params.Ctx.Done()
		slog.Info("[TCP_SERVER] Context done, shutting down TCP listener", "address", params.ListenAddr)
		listener.Close() // This will cause listener.Accept() to return an error.
	}()

	slog.Info("[TCP_SERVER] Server started", "address", params.ListenAddr)

	for {
		select {
		case <-params.Ctx.Done():
			slog.Info("[TCP_SERVER] Context cancelled, server loop shutting down", "address", params.ListenAddr)
			return
		default:
			conn, err := listener.Accept() // Accept will block until a new connection or an error
			if err != nil {
				select {
				case <-params.Ctx.Done():
					slog.Info("[TCP_SERVER] Listener closed as part of shutdown", "address", params.ListenAddr)
				default:
					slog.Error("[TCP_SERVER] Error accepting connection", "error", err, "address", params.ListenAddr)
				}
				return
			}
			go HandleConnection(conn, params.EventManager)
		}
	}
}

// SendAsyncRPC sends a protobuf message to a connection asynchronously.
func SendAsyncRPC(addr string, message *dto.RaftRPC) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("error getting connection: %v", err)
	}
	defer conn.Close()

	data, err := proto.Marshal(message)
	if err != nil {
		slog.Debug("[SEND_ASYNC_RPC] Error marshaling message", "error", err, "message", message, "addr", addr)
		return fmt.Errorf("error marshaling message: %v", err)
	}

	_, err = conn.Write(data)
	if err != nil {
		slog.Debug("[SEND_ASYNC_RPC] Error sending data", "error", err, "message", message, "addr", addr)
		return fmt.Errorf("error sending data: %v", err)
	}

	return nil
}

// SendSyncRPC sends a protobuf message and waits for a response.
func SendSyncRPC(addr string, request *dto.RaftRPC) (*dto.RaftRPC, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("error getting connection: %v", err)
	}
	defer conn.Close()

	data, err := proto.Marshal(request)
	if err != nil {
		slog.Debug("[SEND_SYNC_RPC] Error marshaling request", "error", err, "request", request, "addr", addr)
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	_, err = conn.Write(data)
	if err != nil {
		slog.Debug("[SEND_SYNC_RPC] Error sending request", "error", err, "request", request, "addr", addr)
		return nil, fmt.Errorf("error sending request: %v", err)
	}

	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	if err != nil {
		slog.Debug("[SEND_SYNC_RPC] Error receiving response", "error", err, "request", request, "addr", addr)
		return nil, fmt.Errorf("error receiving response: %v", err)
	}

	var response dto.RaftRPC
	if err := proto.Unmarshal(buffer[:n], &response); err != nil {
		slog.Debug("[SEND_SYNC_RPC] Error unmarshaling response", "error", err, "request", request, "addr", addr)
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &response, nil
}
