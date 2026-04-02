package observatory

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	agentpkg "spwn.sh/core/agent"
	"spwn.sh/core/universe/internal/architect"
	"spwn.sh/core/universe/internal/state"
)

// Server serves the Observatory HTTP API.
type Server struct {
	state *state.Store
	arch  *architect.Architect // nil = read-only mode
	addr  string
	srv   *http.Server
}

// New creates an Observatory server. arch may be nil for read-only mode.
func New(s *state.Store, arch *architect.Architect, addr string) *Server {
	return &Server{state: s, arch: arch, addr: addr}
}

// cors wraps a handler with CORS headers.
func cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

// jsonError writes an error response.
func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// jsonOK writes a JSON success response.
func jsonOK(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// requireArch checks that the architect is available (non-read-only mode).
func (s *Server) requireArch(w http.ResponseWriter) bool {
	if s.arch == nil {
		jsonError(w, "server is in read-only mode (no architect configured)", http.StatusServiceUnavailable)
		return false
	}
	return true
}

// Start begins serving the API.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// --- READ endpoints ---
	mux.HandleFunc("GET /api/health", cors(s.handleHealth))
	mux.HandleFunc("GET /api/status", cors(s.handleStatus))
	mux.HandleFunc("GET /api/universes", cors(s.handleListUniverses))
	mux.HandleFunc("GET /api/agents", cors(s.handleListAgents))
	mux.HandleFunc("GET /api/agents/{name}", cors(s.handleGetAgent))
	mux.HandleFunc("GET /api/agents/{name}/journal", cors(s.handleGetAgentJournal))
	mux.HandleFunc("GET /api/worlds/{id}/logs", cors(s.handleWorldLogs))

	// --- WRITE endpoints ---
	mux.HandleFunc("POST /api/worlds", cors(s.handleCreateWorld))
	mux.HandleFunc("DELETE /api/worlds/{id}", cors(s.handleDestroyWorld))
	mux.HandleFunc("POST /api/worlds/{id}/snapshot", cors(s.handleSnapshot))
	mux.HandleFunc("POST /api/agents", cors(s.handleCreateAgent))
	mux.HandleFunc("DELETE /api/agents/{name}", cors(s.handleDeleteAgent))
	mux.HandleFunc("POST /api/agents/{name}/dream", cors(s.handleDream))
	mux.HandleFunc("POST /api/agents/{name}/sleep", cors(s.handleSleep))
	mux.HandleFunc("POST /api/agents/{name}/fork", cors(s.handleFork))

	// --- CORS preflight for all paths ---
	mux.HandleFunc("OPTIONS /", cors(func(w http.ResponseWriter, r *http.Request) {}))

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

// ============================================================
// READ handlers
// ============================================================

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, map[string]string{"status": "ok"})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	worlds, _ := s.state.List()
	agents, _ := agentpkg.ListAgents()

	status := map[string]interface{}{
		"worlds":    len(worlds),
		"agents":    len(agents),
		"architect": s.arch != nil,
	}
	jsonOK(w, status)
}

func (s *Server) handleListUniverses(w http.ResponseWriter, r *http.Request) {
	universes, err := s.state.List()
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonOK(w, universes)
}

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := agentpkg.ListAgents()
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonOK(w, agents)
}

func (s *Server) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		jsonError(w, "agent name is required", 400)
		return
	}

	info, err := agentpkg.InspectAgent(name)
	if err != nil {
		jsonError(w, err.Error(), 404)
		return
	}
	jsonOK(w, info)
}

func (s *Server) handleGetAgentJournal(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		jsonError(w, "agent name is required", 400)
		return
	}

	mindPath := agentpkg.AgentDir(name)
	entries, err := agentpkg.ListJournal(mindPath, 100)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonOK(w, entries)
}

func (s *Server) handleWorldLogs(w http.ResponseWriter, r *http.Request) {
	if !s.requireArch(w) {
		return
	}

	worldID := r.PathValue("id")
	if worldID == "" {
		jsonError(w, "world id is required", 400)
		return
	}

	follow := r.URL.Query().Get("follow") == "true"
	tail := r.URL.Query().Get("tail")
	if tail == "" {
		tail = "100"
	}

	reader, err := s.arch.Logs(r.Context(), worldID, follow, tail)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	defer reader.Close()

	if follow {
		// SSE streaming
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			jsonError(w, "streaming not supported", 500)
			return
		}

		buf := make([]byte, 4096)
		for {
			n, err := reader.Read(buf)
			if n > 0 {
				lines := strings.Split(string(buf[:n]), "\n")
				for _, line := range lines {
					if line != "" {
						fmt.Fprintf(w, "data: %s\n\n", line)
					}
				}
				flusher.Flush()
			}
			if err != nil {
				break
			}
		}
	} else {
		w.Header().Set("Content-Type", "text/plain")
		io.Copy(w, reader)
	}
}

// ============================================================
// WRITE handlers
// ============================================================

func (s *Server) handleCreateWorld(w http.ResponseWriter, r *http.Request) {
	if !s.requireArch(w) {
		return
	}

	var body struct {
		ConfigName string `json:"config"`
		AgentName  string `json:"agent"`
		Workspace  string `json:"workspace"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), 400)
		return
	}

	result, err := s.arch.Spawn(r.Context(), architect.SpawnOpts{
		ConfigName: body.ConfigName,
		AgentName:  body.AgentName,
		Workspace:  body.Workspace,
	})
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonOK(w, result)
}

func (s *Server) handleDestroyWorld(w http.ResponseWriter, r *http.Request) {
	if !s.requireArch(w) {
		return
	}

	worldID := r.PathValue("id")
	if worldID == "" {
		jsonError(w, "world id is required", 400)
		return
	}

	world, err := s.arch.Destroy(r.Context(), worldID)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonOK(w, world)
}

func (s *Server) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	if !s.requireArch(w) {
		return
	}

	worldID := r.PathValue("id")
	if worldID == "" {
		jsonError(w, "world id is required", 400)
		return
	}

	var body struct {
		Name string `json:"name"`
	}
	json.NewDecoder(r.Body).Decode(&body) // optional body

	tag, err := s.arch.Snapshot(r.Context(), worldID, body.Name)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonOK(w, map[string]string{"tag": tag})
}

func (s *Server) handleCreateAgent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		jsonError(w, "name is required", 400)
		return
	}

	path, err := agentpkg.InitMind(body.Name)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonOK(w, map[string]string{"name": body.Name, "path": path})
}

func (s *Server) handleDeleteAgent(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		jsonError(w, "agent name is required", 400)
		return
	}

	if err := agentpkg.DeleteAgent(name); err != nil {
		jsonError(w, err.Error(), 404)
		return
	}
	jsonOK(w, map[string]string{"deleted": name})
}

func (s *Server) handleDream(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		jsonError(w, "agent name is required", 400)
		return
	}

	result, err := agentpkg.Dream(name)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonOK(w, result)
}

func (s *Server) handleSleep(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		jsonError(w, "agent name is required", 400)
		return
	}

	result, err := agentpkg.Sleep(name)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonOK(w, result)
}

func (s *Server) handleFork(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		jsonError(w, "agent name is required", 400)
		return
	}

	var body struct {
		Target string   `json:"target"`
		Layers []string `json:"layers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Target == "" {
		jsonError(w, "target is required", 400)
		return
	}

	result, err := agentpkg.Fork(name, body.Target, body.Layers)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonOK(w, result)
}
