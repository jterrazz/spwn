package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func TestNewServer(t *testing.T) {
	s := NewServer(StubHandler())
	if s == nil {
		t.Fatal("NewServer returned nil")
	}
	if s.Port() != 0 {
		t.Errorf("Port() before Start() = %d, want 0", s.Port())
	}
}

func TestServerLifecycle(t *testing.T) {
	s := NewServer(StubHandler())

	if err := s.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Stop()

	port := s.Port()
	if port == 0 {
		t.Fatal("Port() after Start() = 0, want non-zero")
	}

	// Health endpoint should respond
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /health status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var health map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if health["status"] != "ok" {
		t.Errorf("health status = %q, want %q", health["status"], "ok")
	}
}

func TestServerInvoke(t *testing.T) {
	tests := []struct {
		name       string
		handler    InvokeHandler
		request    InvokeRequest
		wantCode   int
		wantResult InvokeResult
	}{
		{
			name:    "stub_handler",
			handler: StubHandler(),
			request: InvokeRequest{Element: "claude", Args: []string{"--help"}},
			wantCode: http.StatusOK,
			wantResult: InvokeResult{
				ExitCode: 1,
				Stderr:   `element bridge "claude": not implemented (MCP forwarding deferred)`,
			},
		},
		{
			name: "custom_handler_success",
			handler: func(element string, args []string) (InvokeResult, error) {
				return InvokeResult{ExitCode: 0, Stdout: "hello " + element}, nil
			},
			request:  InvokeRequest{Element: "echo", Args: []string{"hi"}},
			wantCode: http.StatusOK,
			wantResult: InvokeResult{
				ExitCode: 0,
				Stdout:   "hello echo",
			},
		},
		{
			name: "handler_returns_error",
			handler: func(element string, args []string) (InvokeResult, error) {
				return InvokeResult{}, fmt.Errorf("boom")
			},
			request:  InvokeRequest{Element: "fail"},
			wantCode: http.StatusOK,
			wantResult: InvokeResult{
				ExitCode: 1,
				Stderr:   "boom",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewServer(tt.handler)
			if err := s.Start(); err != nil {
				t.Fatalf("Start: %v", err)
			}
			defer s.Stop()

			body, _ := json.Marshal(tt.request)
			resp, err := http.Post(
				fmt.Sprintf("http://127.0.0.1:%d/invoke", s.Port()),
				"application/json",
				bytes.NewReader(body),
			)
			if err != nil {
				t.Fatalf("POST /invoke: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantCode {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.wantCode)
			}

			var result InvokeResult
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("decode: %v", err)
			}

			if result.ExitCode != tt.wantResult.ExitCode {
				t.Errorf("ExitCode = %d, want %d", result.ExitCode, tt.wantResult.ExitCode)
			}
			if result.Stdout != tt.wantResult.Stdout {
				t.Errorf("Stdout = %q, want %q", result.Stdout, tt.wantResult.Stdout)
			}
			if result.Stderr != tt.wantResult.Stderr {
				t.Errorf("Stderr = %q, want %q", result.Stderr, tt.wantResult.Stderr)
			}
		})
	}
}

func TestServerInvokeMethodNotAllowed(t *testing.T) {
	s := NewServer(StubHandler())
	if err := s.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Stop()

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/invoke", s.Port()))
	if err != nil {
		t.Fatalf("GET /invoke: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("GET /invoke status = %d, want %d", resp.StatusCode, http.StatusMethodNotAllowed)
	}
}
