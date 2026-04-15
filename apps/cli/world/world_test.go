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
		{name: "single bare path", in: []string{"/host/a"}, wantName: "workspace0", wantPath: "/host/a", wantLen: 1},
		{name: "single named", in: []string{"web=/host/a"}, wantName: "web", wantPath: "/host/a", wantLen: 1},
		{name: "read-only", in: []string{"docs=/host/d:ro"}, wantName: "docs", wantPath: "/host/d", wantRO: true, wantLen: 1},
		{name: "multi named", in: []string{"web=/a", "api=/b"}, wantName: "web", wantPath: "/a", wantLen: 2},
		{name: "bare in multi gets workspace<N>", in: []string{"/a", "/b"}, wantName: "workspace0", wantPath: "/a", wantLen: 2},
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
