package tcp

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/events"

	"google.golang.org/protobuf/proto"
)

var TCP_TIMEOUT = 5 * time.Second

// Start starts and manages the TCP server lifecycle.
func Start(addr string, eventManager *events.EventManager, ctx context.Context) {

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
	}()

	slog.Info("[TCP_SERVER] Server started", "address", addr)

	for {
		select {
		case <-ctx.Done():
			slog.Info("[TCP_SERVER] Context cancelled, server loop shutting down", "address", addr)
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
			replyCh := make(chan *dto.GenericResponse)
			eventManager.SetCommandRequestChan <- events.SetCommandEvent{
				Payload: args,
				Reply:   replyCh,
			}

			select {
			case response := <-replyCh:
				rpcResponse := &dto.RaftRPC{
					Type: consts.GenericResponse.String(),
					Args: &dto.RaftRPC_GenericResponse{
						GenericResponse: response,
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
			case <-time.After(TCP_TIMEOUT):
				slog.Warn("Received GetCommand with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
				sendResponse(conn, &dto.RaftRPC{
					Type: consts.GetCommand.String(),
					Args: &dto.RaftRPC_GetCommandResponse{
						GetCommandResponse: &dto.GetCommandResponse{
							Value: "",
							Error: "Timeout",
						},
					},
				})
			}
		} else {
			slog.Warn("Received GetCommand with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case consts.IncrCommand:
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
			case <-time.After(TCP_TIMEOUT):
				slog.Warn("Received IncrCommand with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
				sendResponse(conn, &dto.RaftRPC{
					Type: consts.IncrCommand.String(),
					Args: &dto.RaftRPC_IncrCommandResponse{
						IncrCommandResponse: &dto.IncrCommandResponse{
							Value: 0,
							Error: "Timeout",
						},
					},
				})
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
			case <-time.After(TCP_TIMEOUT):
				slog.Warn("Received DecrCommand with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
				sendResponse(conn, &dto.RaftRPC{
					Type: consts.DecrCommand.String(),
					Args: &dto.RaftRPC_DecrCommandResponse{
						DecrCommandResponse: &dto.DecrCommandResponse{
							Value: 0,
							Error: "Timeout",
						},
					},
				})
			}
		} else {
			slog.Warn("Received DecrCommand with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case consts.RemoveCommand:
		if args := rpc.GetRemoveCommandRequest(); args != nil {
			replyCh := make(chan *dto.GenericResponse)
			eventManager.RemoveCommandRequestChan <- events.RemoveCommandEvent{
				Payload: args,
				Reply:   replyCh,
			}

			select {
			case response := <-replyCh:
				rpcResponse := &dto.RaftRPC{
					Type: consts.RemoveCommand.String(),
					Args: &dto.RaftRPC_GenericResponse{
						GenericResponse: response,
					},
				}
				sendResponse(conn, rpcResponse)
			case <-time.After(TCP_TIMEOUT):
				slog.Warn("Received RemoveCommand with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
				sendResponse(conn, &dto.RaftRPC{
					Type: consts.RemoveCommand.String(),
					Args: &dto.RaftRPC_GenericResponse{
						GenericResponse: &dto.GenericResponse{
							Ok: false,
						},
					},
				})
			}
		} else {
			slog.Warn("Received RemoveCommand with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case consts.LpushCommand:
		if args := rpc.GetLpushCommandRequest(); args != nil {
			replyCh := make(chan *dto.LpushCommandResponse)
			eventManager.LpushCommandRequestChan <- events.LpushCommandEvent{
				Payload: args,
				Reply:   replyCh,
			}

			select {
			case response := <-replyCh:
				rpcResponse := &dto.RaftRPC{
					Type: consts.LpushCommand.String(),
					Args: &dto.RaftRPC_LpushCommandResponse{
						LpushCommandResponse: response,
					},
				}
				sendResponse(conn, rpcResponse)
			case <-time.After(TCP_TIMEOUT):
				slog.Warn("Received LpushCommand timeout", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
				sendResponse(conn, &dto.RaftRPC{
					Type: consts.LpushCommand.String(),
					Args: &dto.RaftRPC_LpushCommandResponse{
						LpushCommandResponse: &dto.LpushCommandResponse{
							Length: 0,
							Error:  "Timeout",
						},
					},
				})
			}
		} else {
			slog.Warn("Received LpushCommand with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case consts.LpopCommand:
		if args := rpc.GetLpopCommandRequest(); args != nil {
			replyCh := make(chan *dto.LpopCommandResponse)
			eventManager.LpopCommandRequestChan <- events.LpopCommandEvent{
				Payload: args,
				Reply:   replyCh,
			}

			select {
			case response := <-replyCh:
				rpcResponse := &dto.RaftRPC{
					Type: consts.LpopCommand.String(),
					Args: &dto.RaftRPC_LpopCommandResponse{
						LpopCommandResponse: response,
					},
				}
				sendResponse(conn, rpcResponse)
			case <-time.After(TCP_TIMEOUT):
				slog.Warn("Received LpopCommand timeout", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
				sendResponse(conn, &dto.RaftRPC{
					Type: consts.LpopCommand.String(),
					Args: &dto.RaftRPC_LpopCommandResponse{
						LpopCommandResponse: &dto.LpopCommandResponse{
							Element: "",
							Error:   "Timeout",
						},
					},
				})
			}
		} else {
			slog.Warn("Received LpopCommand with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case consts.LindexCommand:
		if args := rpc.GetLindexCommandRequest(); args != nil {
			replyCh := make(chan *dto.LindexCommandResponse)
			eventManager.LindexCommandRequestChan <- events.LindexCommandEvent{
				Payload: args,
				Reply:   replyCh,
			}

			select {
			case response := <-replyCh:
				rpcResponse := &dto.RaftRPC{
					Type: consts.LindexCommand.String(),
					Args: &dto.RaftRPC_LindexCommandResponse{
						LindexCommandResponse: response,
					},
				}
				sendResponse(conn, rpcResponse)
			case <-time.After(TCP_TIMEOUT):
				slog.Warn("Received LindexCommand timeout", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
				sendResponse(conn, &dto.RaftRPC{
					Type: consts.LindexCommand.String(),
					Args: &dto.RaftRPC_LindexCommandResponse{
						LindexCommandResponse: &dto.LindexCommandResponse{
							Element: "",
							Error:   "Timeout",
						},
					},
				})
			}
		} else {
			slog.Warn("Received LindexCommand with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case consts.LlenCommand:
		if args := rpc.GetLlenCommandRequest(); args != nil {
			replyCh := make(chan *dto.LlenCommandResponse)
			eventManager.LlenCommandRequestChan <- events.LlenCommandEvent{
				Payload: args,
				Reply:   replyCh,
			}

			select {
			case response := <-replyCh:
				rpcResponse := &dto.RaftRPC{
					Type: consts.LlenCommand.String(),
					Args: &dto.RaftRPC_LlenCommandResponse{
						LlenCommandResponse: response,
					},
				}
				sendResponse(conn, rpcResponse)
			case <-time.After(TCP_TIMEOUT):
				slog.Warn("Received LlenCommand timeout", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
				sendResponse(conn, &dto.RaftRPC{
					Type: consts.LlenCommand.String(),
					Args: &dto.RaftRPC_LlenCommandResponse{
						LlenCommandResponse: &dto.LlenCommandResponse{
							Length: 0,
							Error:  "Timeout",
						},
					},
				})
			}
		} else {
			slog.Warn("Received LlenCommand with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case consts.KeysCommand:
		if args := rpc.GetKeysCommandRequest(); args != nil {
			replyCh := make(chan *dto.KeysCommandResponse)
			eventManager.KeysCommandRequestChan <- events.KeysCommandEvent{
				Payload: args,
				Reply:   replyCh,
			}

			select {
			case response := <-replyCh:
				rpcResponse := &dto.RaftRPC{
					Type: consts.KeysCommand.String(),
					Args: &dto.RaftRPC_KeysCommandResponse{
						KeysCommandResponse: response,
					},
				}
				sendResponse(conn, rpcResponse)
			case <-time.After(TCP_TIMEOUT):
				slog.Warn("Received KeysCommand timeout", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
				sendResponse(conn, &dto.RaftRPC{
					Type: consts.KeysCommand.String(),
					Args: &dto.RaftRPC_KeysCommandResponse{
						KeysCommandResponse: &dto.KeysCommandResponse{
							Keys:  []string{},
							Error: "Timeout",
						},
					},
				})
			}
		} else {
			slog.Warn("Received KeysCommand with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case consts.SaddCommand:
		if args := rpc.GetSaddCommandRequest(); args != nil {
			replyCh := make(chan *dto.SaddCommandResponse)
			eventManager.SaddCommandRequestChan <- events.SaddCommandEvent{
				Payload: args,
				Reply:   replyCh,
			}

			select {
			case response := <-replyCh:
				rpcResponse := &dto.RaftRPC{
					Type: consts.SaddCommand.String(),
					Args: &dto.RaftRPC_SaddCommandResponse{
						SaddCommandResponse: response,
					},
				}
				sendResponse(conn, rpcResponse)
			case <-time.After(TCP_TIMEOUT):
				slog.Warn("Timeout waiting for SaddCommand response", "remote_addr", conn.RemoteAddr())
				sendResponse(conn, &dto.RaftRPC{
					Type: consts.SaddCommand.String(),
					Args: &dto.RaftRPC_SaddCommandResponse{
						SaddCommandResponse: &dto.SaddCommandResponse{
							Added: 0,
							Error: "Timeout",
						},
					},
				})
			}
		} else {
			slog.Warn("Received SaddCommand with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case consts.SremCommand:
		if args := rpc.GetSremCommandRequest(); args != nil {
			replyCh := make(chan *dto.SremCommandResponse)
			eventManager.SremCommandRequestChan <- events.SremCommandEvent{
				Payload: args,
				Reply:   replyCh,
			}

			select {
			case response := <-replyCh:
				rpcResponse := &dto.RaftRPC{
					Type: consts.SremCommand.String(),
					Args: &dto.RaftRPC_SremCommandResponse{
						SremCommandResponse: response,
					},
				}
				sendResponse(conn, rpcResponse)
			case <-time.After(TCP_TIMEOUT):
				slog.Warn("Timeout waiting for SremCommand response", "remote_addr", conn.RemoteAddr())
				sendResponse(conn, &dto.RaftRPC{
					Type: consts.SremCommand.String(),
					Args: &dto.RaftRPC_SremCommandResponse{
						SremCommandResponse: &dto.SremCommandResponse{
							Removed: 0,
							Error:   "Timeout",
						},
					},
				})
			}
		} else {
			slog.Warn("Received SremCommand with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case consts.SismemberCommand:
		if args := rpc.GetSismemberCommandRequest(); args != nil {
			replyCh := make(chan *dto.SismemberCommandResponse)
			eventManager.SismemberCommandRequestChan <- events.SismemberCommandEvent{
				Payload: args,
				Reply:   replyCh,
			}

			select {
			case response := <-replyCh:
				rpcResponse := &dto.RaftRPC{
					Type: consts.SismemberCommand.String(),
					Args: &dto.RaftRPC_SismemberCommandResponse{
						SismemberCommandResponse: response,
					},
				}
				sendResponse(conn, rpcResponse)
			case <-time.After(TCP_TIMEOUT):
				slog.Warn("Timeout waiting for SismemberCommand response", "remote_addr", conn.RemoteAddr())
				sendResponse(conn, &dto.RaftRPC{
					Type: consts.SismemberCommand.String(),
					Args: &dto.RaftRPC_SismemberCommandResponse{
						SismemberCommandResponse: &dto.SismemberCommandResponse{
							IsMember: 0,
							Error:    "Timeout",
						},
					},
				})
			}
		} else {
			slog.Warn("Received SismemberCommand with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case consts.SinterCommand:
		if args := rpc.GetSinterCommandRequest(); args != nil {
			replyCh := make(chan *dto.SinterCommandResponse)
			eventManager.SinterCommandRequestChan <- events.SinterCommandEvent{
				Payload: args,
				Reply:   replyCh,
			}

			select {
			case response := <-replyCh:
				rpcResponse := &dto.RaftRPC{
					Type: consts.SinterCommand.String(),
					Args: &dto.RaftRPC_SinterCommandResponse{
						SinterCommandResponse: response,
					},
				}
				sendResponse(conn, rpcResponse)
			case <-time.After(TCP_TIMEOUT):
				slog.Warn("Timeout waiting for SinterCommand response", "remote_addr", conn.RemoteAddr())
				sendResponse(conn, &dto.RaftRPC{
					Type: consts.SinterCommand.String(),
					Args: &dto.RaftRPC_SinterCommandResponse{
						SinterCommandResponse: &dto.SinterCommandResponse{
							Members: []string{},
							Error:   "Timeout",
						},
					},
				})
			}
		} else {
			slog.Warn("Received SinterCommand with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case consts.ScardCommand:
		if args := rpc.GetScardCommandRequest(); args != nil {
			replyCh := make(chan *dto.ScardCommandResponse)
			eventManager.ScardCommandRequestChan <- events.ScardCommandEvent{
				Payload: args,
				Reply:   replyCh,
			}

			select {
			case response := <-replyCh:
				rpcResponse := &dto.RaftRPC{
					Type: consts.ScardCommand.String(),
					Args: &dto.RaftRPC_ScardCommandResponse{
						ScardCommandResponse: response,
					},
				}
				sendResponse(conn, rpcResponse)
			case <-time.After(TCP_TIMEOUT):
				slog.Warn("Timeout waiting for ScardCommand response", "remote_addr", conn.RemoteAddr())
				sendResponse(conn, &dto.RaftRPC{
					Type: consts.ScardCommand.String(),
					Args: &dto.RaftRPC_ScardCommandResponse{
						ScardCommandResponse: &dto.ScardCommandResponse{
							Cardinality: 0,
							Error:       "Timeout",
						},
					},
				})
			}
		} else {
			slog.Warn("Received ScardCommand with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case consts.HsetCommand:
		if args := rpc.GetHsetCommandRequest(); args != nil {
			replyCh := make(chan *dto.HsetCommandResponse)
			eventManager.HsetCommandRequestChan <- events.HsetCommandEvent{
				Payload: args,
				Reply:   replyCh,
			}

			select {
			case response := <-replyCh:
				rpcResponse := &dto.RaftRPC{
					Type: consts.HsetCommand.String(),
					Args: &dto.RaftRPC_HsetCommandResponse{
						HsetCommandResponse: response,
					},
				}
				sendResponse(conn, rpcResponse)
			case <-time.After(TCP_TIMEOUT):
				slog.Warn("Timeout waiting for HsetCommand response", "remote_addr", conn.RemoteAddr())
				sendResponse(conn, &dto.RaftRPC{
					Type: consts.HsetCommand.String(),
					Args: &dto.RaftRPC_HsetCommandResponse{
						HsetCommandResponse: &dto.HsetCommandResponse{
							Added: 0,
							Error: "Timeout",
						},
					},
				})
			}
		} else {
			slog.Warn("Received HsetCommand with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case consts.HgetCommand:
		if args := rpc.GetHgetCommandRequest(); args != nil {
			replyCh := make(chan *dto.HgetCommandResponse)
			eventManager.HgetCommandRequestChan <- events.HgetCommandEvent{
				Payload: args,
				Reply:   replyCh,
			}

			select {
			case response := <-replyCh:
				rpcResponse := &dto.RaftRPC{
					Type: consts.HgetCommand.String(),
					Args: &dto.RaftRPC_HgetCommandResponse{
						HgetCommandResponse: response,
					},
				}
				sendResponse(conn, rpcResponse)
			case <-time.After(TCP_TIMEOUT):
				slog.Warn("Timeout waiting for HgetCommand response", "remote_addr", conn.RemoteAddr())
				sendResponse(conn, &dto.RaftRPC{
					Type: consts.HgetCommand.String(),
					Args: &dto.RaftRPC_HgetCommandResponse{
						HgetCommandResponse: &dto.HgetCommandResponse{
							Value: "",
							Error: "Timeout",
						},
					},
				})
			}
		} else {
			slog.Warn("Received HgetCommand with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case consts.HmgetCommand:
		if args := rpc.GetHmgetCommandRequest(); args != nil {
			replyCh := make(chan *dto.HmgetCommandResponse)
			eventManager.HmgetCommandRequestChan <- events.HmgetCommandEvent{
				Payload: args,
				Reply:   replyCh,
			}

			select {
			case response := <-replyCh:
				rpcResponse := &dto.RaftRPC{
					Type: consts.HmgetCommand.String(),
					Args: &dto.RaftRPC_HmgetCommandResponse{
						HmgetCommandResponse: response,
					},
				}
				sendResponse(conn, rpcResponse)
			case <-time.After(TCP_TIMEOUT):
				slog.Warn("Timeout waiting for HmgetCommand response", "remote_addr", conn.RemoteAddr())
				sendResponse(conn, &dto.RaftRPC{
					Type: consts.HmgetCommand.String(),
					Args: &dto.RaftRPC_HmgetCommandResponse{
						HmgetCommandResponse: &dto.HmgetCommandResponse{
							Values: []string{},
							Error:  "Timeout",
						},
					},
				})
			}
		} else {
			slog.Warn("Received HmgetCommand with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case consts.HincrbyCommand:
		if args := rpc.GetHincrbyCommandRequest(); args != nil {
			replyCh := make(chan *dto.HincrbyCommandResponse)
			eventManager.HincrbyCommandRequestChan <- events.HincrbyCommandEvent{
				Payload: args,
				Reply:   replyCh,
			}

			select {
			case response := <-replyCh:
				rpcResponse := &dto.RaftRPC{
					Type: consts.HincrbyCommand.String(),
					Args: &dto.RaftRPC_HincrbyCommandResponse{
						HincrbyCommandResponse: response,
					},
				}
				sendResponse(conn, rpcResponse)
			case <-time.After(TCP_TIMEOUT):
				slog.Warn("Timeout waiting for HincrbyCommand response", "remote_addr", conn.RemoteAddr())
				sendResponse(conn, &dto.RaftRPC{
					Type: consts.HincrbyCommand.String(),
					Args: &dto.RaftRPC_HincrbyCommandResponse{
						HincrbyCommandResponse: &dto.HincrbyCommandResponse{
							Value: 0,
							Error: "Timeout",
						},
					},
				})
			}
		} else {
			slog.Warn("Received HincrbyCommand with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
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
