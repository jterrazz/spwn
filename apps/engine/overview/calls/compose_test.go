package calls

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
	"github.com/jterrazz/studio-station/apps/engine/core/store"
	machinefilecalls "github.com/jterrazz/studio-station/apps/engine/machinefile/calls"
	"github.com/jterrazz/studio-station/apps/engine/overview"
)

func TestARefusedFileOfTheOwnerIsANeedWithTheFileToFix(t *testing.T) {
	head := overview.Head{Name: "laptop", Kind: machine.Computer, ThisMac: true}
	refused := &Refusal{Text: "~/.config/station/catalog/apps/jq.yaml:3: unknown key", File: "~/.config/station/catalog/apps/jq.yaml"}

	got := summarize(head, nil, nil, nil, nil, refused).Needs
	want := overview.Need{Machine: "laptop", Section: section.Overview, Broken: true, Severity: finding.Broken, Subject: overview.RefusedSubject,
		Text: refused.Text, Path: refused.File}
	if len(got) != 1 || !reflect.DeepEqual(got[0], want) {
		t.Fatalf("needs %+v, want exactly %+v", got, want)
	}
	if none := summarize(head, nil, nil, nil, nil, nil).Needs; len(none) != 0 {
		t.Errorf("no refused file, yet needs %+v", none)
	}
}

func TestALedgerAnOlderStationWroteIsANeedFixedByInit(t *testing.T) {
	laptop := machine.Ref{Name: "laptop"}
	older := &store.UnreadableError{Path: "~/.local/state/station/ledger.json", Cause: &store.OlderError{Version: 1, Current: 2}}
	found := overview.AppsRead(nil, laptop, nightsOf(machinefilecalls.Nightly{}, older, time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC))).Findings
	if len(found) != 1 || found[0].Fix.Command != "station init" || found[0].Severity != finding.Broken || !strings.Contains(found[0].Text, "written at version 1 by an older station") {
		t.Fatalf("findings %+v, want one broken line saying the ledger's version, fixed by station init", found)
	}
	if other := overview.AppsRead(nil, laptop, nightsOf(machinefilecalls.Nightly{}, errors.New("no such call"), time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC))).Findings; len(other) != 0 {
		t.Errorf("a night that cannot be asked for another reason, yet findings %+v", other)
	}
}
