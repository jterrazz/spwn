package source

import "testing"

func TestParsePlaybookHeader(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantOK  bool
		wantN   string
		wantD   string
	}{
		{
			name:   "valid frontmatter",
			body:   "---\nname: deploy\ndescription: Ship code to prod.\n---\n\nbody here",
			wantOK: true,
			wantN:  "deploy",
			wantD:  "Ship code to prod.",
		},
		{
			name:   "missing frontmatter",
			body:   "# Just a plain playbook\n\nNo header.",
			wantOK: false,
		},
		{
			name:   "missing name",
			body:   "---\ndescription: No name here.\n---\nbody",
			wantOK: false,
		},
		{
			name:   "missing description",
			body:   "---\nname: only-name\n---\nbody",
			wantOK: false,
		},
		{
			name:   "unterminated block",
			body:   "---\nname: foo\ndescription: bar\n(no closing marker)",
			wantOK: false,
		},
		{
			name:   "empty body",
			body:   "",
			wantOK: false,
		},
		{
			name:   "only opening marker then EOF",
			body:   "---",
			wantOK: false,
		},
		{
			name:   "whitespace in keys",
			body:   "---\nname:   spaced   \ndescription:   also spaced   \n---\n",
			wantOK: true,
			wantN:  "spaced",
			wantD:  "also spaced",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parsePlaybookHeader([]byte(tc.body))
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v (body: %q)", ok, tc.wantOK, tc.body)
			}
			if !ok {
				return
			}
			if got.Name != tc.wantN {
				t.Errorf("Name = %q, want %q", got.Name, tc.wantN)
			}
			if got.Description != tc.wantD {
				t.Errorf("Description = %q, want %q", got.Description, tc.wantD)
			}
		})
	}
}
