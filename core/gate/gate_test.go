package gate

import (
	"strings"
	"testing"
)

func TestStubHandler_ReturnsHandler(t *testing.T) {
	handler := StubHandler()
	if handler == nil {
		t.Fatal("StubHandler returned nil")
	}
}

func TestNewServer_ReturnsServer(t *testing.T) {
	handler := StubHandler()
	server := NewServer(handler)
	if server == nil {
		t.Fatal("NewServer returned nil")
	}
}

func TestGenerateWrapperScript_ContainsElementName(t *testing.T) {
	script := GenerateWrapperScript("my-tool", 8080)
	if !strings.Contains(script, "my-tool") {
		t.Errorf("Expected wrapper script to contain element name 'my-tool', got:\n%s", script)
	}
}

func TestGenerateWrapperScript_ContainsPort(t *testing.T) {
	script := GenerateWrapperScript("my-tool", 9999)
	if !strings.Contains(script, "9999") {
		t.Errorf("Expected wrapper script to contain port '9999', got:\n%s", script)
	}
}

func TestSetupBridges_EmptyBridges(t *testing.T) {
	dir := t.TempDir()
	err := SetupBridges(dir, nil, 8080)
	if err != nil {
		t.Fatalf("SetupBridges with empty bridges failed: %v", err)
	}
}

func TestExecHandler_ReturnsHandler(t *testing.T) {
	bridges := []Bridge{
		{Source: "echo hello", As: "test-tool"},
	}
	handler := ExecHandler(bridges)
	if handler == nil {
		t.Fatal("ExecHandler returned nil")
	}
}
