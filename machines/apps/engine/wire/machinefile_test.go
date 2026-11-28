package wire

import (
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	corejsonfile "github.com/jterrazz/studio-station/apps/engine/core/adapter/jsonfile"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/adapter/git"
	machinefilecalls "github.com/jterrazz/studio-station/apps/engine/machinefile/calls"
	machinefilecontract "github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/platform/local/localtest"
	"github.com/jterrazz/studio-station/apps/engine/setup"
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

func TestTheSharedSourceIsReadInTheFolderOfThisMachinesProfileName(t *testing.T) {
	paths := setup.Paths{Prefs: "/prefs", Cache: "/cache"}
	sys := local.New(nil, nil, io.Discard, io.Discard)
	sys.FS = localtest.NewFS()
	shared := machinefilecontract.SharedSource{Name: "studio", URL: "https://example.com/studio.git"}
	prefs := corejsonfile.NewPreferences(sys.FS, paths.Prefs, preferenceGroups())
	if err := prefs.Write(machinefilecontract.GroupSource, map[string]any{"source": shared}); err != nil {
		t.Fatal(err)
	}
	m := machine{sys: sys, paths: paths, stores: &stores{prefs: prefs}, profile: coremachine.Profile{Ref: coremachine.Ref{Name: "mini"}}}
	sources := m.machinefileSources()
	if len(sources) != 3 {
		t.Fatalf("%d sources; want the owner's, the shared one and the built-in", len(sources))
	}
	if !reflect.DeepEqual(sources[1], git.New(sys.Run, sys.FS, paths.Cache, shared, "mini")) {
		t.Errorf("the shared source is %+v; want it read by this machine's profile name, mini", sources[1])
	}
	if reflect.DeepEqual(sources[1], git.New(sys.Run, sys.FS, paths.Cache, shared, "other")) {
		t.Error("the shared source reads as another computer's")
	}
}

func TestARefusedCatalogFileSaysItsProblemWithItsTokenWithheld(t *testing.T) {
	bad := &machinefile.RefusedFileError{Format: "catalog", Err: &catalog.FileError{File: "~/.config/station/catalog/apps/jq.yaml", Line: 3, Problem: "unknown key https://bot:hunter2@proxy.example/x"}}
	got := refusalOf(machinefilecalls.Worded(bad))
	if got == nil || strings.Contains(got.Text, "hunter2") || !strings.Contains(got.Text, "https://withheld@proxy.example/x") {
		t.Errorf("refusal %+v, want its credential withheld", got)
	}
}
