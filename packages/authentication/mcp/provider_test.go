package mcp

import (
	"errors"
	"strings"
	"testing"
)

func TestLookup(t *testing.T) {
	tests := []struct {
		in    string
		ok    bool
		wantN string
	}{
		{"notion", true, "notion"},
		{"NOTION", true, "notion"},
		{"  notion  ", true, "notion"},
		{"unknown-thing", false, ""},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			p, ok := Lookup(tc.in)
			if ok != tc.ok {
				t.Fatalf("Lookup(%q) ok=%v want %v", tc.in, ok, tc.ok)
			}
			if ok && p.Name != tc.wantN {
				t.Errorf("name=%q want %q", p.Name, tc.wantN)
			}
		})
	}
}

func TestUnknownProviderError_ListsKnown(t *testing.T) {
	err := UnknownProviderError("bogus")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "notion") {
		t.Errorf("error should list known providers, got %q", err)
	}
	// Must not be a sentinel that swallows the raw input.
	if errors.Is(err, errors.New("hardcoded")) {
		t.Error("error should not match unrelated sentinel")
	}
}
