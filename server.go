package raft

import (
	"fmt"
	"log"
	"net"
)

func StartServer(server *Server) {
	listener, err := net.Listen("tcp", server.config.SelfServer.Addr)

	if err != nil {
		panic("cannot start tcp server")
	}

	defer listener.Close()

	log.Printf("Server listening on %s", server.config.SelfServer.Addr)

	for {
		conn, err := listener.Accept()

		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}

		go func(conn net.Conn) {
			defer conn.Close()

			// Read incoming data
			buffer := make([]byte, 1024)
			_, err := conn.Read(buffer)

			if err != nil {
				fmt.Println("Error reading:", err)
				return
			}

			fmt.Println("Received data:", string(buffer))

		}(conn)
	}
}
