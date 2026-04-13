package api

import (
	"encoding/json"
	"strings"
	"testing"
)

// These tests pin the JSON shape of POST /api/worlds. They exist because a
// stale binary once had only `Workspace string` (legacy) on its body struct,
// so JSON requests carrying `workspaces: [...]` were silently dropped — the
// world spawned ephemeral and Docker auto-created an anonymous /workspace
// volume. Any regression of the field name or JSON tag will fail here.

func TestCreateWorldRequest_DecodesNewWorkspacesArray(t *testing.T) {
	raw := `{
		"config": "default",
		"agent": "neo",
		"workspaces": [
			{"name": "web", "path": "/host/web"},
			{"name": "api", "path": "/host/api", "readonly": true}
		]
	}`

	var req createWorldRequest
	if err := json.NewDecoder(strings.NewReader(raw)).Decode(&req); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(req.Workspaces) != 2 {
		t.Fatalf("expected 2 workspaces, got %d; struct=%+v", len(req.Workspaces), req)
	}
	if req.Workspaces[0].Name != "web" || req.Workspaces[0].Path != "/host/web" {
		t.Errorf("first workspace: %+v", req.Workspaces[0])
	}
	if req.Workspaces[1].Name != "api" || !req.Workspaces[1].ReadOnly {
		t.Errorf("second workspace: %+v", req.Workspaces[1])
	}
}

func TestCreateWorldRequest_MigratesLegacyWorkspaceField(t *testing.T) {
	raw := `{"config":"default","agent":"neo","workspace":"/host/legacy"}`

	var req createWorldRequest
	if err := json.NewDecoder(strings.NewReader(raw)).Decode(&req); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if req.Workspace != "/host/legacy" {
		t.Errorf("legacy field not decoded: %+v", req)
	}

	resolved := req.resolveWorkspaces()
	if len(resolved) != 1 || resolved[0].Path != "/host/legacy" || resolved[0].Name != "default" {
		t.Errorf("legacy workspace not migrated: %+v", resolved)
	}
}

func TestCreateWorldRequest_NewSliceWinsOverLegacy(t *testing.T) {
	// If a request sends both, the new slice takes precedence — the legacy
	// field is only a fallback when the slice is empty.
	raw := `{"workspace":"/legacy","workspaces":[{"name":"a","path":"/new"}]}`

	var req createWorldRequest
	if err := json.NewDecoder(strings.NewReader(raw)).Decode(&req); err != nil {
		t.Fatalf("decode: %v", err)
	}
	resolved := req.resolveWorkspaces()
	if len(resolved) != 1 || resolved[0].Path != "/new" {
		t.Errorf("new slice should win: %+v", resolved)
	}
}

func TestCreateWorldRequest_EmptyBodyIsEphemeral(t *testing.T) {
	var req createWorldRequest
	if err := json.NewDecoder(strings.NewReader(`{}`)).Decode(&req); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(req.resolveWorkspaces()) != 0 {
		t.Errorf("empty body should resolve to ephemeral (0 workspaces), got %+v", req.resolveWorkspaces())
	}
}

func TestCreateWorldRequest_ReadOnlyFlag(t *testing.T) {
	raw := `{"workspaces":[{"name":"docs","path":"/d","readonly":true}]}`
	var req createWorldRequest
	if err := json.NewDecoder(strings.NewReader(raw)).Decode(&req); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !req.Workspaces[0].ReadOnly {
		t.Errorf("readonly flag should decode to true, got %+v", req.Workspaces[0])
	}
}
