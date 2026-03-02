package gate

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
)

// InvokeRequest is the wire format for element bridge calls.
type InvokeRequest struct {
	Element string   `json:"element"`
	Args    []string `json:"args"`
}

// InvokeResult is the response from an element bridge call.
type InvokeResult struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

// InvokeHandler processes an element bridge invocation.
type InvokeHandler func(element string, args []string) (InvokeResult, error)

// Server is an HTTP-over-Unix-socket server for the host-side Gate.
type Server struct {
	socketPath string
	listener   net.Listener
	handler    InvokeHandler
	httpServer *http.Server
}

// NewServer creates a Gate server that listens on a Unix socket.
func NewServer(socketPath string, handler InvokeHandler) *Server {
	return &Server{
		socketPath: socketPath,
		handler:    handler,
	}
}

// Start begins listening on the Unix socket.
func (s *Server) Start() error {
	// Remove stale socket file
	os.Remove(s.socketPath)

	ln, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.socketPath, err)
	}
	s.listener = ln

	mux := http.NewServeMux()
	mux.HandleFunc("/invoke", s.handleInvoke)
	mux.HandleFunc("/health", s.handleHealth)

	s.httpServer = &http.Server{Handler: mux}

	go s.httpServer.Serve(ln)
	return nil
}

// Stop shuts down the server and removes the socket file.
func (s *Server) Stop() error {
	if s.httpServer != nil {
		s.httpServer.Shutdown(context.Background())
	}
	os.Remove(s.socketPath)
	return nil
}

// SocketPath returns the path to the Unix socket.
func (s *Server) SocketPath() string {
	return s.socketPath
}

func (s *Server) handleInvoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req InvokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	result, err := s.handler(req.Element, req.Args)
	if err != nil {
		result = InvokeResult{
			ExitCode: 1,
			Stderr:   err.Error(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// StubHandler returns a handler that always reports "not implemented".
func StubHandler() InvokeHandler {
	return func(element string, args []string) (InvokeResult, error) {
		return InvokeResult{
			ExitCode: 1,
			Stderr:   fmt.Sprintf("element bridge %q: not implemented (MCP forwarding deferred)", element),
		}, nil
	}
}
