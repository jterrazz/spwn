package wire_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/project/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/project/adapter/manifest"
	"github.com/jterrazz/studio-station/apps/engine/project/contract"
	"github.com/jterrazz/studio-station/apps/engine/project/contract/contracttest"
)

// invariant: the fake projects are wired here, to the real adapter's
// reading of a layout, as the world wires them: no adapter imports
// another.
func TestTheFakeProjectsKeepTheContract(t *testing.T) {
	contracttest.Projects(t, func(t *testing.T) (contract.Projects, string) { return fake.New(manifest.Folders), t.TempDir() })
}
