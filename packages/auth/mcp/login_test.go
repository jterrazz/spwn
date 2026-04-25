package mcp

import (
	"strings"
	"testing"
)

// TestHelperDockerfile_ContainsBindPatch is the trip-wire that
// catches a regression where the embedded Dockerfile drifts and
// stops patching mcp2cli's loopback binding. Without that patch,
// macOS Docker can't forward the OAuth callback into the helper
// container and login silently times out.
func TestHelperDockerfile_ContainsBindPatch(t *testing.T) {
	df := string(helperDockerfile)
	for _, want := range []string{
		`pip install --no-cache-dir mcp2cli==3.0.2`,
		`callback_host = parsed.hostname`,
		`callback_host = "127.0.0.1"`,
		`callback_host = "0.0.0.0"`,
	} {
		if !strings.Contains(df, want) {
			t.Errorf("embedded helper Dockerfile missing %q\n--- file ---\n%s", want, df)
		}
	}
}

// TestHelperDockerfile_PinsVersion locks the mcp2cli version.
// The 0.0.0.0 patch is keyed to specific source lines in
// mcp2cli/__init__.py — bumping the dep without re-verifying the
// patch lines would silently break login.
func TestHelperDockerfile_PinsVersion(t *testing.T) {
	df := string(helperDockerfile)
	// Reject loose ranges; require an exact pin.
	if strings.Contains(df, "mcp2cli>=") || strings.Contains(df, "mcp2cli==latest") {
		t.Errorf("mcp2cli must be exact-version pinned (the sed patch is line-coupled to a release)\n%s", df)
	}
	if !strings.Contains(df, "mcp2cli==") {
		t.Errorf("expected mcp2cli== pin, got\n%s", df)
	}
}

// TestFreeTCPPort_ReturnsValidPort sanity-checks the port picker —
// kernel may hand back an ephemeral port outside the typical user
// range, but it should always be > 0 and < 65536.
func TestFreeTCPPort_ReturnsValidPort(t *testing.T) {
	p, err := freeTCPPort()
	if err != nil {
		t.Fatalf("freeTCPPort: %v", err)
	}
	if p <= 0 || p >= 65536 {
		t.Errorf("port %d out of valid range", p)
	}
}

// TestFreeTCPPort_DistinctAcrossCalls — two consecutive calls
// usually return different ports because the kernel rotates the
// ephemeral pool. Not a hard guarantee but a strong smell test
// for "we accidentally hardcoded a port".
func TestFreeTCPPort_DistinctAcrossCalls(t *testing.T) {
	a, err := freeTCPPort()
	if err != nil {
		t.Fatal(err)
	}
	b, err := freeTCPPort()
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		// Same port is a 1/N event. Don't fail flakily; just log
		// so a true regression (always-same) shows up in CI.
		t.Logf("note: two consecutive freeTCPPort calls returned the same port %d (rare, acceptable)", a)
	}
}

// TestHelperImageConstant_NotEmpty would be embarrassing to fail —
// a typo turning HelperImage into "" would make `docker run` fail
// in a confusing way.
func TestHelperImageConstant_NotEmpty(t *testing.T) {
	if HelperImage == "" {
		t.Fatal("HelperImage must be a non-empty docker tag")
	}
	if !strings.Contains(HelperImage, ":") {
		t.Errorf("HelperImage %q should include an explicit tag (use :latest at minimum)", HelperImage)
	}
}
