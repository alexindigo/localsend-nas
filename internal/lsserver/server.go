// Package lsserver runs the node's LocalSend-facing HTTPS endpoint.
// It answers discovery-protocol requests (/info, /register) so the node
// appears in other devices' lists, and politely declines inbound
// transfers: localsend-nas is send-only.
package lsserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/alexindigo/localsend-nas/internal/identity"
	"github.com/alexindigo/localsend-nas/internal/localsend"
)

// Registry records peers that register with us.
type Registry interface {
	Upsert(info localsend.Info, sourceIP string)
}

type Server struct {
	info     localsend.Info
	registry Registry
	http     *http.Server
}

// New builds the server bound to the LocalSend protocol port.
func New(port int, id *identity.Identity, info localsend.Info, registry Registry) *Server {
	s := &Server{info: info, registry: registry}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/localsend/v2/info", s.handleInfo)
	mux.HandleFunc("POST /api/localsend/v2/register", s.handleRegister)
	mux.HandleFunc("POST /api/localsend/v2/prepare-upload", s.handleDeclined)
	mux.HandleFunc("POST /api/localsend/v2/upload", s.handleDeclined)
	mux.HandleFunc("POST /api/localsend/v2/cancel", s.handleCancel)
	s.http = &http.Server{
		Addr:      fmt.Sprintf(":%d", port),
		Handler:   mux,
		TLSConfig: id.TLSConfig(),
	}
	return s
}

// ListenAndServeTLS runs until the server is shut down.
func (s *Server) ListenAndServeTLS() error {
	return s.http.ListenAndServeTLS("", "") // certs come from TLSConfig
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.info)
}

// handleRegister: peers reply to our multicast announcements by POSTing
// their info here; respond with ours (protocol v2 §3).
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var peer localsend.Info
	if err := json.NewDecoder(r.Body).Decode(&peer); err != nil || peer.Fingerprint == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid register body"})
		return
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	s.registry.Upsert(peer, host)
	writeJSON(w, http.StatusOK, s.info)
}

func (s *Server) handleDeclined(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusForbidden, map[string]string{"message": "localsend-nas is send-only"})
}

func (s *Server) handleCancel(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
