//go:build integration

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/jterrazz/universe/internal/config"
	"github.com/jterrazz/universe/internal/gate"
)

func TestGateRoundTrip(t *testing.T) {
	b, ctx := TestBackend(t)

	// Set up gate server on host.
	gateDir, err := os.MkdirTemp("", "gate-e2e-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(gateDir)

	g := gate.New(gateDir)
	g.RegisterHandler("echo-test", gate.NewEchoHandler())
	if err := g.Start(); err != nil {
		t.Fatalf("start gate: %v", err)
	}
	defer g.Stop(context.Background())

	// Create universe with gate mount.
	cfg := &config.UniverseConfig{
		Image:   TestImage,
		GateDir: gateDir,
		Interactions: []config.Interaction{
			{Source: "test/echo", As: "echo-test"},
		},
	}
	id := MustCreateAndStart(t, b, ctx, cfg)

	// Install wrapper scripts.
	for _, cmd := range gate.AllWrapperCommands(cfg.Interactions) {
		result, err := b.Exec(ctx, id, []string{"sh", "-c", cmd})
		if err != nil {
			t.Fatalf("install wrapper: %v", err)
		}
		if result.ExitCode != 0 {
			t.Fatalf("wrapper install failed (exit %d): %s", result.ExitCode, result.Stderr)
		}
	}

	// Execute wrapper from inside container.
	result, err := b.Exec(ctx, id, []string{"sh", "-c", "/gate/bin/echo-test hello world"})
	if err != nil {
		t.Fatalf("exec wrapper: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d (stderr: %s)", result.ExitCode, result.Stderr)
	}

	// Also verify directly via gate socket from host side.
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", gateDir+"/gate.sock")
			},
		},
		Timeout: 5 * time.Second,
	}

	req := gate.Request{Interaction: "echo-test", Args: []string{"host-side"}}
	body, _ := json.Marshal(req)

	resp, err := client.Post("http://localhost/invoke", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("host invoke: %v", err)
	}
	defer resp.Body.Close()

	var gateResp gate.Response
	if err := json.NewDecoder(resp.Body).Decode(&gateResp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if gateResp.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", gateResp.ExitCode)
	}
	if gateResp.Stdout != "echo: echo-test host-side" {
		t.Errorf("unexpected stdout: %q", gateResp.Stdout)
	}
}
