package raft

import (
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"
)

type ConnectionPool struct {
	connections map[string]net.Conn
	mu          sync.Mutex
}

func NewConnectionPool() *ConnectionPool {
	return &ConnectionPool{
		connections: make(map[string]net.Conn),
	}
}

func (cp *ConnectionPool) Close() {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	for _, conn := range cp.connections {
		conn.Close()
	}
	cp.connections = make(map[string]net.Conn)
}

func (cp *ConnectionPool) GetConnection(addr string) (net.Conn, error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	conn, exists := cp.connections[addr]
	if exists {
		// Check if the existing connection is still valid
		err := conn.SetReadDeadline(time.Now().Add(time.Second))
		if err != nil {
			slog.Debug("Error setting read deadline", "error", err)
		}

		_, err = conn.Read(make([]byte, 0))
		if err == nil {
			// Connection is still good, reset the read deadline and return
			conn.SetReadDeadline(time.Time{})
			return conn, nil
		}

		// Connection is dead, close it and remove from the pool
		slog.Debug("Detected closed connection, removing from pool", "addr", addr)
		conn.Close()
		delete(cp.connections, addr)
	}

	// Create a new connection
	newConn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("error connecting to peer: %v", err)
	}

	// Add the new connection to the pool
	cp.connections[addr] = newConn
	return newConn, nil
}
