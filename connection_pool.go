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

func isConnectionAlive(conn net.Conn) bool {
	// Set a deadline for reading
	err := conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	if err != nil {
		fmt.Println("Error setting read deadline:", err)
		return false
	}

	// Try to read a single byte from the connection
	var buf [1]byte
	_, err = conn.Read(buf[:])

	// If we get a timeout error, it means the connection is still alive
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}

	// Any other error likely means the connection is closed
	return err == nil
}

func (cp *ConnectionPool) GetConnection(addr string) (net.Conn, error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	conn, exists := cp.connections[addr]
	if exists {
		slog.Debug("Reusing existing connection", "addr", addr)

		// Check if the existing connection is still valid
		if isConnectionAlive(conn) {
			return conn, nil
		} else {
			slog.Debug("Detected closed connection, removing from pool", "addr", addr)
			delete(cp.connections, addr)
		}
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
