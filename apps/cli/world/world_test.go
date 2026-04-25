package world

import (
	"testing"
)

func TestParseWorkspaceFlags(t *testing.T) {
	tests := []struct {
		name     string
		in       []string
		wantName string // name of first workspace
		wantPath string // path of first workspace
		wantRO   bool
		wantLen  int
		wantErr  bool
	}{
		{name: "empty → ephemeral", in: nil, wantLen: 0},
		{name: "single bare path uses basename", in: []string{"/host/myproject"}, wantName: "myproject", wantPath: "/host/myproject", wantLen: 1},
		{name: "basename lowercased", in: []string{"/host/MyProject"}, wantName: "myproject", wantPath: "/host/MyProject", wantLen: 1},
		{name: "single named", in: []string{"web=/host/a"}, wantName: "web", wantPath: "/host/a", wantLen: 1},
		{name: "read-only", in: []string{"docs=/host/d:ro"}, wantName: "docs", wantPath: "/host/d", wantRO: true, wantLen: 1},
		{name: "multi named", in: []string{"web=/a", "api=/b"}, wantName: "web", wantPath: "/a", wantLen: 2},
		{name: "bare basename in multi", in: []string{"/host/alpha", "/host/beta"}, wantName: "alpha", wantPath: "/host/alpha", wantLen: 2},
		{name: "non-slug basename falls back to workspace<N>", in: []string{"/host/My Project"}, wantName: "workspace0", wantPath: "/host/My Project", wantLen: 1},
		{name: "leading-digit basename falls back", in: []string{"/host/123-app"}, wantName: "workspace0", wantPath: "/host/123-app", wantLen: 1},
		{name: "empty path errors", in: []string{"name="}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseWorkspaceFlags(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got workspaces=%+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("len = %d, want %d; got=%+v", len(got), tt.wantLen, got)
			}
			if tt.wantLen == 0 {
				return
			}
			if got[0].Name != tt.wantName {
				t.Errorf("Name = %q, want %q", got[0].Name, tt.wantName)
			}
			if got[0].Path != tt.wantPath {
				t.Errorf("Path = %q, want %q", got[0].Path, tt.wantPath)
			}
			if got[0].ReadOnly != tt.wantRO {
				t.Errorf("ReadOnly = %v, want %v", got[0].ReadOnly, tt.wantRO)
			}
		})
	}
}
