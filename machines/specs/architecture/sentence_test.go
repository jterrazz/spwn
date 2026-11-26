package architecture_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/approvals"
)

// invariant: the sentence of Local Network is one: the engine's row, the
// window bundle's NSLocalNetworkUsageDescription and the tier check agree
// byte for byte.
func TestLocalNetworkSentenceIsTheBundlesPrompt(t *testing.T) {
	root := repoRoot(t)
	plist, err := os.ReadFile(filepath.Join(root, "apps", "desktop", "src-tauri", "Info.plist"))
	if err != nil {
		t.Fatal(err)
	}
	_, after, ok := strings.Cut(string(plist), "<key>NSLocalNetworkUsageDescription</key>")
	if !ok {
		t.Fatal("Info.plist has no NSLocalNetworkUsageDescription")
	}
	_, after, _ = strings.Cut(after, "<string>")
	prompt, _, _ := strings.Cut(after, "</string>")
	if prompt != approvals.LocalNetworkSentence {
		t.Errorf("Info.plist says %q, the approvals table says %q", prompt, approvals.LocalNetworkSentence)
	}
	tier, err := os.ReadFile(filepath.Join(root, "specs", "machine", "macos", "tier.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(tier), `"`+approvals.LocalNetworkSentence+`"`) {
		t.Errorf("specs/machine/macos/tier.sh does not hold %q", approvals.LocalNetworkSentence)
	}
}
