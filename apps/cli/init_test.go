package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"github.com/spf13/cobra"

	"spwn.sh/packages/dependency"
)

func TestParseExampleRef(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		// Explicit spwn:<slug> form always accepted.
		{"spwn:matrix", "matrix", false},
		{"spwn:startup", "startup", false},
		// Bare names that match a gallery entry auto-resolve. This is
		// the docs-friendly shorthand: `spwn init matrix`.
		{"matrix", "matrix", false},
		{"startup", "startup", false},
		// Bare names that don't match any gallery entry error out.
		// `qmd` is a real catalog tool but NOT a gallery entry (no
		// worlds: section) so init rejects it here — the caller needs
		// to either pick a gallery slug or use `agent add --dep`.
		{"qmd", "", true},
		{"nonesuch", "", true},
		// Malformed / legacy shapes still error.
		{"@other/matrix", "", true},
		{"spwn:", "", true},
		{"spwn:ma trix", "", true},
		{"spwn:foo/bar", "", true},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got, err := parseExampleRef(c.in)
			if c.wantErr {
				if err == nil {
					t.Errorf("parseExampleRef(%q) = %q, want error", c.in, got)
				}
				return
			}
			if err != nil {
				t.Errorf("parseExampleRef(%q) returned error: %v", c.in, err)
			}
			if got != c.want {
				t.Errorf("parseExampleRef(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// withTempCwd chdirs into a fresh temp directory for the duration of
// the test, restoring the prior working directory on cleanup.
func withTempCwd(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	return tmp
}

// TestRunInitExample_InstallsFromCatalog verifies the init example
// path materialises an example into a temp project dir. Invokes the
// RunE directly to avoid cobra's shared-state issues with --help
// flags leaking across tests in the package.
func TestRunInitExample_InstallsFromCatalog(t *testing.T) {
	if _, err := dependency.GalleryEntryBySlug("matrix"); err != nil {
		t.Skipf("matrix example not bundled: %v", err)
	}

	tmp := withTempCwd(t)

	initName = ""
	initForce = false
	initGlobal = false

	cmd := &cobra.Command{}
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))

	if err := runInitExample(cmd, "spwn:matrix"); err != nil {
		t.Fatalf("runInitExample: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "spwn.yaml")); err != nil {
		t.Fatalf("spwn.yaml missing after init: %v", err)
	}
	agentsDir := filepath.Join(tmp, "spwn", "agents")
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		t.Fatalf("read agents dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("no agents written under %s", agentsDir)
	}
}

func TestRunInitExample_RejectsBadRef(t *testing.T) {
	withTempCwd(t)

	initName = ""
	initForce = false
	initGlobal = false

	cmd := &cobra.Command{}
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))

	// A bare name that isn't a gallery entry must still error — the
	// resolver only auto-promotes when the catalog has the slug.
	if err := runInitExample(cmd, "nonesuch-gallery-slug"); err == nil {
		t.Fatalf("expected error for bad ref, got nil")
	}
}

// TestRunInitExample_AcceptsBareGallerySlug is the counterpart: a
// bare name that matches a gallery entry must install the example
// through the catalog resolver, same as the explicit spwn:<slug> form.
func TestRunInitExample_AcceptsBareGallerySlug(t *testing.T) {
	if _, err := dependency.GalleryEntryBySlug("matrix"); err != nil {
		t.Skipf("matrix example not bundled: %v", err)
	}

	tmp := withTempCwd(t)

	initName = ""
	initForce = false
	initGlobal = false

	cmd := &cobra.Command{}
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))

	if err := runInitExample(cmd, "matrix"); err != nil {
		t.Fatalf("runInitExample(bare): %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "spwn.yaml")); err != nil {
		t.Fatalf("spwn.yaml missing after bare-name init: %v", err)
	}
}

func TestRunInitExample_RejectsNameFlag(t *testing.T) {
	withTempCwd(t)

	initName = "custom"
	initForce = false
	initGlobal = false
	t.Cleanup(func() { initName = "" })

	cmd := &cobra.Command{}
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))

	if err := runInitExample(cmd, "spwn:matrix"); err == nil {
		t.Fatalf("expected error when --name is combined with an example ref")
	}
}
