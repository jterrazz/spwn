//go:build world

package wire

import (
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/secret"
	"github.com/jterrazz/studio-station/apps/engine/output"
)

func TestTheWorldBuildArmsTheTripwireWithTheWorldsSecrets(t *testing.T) {
	const vault, kept = "fake-vault-value-of-this-mac", "fake-secret-another-tool-keeps"
	path := filepath.Join(t.TempDir(), "world.json")
	world := `{"now": "2026-11-04T10:00:00Z", "this_mac": "here", "macs": {"here": {"vault": {"agent": "` +
		base64.StdEncoding.EncodeToString([]byte(vault)) + `"}, "secrets": {"tool": "` + kept + `"}}}}`
	if err := os.WriteFile(path, []byte(world), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(secret.Disarm)
	if _, err := openWorld(map[string]string{"STATION_WORLD": path}, true); err != nil {
		t.Fatal(err)
	}
	for _, leaked := range []string{vault, kept} {
		said := func() (said string) {
			defer func() {
				if r := recover(); r != nil {
					said, _ = r.(string)
				}
			}()
			_ = output.NewWriter(output.Stream{W: io.Discard}, output.Stream{W: io.Discard}, nil).Raw([]byte("token " + leaked + "\n"))
			return ""
		}()
		if !strings.HasPrefix(said, "tripwire: an answer carries a secret") || strings.Contains(said, leaked) {
			t.Errorf("an answer echoing the world's %q panicked with %q, want the tripwire's words without it", leaked, said)
		}
	}
}
