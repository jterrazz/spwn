package architect

import (
	"os"
	"path/filepath"
	"testing"
)

// resolveSpwnBinary picks the host-side spwn binary path that should
// be bind-mounted into chief-mode worlds. The order is documented in
// the function: SPWN_BINARY env > cached cross-compile at
// ~/.spwn/cache/spwn-linux > on-demand cross-compile.
//
// The cross-compile path is exercised by integration tests with a
// real Go toolchain; the unit tests below pin the cheaper edges
// (env override, cache hit) which run in milliseconds.

func TestResolveSpwnBinary_EnvOverrideWins(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "spwn-fake")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SPWN_BINARY", bin)

	got, ok := resolveSpwnBinary()
	if !ok {
		t.Fatal("ok = false, want true (env override should resolve)")
	}
	if got != bin {
		t.Errorf("path = %q, want %q (SPWN_BINARY env)", got, bin)
	}
}

func TestResolveSpwnBinary_EnvOverrideRejectsDirectory(t *testing.T) {
	// SPWN_BINARY pointing to a directory must not be returned —
	// the bind-mount would fail downstream and the chief would
	// silently lose the spwn capability without a clear error trail.
	// Falls through to the cache/cross-compile path; we don't assert
	// the eventual return because that exercises the toolchain.
	dir := t.TempDir()
	t.Setenv("SPWN_BINARY", dir)

	// We can't easily isolate the cache path without exposing
	// platform.BaseDir() as a test seam, so just assert the env
	// directory is NOT what gets returned.
	got, _ := resolveSpwnBinary()
	if got == dir {
		t.Errorf("path = %q (a directory), want fallthrough", got)
	}
}
