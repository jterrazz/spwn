package architect

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"spwn.sh/core/universe/internal/models"
)

// These tests pin the exact Docker bind specs produced for each workspace
// configuration. In multi-mode we mount each workspace at /workspace/<name>
// so the agent can run `ls /workspace` and immediately see what it can
// work with. In single-mode we keep the legacy flat /workspace for
// backward compat. If the 0/1/N contract ever regresses, these tests fail
// on a pure Go unit — no Docker required.
//
// Contract:
//   0 workspaces: no binds (image-baked /workspace).
//   1 workspace:  one bind — /workspace (flat legacy layout).
//   2+:           N binds, one per workspace at /workspace/<name>.

func TestBuildWorkspaceBinds_Ephemeral(t *testing.T) {
	got := buildWorkspaceBinds(nil)
	if got != nil {
		t.Errorf("ephemeral world should produce zero binds, got %v", got)
	}
}

func TestBuildWorkspaceBinds_SingleWorkspace_FlatLegacyLayout(t *testing.T) {
	got := buildWorkspaceBinds([]models.Workspace{
		{Name: "proj", Path: "/host/project"},
	})
	want := []string{"/host/project:/workspace"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("single workspace binds:\n got  %v\n want %v", got, want)
	}
}

func TestBuildWorkspaceBinds_SingleWorkspace_ReadOnly(t *testing.T) {
	got := buildWorkspaceBinds([]models.Workspace{
		{Name: "docs", Path: "/host/docs", ReadOnly: true},
	})
	want := []string{"/host/docs:/workspace:ro"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("read-only single workspace binds:\n got  %v\n want %v", got, want)
	}
}

func TestBuildWorkspaceBinds_MultiWorkspace_NamedSubdirs(t *testing.T) {
	got := buildWorkspaceBinds([]models.Workspace{
		{Name: "web", Path: "/host/web"},
		{Name: "api", Path: "/host/api"},
	})
	sort.Strings(got)
	want := []string{
		"/host/api:/workspace/api",
		"/host/web:/workspace/web",
	}
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("multi binds:\n got  %v\n want %v", got, want)
	}
	// In multi-mode no workspace takes over the flat /workspace — that would
	// make one workspace secretly "primary" and hide the others.
	for _, b := range got {
		if strings.HasSuffix(b, ":/workspace") || strings.HasSuffix(b, ":/workspace:ro") {
			t.Errorf("multi-workspace must not bind to flat /workspace, got: %q", b)
		}
	}
}

func TestBuildWorkspaceBinds_MultiWorkspace_MixedReadOnly(t *testing.T) {
	got := buildWorkspaceBinds([]models.Workspace{
		{Name: "code", Path: "/host/code"},
		{Name: "docs", Path: "/host/docs", ReadOnly: true},
		{Name: "data", Path: "/host/data"},
	})
	sort.Strings(got)
	want := []string{
		"/host/code:/workspace/code",
		"/host/data:/workspace/data",
		"/host/docs:/workspace/docs:ro",
	}
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("mixed RO multi binds:\n got  %v\n want %v", got, want)
	}
}

func TestBuildWorkspaceBinds_ThreeWorkspaces_AllUnderWorkspace(t *testing.T) {
	got := buildWorkspaceBinds([]models.Workspace{
		{Name: "a", Path: "/a"},
		{Name: "b", Path: "/b"},
		{Name: "c", Path: "/c"},
	})
	if len(got) != 3 {
		t.Errorf("expected exactly 3 binds, got %d: %v", len(got), got)
	}
	for _, b := range got {
		if strings.HasSuffix(b, ":/workspace") {
			t.Errorf("no workspace should bind to flat /workspace in multi-mode, got: %q", b)
		}
		if !strings.Contains(b, ":/workspace/") {
			t.Errorf("every bind should target /workspace/<name>, got: %q", b)
		}
	}
}

func TestWorkspaceContainerPath(t *testing.T) {
	tests := []struct {
		name  string
		total int
		want  string
	}{
		{"proj", 1, "/workspace"},
		{"web", 2, "/workspace/web"},
		{"api", 2, "/workspace/api"},
		{"docs", 5, "/workspace/docs"},
	}
	for _, tt := range tests {
		if got := workspaceContainerPath(tt.name, tt.total); got != tt.want {
			t.Errorf("workspaceContainerPath(%q, %d) = %q, want %q", tt.name, tt.total, got, tt.want)
		}
	}
}
