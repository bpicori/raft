package core

import (
	"bpicori/raft/pkgs/dto"
	"fmt"
	"log"
	"log/slog"
	"net"

	"google.golang.org/protobuf/proto"
)

func handleConnection(conn net.Conn, server *Server) {
	defer conn.Close()

	// Buffer to read incoming data
	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	if err != nil {
		log.Printf("Error reading from connection: %v", err)
		return
	}

	// Unmarshal the protobuf message
	var rpc dto.RaftRPC // Use the correct type from the generated package
	if err := proto.Unmarshal(buffer[:n], &rpc); err != nil {
		log.Printf("Error unmarshaling protobuf message: %v", err)
		return
	}

	switch rpc.Type {
	case "RequestVoteReq":
		if args := rpc.GetRequestVoteArgs(); args != nil {

			server.eventLoop.requestVoteReqCh <- Event[dto.RequestVoteArgs]{
				Type: RequestVoteReq,
				Data: args,
			}
		}
	case "RequestVoteResp":
		if args := rpc.GetRequestVoteReply(); args != nil {

			server.eventLoop.requestVoteRespCh <- Event[dto.RequestVoteReply]{
				Type: RequestVoteResp,
				Data: args,
			}
		}
	case "AppendEntriesResp":
		if args := rpc.GetAppendEntriesReply(); args != nil {

			server.eventLoop.appendEntriesResCh <- Event[dto.AppendEntriesReply]{
				Type: AppendEntriesResp,
				Data: args,
			}
		}
	case "AppendEntriesReq":
		if args := rpc.GetAppendEntriesArgs(); args != nil {

			if len(args.Entries) > 0 {
				server.eventLoop.appendEntriesReqCh <- Event[dto.AppendEntriesArgs]{
					Type: AppendEntriesReq,
					Data: args,
				}
			} else {
				server.eventLoop.heartbeatReqCh <- Event[dto.AppendEntriesArgs]{
					Type: HeartbeatReq,
					Data: args,
				}
			}
		}
	case "ClusterState":
		clusterState := &dto.ClusterState{
			Leader: server.currentLeader,
		}

		data, err := proto.Marshal(clusterState)
		if err != nil {
			slog.Error("Error marshaling cluster state", "error", err)
			return
		}
    slog.Info("Received ClusterState RPC", "leader", clusterState.Leader)
    fmt.Println(data)

		_, err = conn.Write(append(data, '\n'))
		if err != nil {
			slog.Error("Error sending cluster state", "error", err)
			return
		}

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
