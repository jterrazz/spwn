package gate

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

// Handler processes an interaction request.
type Handler func(ctx context.Context, req *Request) (*Response, error)

// Request represents an invocation from a wrapper script.
type Request struct {
	Interaction string   `json:"interaction"`
	Args        []string `json:"args"`
}

// Response is returned to the wrapper script.
type Response struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

// Gate serves HTTP over a Unix socket to bridge interactions into a universe.
type Gate struct {
	socketDir  string
	socketPath string
	handlers   map[string]Handler
	mu         sync.RWMutex
	server     *http.Server
	listener   net.Listener
}

// New creates a Gate that will listen on socketDir/gate.sock.
func New(socketDir string) *Gate {
	return &Gate{
		socketDir:  socketDir,
		socketPath: filepath.Join(socketDir, "gate.sock"),
		handlers:   make(map[string]Handler),
	}
}

// RegisterHandler registers a handler for a named interaction.
func (g *Gate) RegisterHandler(name string, h Handler) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.handlers[name] = h
}

// SocketDir returns the directory containing the socket file.
func (g *Gate) SocketDir() string {
	return g.socketDir
}

// Start begins listening on the Unix socket in a background goroutine.
func (g *Gate) Start() error {
	if err := os.MkdirAll(g.socketDir, 0o755); err != nil {
		return fmt.Errorf("creating socket directory: %w", err)
	}

	// Remove stale socket file.
	os.Remove(g.socketPath)

	listener, err := net.Listen("unix", g.socketPath)
	if err != nil {
		return fmt.Errorf("listening on unix socket: %w", err)
	}
	g.listener = listener

	mux := http.NewServeMux()
	mux.HandleFunc("POST /invoke", g.handleInvoke)
	mux.HandleFunc("GET /health", g.handleHealth)

	g.server = &http.Server{Handler: mux}

	go g.server.Serve(listener)

	return nil
}

// Stop gracefully shuts down the gate server.
func (g *Gate) Stop(ctx context.Context) error {
	if g.server == nil {
		return nil
	}
	err := g.server.Shutdown(ctx)
	os.Remove(g.socketPath)
	return err
}

func (g *Gate) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (g *Gate) handleInvoke(w http.ResponseWriter, r *http.Request) {
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	g.mu.RLock()
	handler, ok := g.handlers[req.Interaction]
	g.mu.RUnlock()

	if !ok {
		resp := &Response{
			ExitCode: 1,
			Stderr:   fmt.Sprintf("unknown interaction: %s", req.Interaction),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp, err := handler(r.Context(), &req)
	if err != nil {
		resp = &Response{
			ExitCode: 1,
			Stderr:   err.Error(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
