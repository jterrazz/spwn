package gate

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"testing"
	"time"
)

func testClient(socketPath string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}
}

func TestStartAndStop(t *testing.T) {
	dir, err := os.MkdirTemp("", "gate-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	g := New(dir)
	if err := g.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := g.Stop(ctx); err != nil {
		t.Fatalf("stop: %v", err)
	}
}

func TestHealthEndpoint(t *testing.T) {
	dir, err := os.MkdirTemp("", "gate-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	g := New(dir)
	if err := g.Start(); err != nil {
		t.Fatal(err)
	}
	defer g.Stop(context.Background())

	client := testClient(g.socketPath)
	resp, err := client.Get("http://localhost/health")
	if err != nil {
		t.Fatalf("health request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestInvokeEchoHandler(t *testing.T) {
	dir, err := os.MkdirTemp("", "gate-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	g := New(dir)
	g.RegisterHandler("echo-test", NewEchoHandler())
	if err := g.Start(); err != nil {
		t.Fatal(err)
	}
	defer g.Stop(context.Background())

	client := testClient(g.socketPath)
	req := Request{Interaction: "echo-test", Args: []string{"hello", "world"}}
	body, _ := json.Marshal(req)

	resp, err := client.Post("http://localhost/invoke", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("invoke request: %v", err)
	}
	defer resp.Body.Close()

	var gateResp Response
	if err := json.NewDecoder(resp.Body).Decode(&gateResp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if gateResp.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", gateResp.ExitCode)
	}
	if gateResp.Stdout != "echo: echo-test hello world" {
		t.Errorf("unexpected stdout: %q", gateResp.Stdout)
	}
}

func TestInvokeUnknownInteraction(t *testing.T) {
	dir, err := os.MkdirTemp("", "gate-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	g := New(dir)
	if err := g.Start(); err != nil {
		t.Fatal(err)
	}
	defer g.Stop(context.Background())

	client := testClient(g.socketPath)
	req := Request{Interaction: "nonexistent", Args: nil}
	body, _ := json.Marshal(req)

	resp, err := client.Post("http://localhost/invoke", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("invoke request: %v", err)
	}
	defer resp.Body.Close()

	var gateResp Response
	if err := json.NewDecoder(resp.Body).Decode(&gateResp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if gateResp.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", gateResp.ExitCode)
	}
	if gateResp.Stderr != "unknown interaction: nonexistent" {
		t.Errorf("unexpected stderr: %q", gateResp.Stderr)
	}
}
