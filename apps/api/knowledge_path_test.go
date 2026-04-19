package api

import "testing"

func TestSanitizeKnowledgePath(t *testing.T) {
	cases := []struct {
		name   string
		in     string
		want   string
		wantOK bool
	}{
		{"simple filename", "notes.md", "notes.md", true},
		{"nested path", "domain/api.md", "domain/api.md", true},
		{"deep nested", "a/b/c/d.md", "a/b/c/d.md", true},
		{"empty string", "", "", false},
		{"lone dotdot", "..", "", false},
		{"parent escape", "../etc/passwd", "", false},
		{"multi-hop escape", "a/../../etc", "", false},
		{"absolute path", "/etc/passwd", "", false},
		{"trailing dotdot escape", "a/b/../../..", "", false},
		{"bare slash", "/", "", false},
		{"cleaned to same", "./foo.md", "foo.md", true},
		// Legitimate substrings that contain ".." but don't escape.
		{"filename with dotdot literal", "a..b.md", "a..b.md", true},
		{"dotdot prefix on filename", "..foo.md", "..foo.md", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := sanitizeKnowledgePath(tc.in)
			if ok != tc.wantOK {
				t.Fatalf("sanitizeKnowledgePath(%q): ok = %v, want %v", tc.in, ok, tc.wantOK)
			}
			if ok && got != tc.want {
				t.Errorf("sanitizeKnowledgePath(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
