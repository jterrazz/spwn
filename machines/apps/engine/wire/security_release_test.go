//go:build !world

package wire

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/platform/local/localtest"
)

func TestTheReleaseBuildRunsTheSystemsSecurityWhateverTheSystemRootSays(t *testing.T) {
	run := localtest.NewRunner(t)
	run.On("/usr/bin/security", "find-generic-password", "-s", "station-devices", "-a", "head", "-w").Reply(local.Output{ExitCode: 44})
	run.On("/usr/bin/security", "find-generic-password", "-s", "station", "-a", "github", "-w").Reply(local.Output{ExitCode: 44})
	vars := map[string]string{"STATION_SYSTEM_ROOT": "/sandbox"}
	if _, err := deviceAnchor(local.System{Run: run}, vars).Read(t.Context()); err != nil {
		t.Fatal(err)
	}
	_, _ = machine{kind: catalog.Mac, sys: local.System{Run: run}, vars: vars}.vault().Get(t.Context(), "github")
	if len(run.Calls) != 2 {
		t.Fatalf("calls %+v; want the anchor and the vault each to run /usr/bin/security", run.Calls)
	}
}
