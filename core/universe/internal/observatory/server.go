package observatory

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	agentpkg "spwn.sh/core/agent"
	"spwn.sh/core/universe/internal/state"
)

// Server serves the Observatory HTTP API.
type Server struct {
	state *state.Store
	addr  string
	srv   *http.Server
}

// New creates an Observatory server.
func New(s *state.Store, addr string) *Server {
	return &Server{state: s, addr: addr}
}

// Start begins serving the API.
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/universes", s.handleListUniverses)
	mux.HandleFunc("GET /api/agents", s.handleListAgents)
	mux.HandleFunc("GET /api/health", s.handleHealth)

	s.srv = &http.Server{Addr: s.addr, Handler: mux}
	fmt.Printf("Observatory API listening on %s\n", s.addr)
	return s.srv.ListenAndServe()
}

// Stop gracefully shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	if s.srv != nil {
		return s.srv.Shutdown(ctx)
	}
	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleListUniverses(w http.ResponseWriter, r *http.Request) {
	universes, err := s.state.List()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(universes)
}

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := agentpkg.ListAgents()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agents)
}
