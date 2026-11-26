//go:build !world

package wire

import (
	"context"
	"strings"
	"testing"

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

func TestTheReleaseBuildRunsTheSystemsKeychainToolWhateverTheSystemRootSays(t *testing.T) {
	run := &keptRun{Runner: localtest.NewRunner(t)}
	vars := map[string]string{"STATION_SYSTEM_ROOT": "/sandbox"}
	if _, err := deviceAnchor(local.System{Run: run}, vars).Read(t.Context()); err != nil {
		t.Fatal(err)
	}
	_, _ = machine{kind: catalog.Mac, sys: local.System{Run: run}, vars: vars}.vault().Get(t.Context(), "github")
	if len(run.ran) != 2 {
		t.Fatalf("ran %q; want the anchor and the vault each to run their tool once", run.ran)
	}
	for _, name := range run.ran {
		if strings.HasPrefix(name, "/sandbox") {
			t.Fatalf("ran %q; want no program under the system root in a release build", run.ran)
		}
	}
}
