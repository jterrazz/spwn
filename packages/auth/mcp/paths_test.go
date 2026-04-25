package mcp

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"
	"testing"
)

// TestProviderKey_MatchesMcp2cli locks in that we hash the URL the
// same way mcp2cli does (sha256, hex-encoded, first 16 chars). If
// either side drifts, the on-disk cache becomes invisible to the
// other and "logged in on the host" silently fails inside the world.
func TestProviderKey_MatchesMcp2cli(t *testing.T) {
	url := "https://mcp.notion.com/mcp"
	want := hex.EncodeToString(sha256.New().Sum(nil)) // dummy; replaced below
	h := sha256.Sum256([]byte(url))
	want = hex.EncodeToString(h[:])[:16]
	got := providerKey(url)
	if got != want {
		t.Fatalf("providerKey(%q)=%q; want %q (mcp2cli hash)", url, got, want)
	}
	if len(got) != 16 || strings.ContainsAny(got, "ABCDEF") {
		t.Errorf("providerKey must be 16 lowercase hex chars, got %q", got)
	}
}

func TestProviderTokenPath_LandsUnderCacheDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)
	p := Provider{Name: "notion", URL: "https://mcp.notion.com/mcp"}
	got := ProviderTokenPath(p)
	want := filepath.Join(tmp, "credentials", "mcp", "oauth", providerKey(p.URL), "tokens.json")
	if got != want {
		t.Errorf("ProviderTokenPath = %q; want %q", got, want)
	}
}
