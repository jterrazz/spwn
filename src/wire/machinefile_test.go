package wire

import (
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/machinefile"
	machinefilecalls "github.com/jterrazz/studio-station/src/machinefile/calls"
)

func TestABrokenCatalogFileIsRefusedAtItsFileAndLineForNeedsYou(t *testing.T) {
	if refusalOf(nil) != nil {
		t.Fatal("no broken file, yet a refusal")
	}
	bad := &machinefile.RefusedFileError{Format: "catalog", Err: &catalog.FileError{File: "~/.config/station/catalog/apps/jq.yaml", Line: 3, Problem: "unknown key"}}
	got := refusalOf(machinefilecalls.Worded(bad))
	if got == nil || got.File != "~/.config/station/catalog/apps/jq.yaml" || !strings.Contains(got.Text, "~/.config/station/catalog/apps/jq.yaml:3") {
		t.Errorf("refusal %+v, want the file and its line", got)
	}
}
