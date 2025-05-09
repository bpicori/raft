package core

import (
	"bpicori/raft/pkgs/consts"
	"bpicori/raft/pkgs/dto"
	"fmt"
	"log/slog"
	"net"

	"google.golang.org/protobuf/proto"
)

func handleConnection(conn net.Conn, server *Server) {
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
			server.eventLoop.voteRequestChan <- Event[dto.VoteRequest]{
				Type: consts.VoteRequest,
				Data: args,
			}
		} else {
			slog.Warn("Received VoteRequest with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case consts.VoteResponse:
		if args := rpc.GetVoteResponse(); args != nil {
			server.eventLoop.voteResponseChan <- Event[dto.VoteResponse]{
				Type: consts.VoteResponse,
				Data: args,
			}
		} else {
			slog.Warn("Received VoteResponse with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case consts.LogRequest:
		if args := rpc.GetLogRequest(); args != nil {
			server.eventLoop.logRequestChan <- Event[dto.LogRequest]{
				Type: consts.LogRequest,
				Data: args,
			}
		} else {
			slog.Warn("Received LogRequest with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case consts.LogResponse:
		if args := rpc.GetLogResponse(); args != nil {
			server.eventLoop.logResponseChan <- Event[dto.LogResponse]{
				Type: consts.LogResponse,
				Data: args,
			}
		} else {
			slog.Warn("Received LogResponse with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case consts.NodeStatus:
		nodeStatus := &dto.NodeStatus{
			NodeId:        server.config.SelfID,
			CurrentTerm:   server.currentTerm,
			VotedFor:      server.votedFor,
			CurrentRole:   consts.MapRoleToString(server.currentRole),
			CurrentLeader: server.currentLeader,
			CommitLength:  server.commitLength,
			LogEntries:    server.logEntry,
		}

		data, err := proto.Marshal(nodeStatus)
		if err != nil {
			slog.Error("Error marshaling node status for response", "error", err, "remote_addr", conn.RemoteAddr())
			return
		}

		_, err = conn.Write(data)
		if err != nil {
			slog.Error("Error sending cluster state response", "error", err, "remote_addr", conn.RemoteAddr())
		}
	case consts.SetCommand:
		if args := rpc.GetSetCommand(); args != nil {
			if server.currentRole != Leader {
				slog.Warn("Received SetCommand from non-leader", "remote_addr", conn.RemoteAddr())
				return
			}
			server.eventLoop.setCommandChan <- Event[dto.SetCommand]{
				Type: consts.SetCommand,
				Data: args,
			}

			// return ok response
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
				return
			}
		} else {
			slog.Warn("Received SetCommand with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	default:
		slog.Error("Unhandled RaftRPCType enum value in switch", "rpcType", rpcType, "remote_addr", conn.RemoteAddr())
	}
}

func (s *Server) RunTcp() {
	defer s.wg.Done()

	listener, err := net.Listen("tcp", s.config.SelfServer.Addr)

	if err != nil {
		slog.Error("[TCP_SERVER] Error starting TCP server", "error", err)
		panic("cannot start tcp server")
	}

	defer listener.Close()

	go func() {
		<-s.ctx.Done()
		slog.Info("[TCP_SERVER] Gracefully shutting down TCP server")
		listener.Close()
	}()

	slog.Info("[TCP_SERVER] Server started", "address", s.config.SelfServer.Addr)

	for {
		select {
		case <-s.ctx.Done():
			slog.Info("[TCP_SERVER] Shutting down TCP server")
			return
		default:
			conn, err := listener.Accept()

			if err != nil {
				slog.Info("[TCP_SERVER] Server stopped accepting connections", "error", err)
				return
			}

			go handleConnection(conn, s)
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

	// encode the message
	data, err := proto.Marshal(message)
	if err != nil {
		slog.Debug("[SEND_ASYNC_RPC] Error marshaling message", "error", err, "message", message, "addr", addr)
		return fmt.Errorf("error marshaling message: %v", err)
	}

	// send the message
	_, err = conn.Write(data)
	if err != nil {
		slog.Debug("[SEND_ASYNC_RPC] Error sending data", "error", err, "message", message, "addr", addr)
		return fmt.Errorf("error sending data: %v", err)
	}

	return nil
}
