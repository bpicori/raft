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

// Start starts and manages the TCP server lifecycle.
func Start(addr string, eventManager *events.EventManager, ctx context.Context, wg *sync.WaitGroup) {

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		slog.Error("[TCP_SERVER] Error starting TCP server", "error", err, "address", addr)
		panic(fmt.Errorf("cannot start tcp server on %s: %w", addr, err))
	}
	defer listener.Close()

	go func() {
		<-ctx.Done()
		slog.Info("[TCP_SERVER] Context done, shutting down TCP listener", "address", addr)
		listener.Close() // This will cause listener.Accept() to return an error.
		wg.Done()
	}()

	slog.Info("[TCP_SERVER] Server started", "address", addr)

	for {
		select {
		case <-ctx.Done():
			slog.Info("[TCP_SERVER] Context cancelled, server loop shutting down", "address", addr)
			wg.Done()
			return
		default:
			conn, err := listener.Accept() // Accept will block until a new connection or an error
			if err != nil {
				select {
				case <-ctx.Done():
					slog.Info("[TCP_SERVER] Listener closed as part of shutdown", "address", addr)
				default:
					slog.Error("[TCP_SERVER] Error accepting connection", "error", err, "address", addr)
				}
				return
			}
			go HandleConnection(conn, eventManager)
		}
	}
}

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
			eventManager.VoteRequestChan <- args
		} else {
			slog.Warn("Received VoteRequest with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case consts.VoteResponse:
		if args := rpc.GetVoteResponse(); args != nil {
			eventManager.VoteResponseChan <- args
		} else {
			slog.Warn("Received VoteResponse with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case consts.LogRequest:
		if args := rpc.GetLogRequest(); args != nil {
			eventManager.LogRequestChan <- args
		} else {
			slog.Warn("Received LogRequest with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case consts.LogResponse:
		if args := rpc.GetLogResponse(); args != nil {
			eventManager.LogResponseChan <- args
		} else {
			slog.Warn("Received LogResponse with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case consts.NodeStatus:
		if args := rpc.GetNodeStatusRequest(); args != nil {
			ch := make(chan *dto.NodeStatusResponse)
			eventManager.NodeStatusChan <- events.NodeStatusEvent{
				Reply: ch,
			}

			response := <-ch
			rpcResponse := &dto.RaftRPC{
				Type: consts.NodeStatus.String(),
				Args: &dto.RaftRPC_NodeStatusResponse{
					NodeStatusResponse: response,
				},
			}
			sendResponse(conn, rpcResponse)
		} else {
			slog.Warn("Received NodeStatus with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case consts.SetCommand:
		if args := rpc.GetSetCommandRequest(); args != nil {
			replyCh := make(chan *dto.OkResponse)
			eventManager.SetCommandRequestChan <- events.SetCommandEvent{
				Payload: args,
				Reply:   replyCh,
			}

			select {
			case response := <-replyCh:
				rpcResponse := &dto.RaftRPC{
					Type: consts.OkResponse.String(),
					Args: &dto.RaftRPC_OkResponse{
						OkResponse: response,
					},
				}
				sendResponse(conn, rpcResponse)
			}
		} else {
			slog.Warn("Received SetCommand with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case consts.GetCommand:
		if args := rpc.GetGetCommandRequest(); args != nil {
			replyCh := make(chan *dto.GetCommandResponse)
			eventManager.GetCommandRequestChan <- events.GetCommandEvent{
				Payload: args,
				Reply:   replyCh,
			}

			select {
			case response := <-replyCh:
				rpcResponse := &dto.RaftRPC{
					Type: consts.GetCommand.String(),
					Args: &dto.RaftRPC_GetCommandResponse{
						GetCommandResponse: response,
					},
				}
				sendResponse(conn, rpcResponse)
			}
		} else {
			slog.Warn("Received GetCommand with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case consts.IncrCommand:
		// TODO: Implement IncrCommand
		if args := rpc.GetIncrCommandRequest(); args != nil {
			replyCh := make(chan *dto.IncrCommandResponse)
			eventManager.IncrCommandRequestChan <- events.IncrCommandEvent{
				Payload: args,
				Reply:   replyCh,
			}

			select {
			case response := <-replyCh:
				rpcResponse := &dto.RaftRPC{
					Type: consts.IncrCommand.String(),
					Args: &dto.RaftRPC_IncrCommandResponse{
						IncrCommandResponse: response,
					},
				}
				sendResponse(conn, rpcResponse)
			}
		} else {
			slog.Warn("Received IncrCommand with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case consts.DecrCommand:
		if args := rpc.GetDecrCommandRequest(); args != nil {
			replyCh := make(chan *dto.DecrCommandResponse)
			eventManager.DecrCommandRequestChan <- events.DecrCommandEvent{
				Payload: args,
				Reply:   replyCh,
			}

			select {
			case response := <-replyCh:
				rpcResponse := &dto.RaftRPC{
					Type: consts.DecrCommand.String(),
					Args: &dto.RaftRPC_DecrCommandResponse{
						DecrCommandResponse: response,
					},
				}
				sendResponse(conn, rpcResponse)
			}
		} else {
			slog.Warn("Received DecrCommand with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case consts.RemoveCommand:
		if args := rpc.GetRemoveCommandRequest(); args != nil {
			replyCh := make(chan *dto.OkResponse)
			eventManager.RemoveCommandRequestChan <- events.RemoveCommandEvent{
				Payload: args,
				Reply:   replyCh,
			}

			select {
			case response := <-replyCh:
				rpcResponse := &dto.RaftRPC{
					Type: consts.RemoveCommand.String(),
					Args: &dto.RaftRPC_OkResponse{
						OkResponse: response,
					},
				}
				sendResponse(conn, rpcResponse)
			}
		} else {
			slog.Warn("Received RemoveCommand with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	default:
		slog.Error("Unhandled RaftRPCType enum value in switch", "rpcType", rpcType, "remote_addr", conn.RemoteAddr())
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

// sendResponse marshals and sends an RPC response to the client.
func sendResponse(conn net.Conn, rpcResponse *dto.RaftRPC) {
	data, err := proto.Marshal(rpcResponse)
	if err != nil {
		slog.Error("Error marshaling response", "error", err, "remote_addr", conn.RemoteAddr())
		return
	}
	_, err = conn.Write(data)
	if err != nil {
		slog.Error("Error sending response", "error", err, "remote_addr", conn.RemoteAddr())
	}
}
