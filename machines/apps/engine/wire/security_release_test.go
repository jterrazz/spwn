//go:build !world

package wire

import (
	"context"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/platform/local/localtest"
)

// value: a runner that answers every program as a keychain with no such
// item, and keeps the path of each program it was asked to run.
type keptRun struct {
	*localtest.Runner
	ran []string
}

func (r *keptRun) Output(_ context.Context, c local.Cmd) (local.Output, error) {
	r.ran = append(r.ran, c.Name)
	return local.Output{ExitCode: 44}, nil
}

// value: a vault book of one entry, kept in memory.
type oneEntry struct{ b accounts.Book }

func (o *oneEntry) Read(context.Context) (accounts.Book, error) { return o.b, nil }

func (o *oneEntry) Change(_ context.Context, edit func(*accounts.Book) error) error {
	return edit(&o.b)
}

func TestTheReleaseBuildRunsTheSystemsKeychainToolWhateverTheSystemRootSays(t *testing.T) {
	run := &keptRun{Runner: localtest.NewRunner(t)}
	vars := map[string]string{"STATION_SYSTEM_ROOT": "/sandbox"}
	if _, err := deviceAnchor(local.System{Run: run}, vars).Read(t.Context()); err != nil {
		t.Fatal(err)
	}
	_, _ = machine{kind: catalog.Mac, sys: local.System{Run: run}, vars: vars}.vault().Get(t.Context(), "github")
	sys := local.New(nil, nil, nil, nil)
	sys.Run = run
	book := &oneEntry{b: accounts.Book{Entries: []accounts.VaultEntry{{ID: "github", Label: "github"}}}}
	kept := machine{kind: catalog.Mac, sys: sys, vars: vars, stores: &stores{book: book}}.vaultKept(nil)
	if _, err := kept.Passwords().Remove(t.Context(), "github"); err != nil {
		t.Fatal(err)
	}
	if len(run.ran) != 3 {
		t.Fatalf("ran %q; want the anchor, the vault and the vault's passwords each to run their tool once", run.ran)
	}
	for _, name := range run.ran {
		if strings.HasPrefix(name, "/sandbox") {
			t.Fatalf("ran %q; want no program under the system root in a release build", run.ran)
		}
	}
}
