package architect

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"spwn.sh/packages/world/models"
)

// These tests pin the exact Docker bind specs produced for each
// workspace configuration. The contract is uniform under the new
// architecture: every workspace gets mounted at /workspaces/<name>, no
// matter how many there are. Running `ls /workspaces` always tells the
// agent which projects are available.
//
// Contract:
//   0 workspaces: no binds (the agent has no /workspaces directory; its
//                 only writable space is /agents/<name>).
//   1+:           one bind per workspace at /workspaces/<name>.

func TestBuildWorkspaceBinds_Ephemeral(t *testing.T) {
	got := buildWorkspaceBinds(nil)
	if got != nil {
		t.Errorf("ephemeral world should produce zero binds, got %v", got)
	}
}

func TestBuildWorkspaceBinds_SingleWorkspace(t *testing.T) {
	got := buildWorkspaceBinds([]models.Workspace{
		{Name: "proj", Path: "/host/project"},
	})
	want := []string{"/host/project:/workspaces/proj"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("single workspace binds:\n got  %v\n want %v", got, want)
	}
}

func TestBuildWorkspaceBinds_SingleWorkspace_ReadOnly(t *testing.T) {
	got := buildWorkspaceBinds([]models.Workspace{
		{Name: "docs", Path: "/host/docs", ReadOnly: true},
	})
	want := []string{"/host/docs:/workspaces/docs:ro"}
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
		"/host/api:/workspaces/api",
		"/host/web:/workspaces/web",
	}
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("multi binds:\n got  %v\n want %v", got, want)
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
		"/host/code:/workspaces/code",
		"/host/data:/workspaces/data",
		"/host/docs:/workspaces/docs:ro",
	}
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("mixed RO multi binds:\n got  %v\n want %v", got, want)
	}
}

func TestBuildWorkspaceBinds_ThreeWorkspaces_AllUnderWork(t *testing.T) {
	got := buildWorkspaceBinds([]models.Workspace{
		{Name: "a", Path: "/a"},
		{Name: "b", Path: "/b"},
		{Name: "c", Path: "/c"},
	})
	if len(got) != 3 {
		t.Errorf("expected exactly 3 binds, got %d: %v", len(got), got)
	}
	for _, b := range got {
		if !strings.Contains(b, ":/workspaces/") {
			t.Errorf("every bind should target /workspaces/<name>, got: %q", b)
		}
	}
}

func TestWorkspaceContainerPath(t *testing.T) {
	tests := []struct {
		name  string
		total int
		want  string
	}{
		{"proj", 1, "/workspaces/proj"},
		{"web", 2, "/workspaces/web"},
		{"api", 2, "/workspaces/api"},
		{"docs", 5, "/workspaces/docs"},
	}
	for _, tt := range tests {
		if got := workspaceContainerPath(tt.name, tt.total); got != tt.want {
			t.Errorf("workspaceContainerPath(%q, %d) = %q, want %q", tt.name, tt.total, got, tt.want)
		}
	}
}
