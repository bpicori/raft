package raft

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	slog.Debug("[HTTP_SERVER] Status request received")

	status := struct {
		NodeID string `json:"node_id"`
		Role   string `json:"role"`
		Term   int    `json:"term"`
		Leader string `json:"leader_id"`
	}{
		NodeID: s.config.SelfID,
		Role:   roleToString(s.currentRole),
		Term:   s.currentTerm,
		Leader: s.currentLeader,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *Server) RunHTTPServer() {
	http.HandleFunc("/status", s.status)

	httpPort := fmt.Sprintf(":%s", s.config.HttpPort)

	http.ListenAndServe(httpPort, nil)
}