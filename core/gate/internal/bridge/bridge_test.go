package bridge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateWrapperScript(t *testing.T) {
	tests := []struct {
		name        string
		elementName string
		gatePort    int
		wantParts   []string
	}{
		{
			name:        "basic_element",
			elementName: "claude",
			gatePort:    9876,
			wantParts: []string{
				"#!/bin/sh",
				"claude",
				"9876",
				"host.docker.internal:9876",
				"/invoke",
			},
		},
		{
			name:        "different_element_and_port",
			elementName: "mytool",
			gatePort:    12345,
			wantParts: []string{
				"mytool",
				"12345",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := GenerateWrapperScript(tt.elementName, tt.gatePort)
			for _, part := range tt.wantParts {
				if !strings.Contains(script, part) {
					t.Errorf("GenerateWrapperScript(%q, %d) missing %q in output:\n%s",
						tt.elementName, tt.gatePort, part, script)
				}
			}
		})
	}
}

func TestSetupBridges(t *testing.T) {
	t.Run("empty_bridges", func(t *testing.T) {
		dir := t.TempDir()
		err := SetupBridges(dir, nil, 9999)
		if err != nil {
			t.Fatalf("SetupBridges with empty bridges: %v", err)
		}
		// bin dir should still be created
		binDir := filepath.Join(dir, "bin")
		info, err := os.Stat(binDir)
		if err != nil {
			t.Fatalf("expected bin dir to exist: %v", err)
		}
		if !info.IsDir() {
			t.Fatal("expected bin to be a directory")
		}
	})

	t.Run("creates_wrapper_scripts", func(t *testing.T) {
		dir := t.TempDir()
		bridges := []GateBridge{
			{Source: "host:claude", As: "claude", Capabilities: []string{"code"}},
			{Source: "host:git", As: "git-bridge"},
		}

		err := SetupBridges(dir, bridges, 5555)
		if err != nil {
			t.Fatalf("SetupBridges: %v", err)
		}

		for _, b := range bridges {
			path := filepath.Join(dir, "bin", b.As)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Errorf("expected wrapper script %q to exist: %v", b.As, err)
				continue
			}

			content := string(data)
			if !strings.Contains(content, "#!/bin/sh") {
				t.Errorf("wrapper %q missing shebang", b.As)
			}
			if !strings.Contains(content, b.As) {
				t.Errorf("wrapper %q missing element name", b.As)
			}
			if !strings.Contains(content, "5555") {
				t.Errorf("wrapper %q missing port", b.As)
			}

			// Check executable permission
			info, _ := os.Stat(path)
			if info.Mode()&0111 == 0 {
				t.Errorf("wrapper %q is not executable", b.As)
			}
		}
	})
}
