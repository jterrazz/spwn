package server

import (
	"runtime"
	"testing"

	"github.com/jterrazz/spwn/core/gate/internal/bridge"
)

func TestExecHandler_Echo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("echo test not portable to Windows")
	}

	bridges := []bridge.GateBridge{
		{Source: "echo", As: "my-echo"},
	}

	handler := ExecHandler(bridges)
	result, err := handler("my-echo", []string{"hello", "world"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", result.ExitCode, result.Stderr)
	}
	if got := result.Stdout; got != "hello world\n" {
		t.Errorf("expected stdout %q, got %q", "hello world\n", got)
	}
}

func TestExecHandler_UnknownElement(t *testing.T) {
	handler := ExecHandler(nil)
	result, err := handler("nonexistent", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", result.ExitCode)
	}
	if result.Stderr == "" {
		t.Fatal("expected non-empty stderr for unknown element")
	}
}

func TestExecHandler_CapabilityDenied(t *testing.T) {
	bridges := []bridge.GateBridge{
		{Source: "echo", As: "restricted", Capabilities: []string{"read"}},
	}

	handler := ExecHandler(bridges)
	result, err := handler("restricted", []string{"write", "data"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", result.ExitCode)
	}
	if result.Stderr == "" {
		t.Fatal("expected non-empty stderr for denied capability")
	}
}

func TestExecHandler_CapabilityAllowed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("echo test not portable to Windows")
	}

	bridges := []bridge.GateBridge{
		{Source: "echo", As: "restricted", Capabilities: []string{"read", "list"}},
	}

	handler := ExecHandler(bridges)
	result, err := handler("restricted", []string{"read", "something"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", result.ExitCode, result.Stderr)
	}
}

func TestExecHandler_NonZeroExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("false command not available on Windows")
	}

	bridges := []bridge.GateBridge{
		{Source: "false", As: "fail-cmd"},
	}

	handler := ExecHandler(bridges)
	result, err := handler("fail-cmd", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode == 0 {
		t.Fatal("expected non-zero exit code from 'false' command")
	}
}
