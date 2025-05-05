package core

import (
	"bpicori/raft/pkgs/dto"
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
		slog.Error("Error unmarshaling protobuf message", "remote_addr", conn.RemoteAddr(), "error", err)
		return
	}

	rpcType, err := mapStringToRPCType(rpc.Type)
	if err != nil {
		slog.Error("Received unknown RPC type", "type", rpc.Type, "remote_addr", conn.RemoteAddr(), "error", err)
		return
	}

	switch rpcType {
	case VoteRequest:
		if args := rpc.GetVoteRequest(); args != nil {
			server.eventLoop.voteRequestChan <- Event[dto.VoteRequest]{
				Type: VoteRequest,
				Data: args,
			}
		} else {
			slog.Warn("Received RequestVoteReq with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case VoteResponse:
		if args := rpc.GetVoteResponse(); args != nil {
			server.eventLoop.voteResponseChan <- Event[dto.VoteResponse]{
				Type: VoteResponse,
				Data: args,
			}
		} else {
			slog.Warn("Received RequestVoteResp with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case LogRequest:
		if args := rpc.GetLogRequest(); args != nil {
			server.eventLoop.logRequestChan <- Event[dto.LogRequest]{
				Type: LogRequest,
				Data: args,
			}
		} else {
			slog.Warn("Received AppendEntriesReq/Heartbeat with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case LogResponse:
		if args := rpc.GetLogResponse(); args != nil {
			server.eventLoop.logResponseChan <- Event[dto.LogResponse]{
				Type: LogResponse,
				Data: args,
			}
		} else {
			slog.Warn("Received AppendEntriesResp with nil args", "rpcType", rpcType.String(), "remote_addr", conn.RemoteAddr())
		}
	case ClusterStateType:
		clusterState := &dto.ClusterState{
			Leader: server.currentLeader,
		}

		data, err := proto.Marshal(clusterState)
		if err != nil {
			slog.Error("Error marshaling cluster state for response", "error", err, "remote_addr", conn.RemoteAddr())
			return
		}
		slog.Info("Responding to ClusterState RPC", "leader", clusterState.Leader, "remote_addr", conn.RemoteAddr())

		_, err = conn.Write(append(data, '\n'))
		if err != nil {
			slog.Error("Error sending cluster state response", "error", err, "remote_addr", conn.RemoteAddr())
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
