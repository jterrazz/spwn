package calls

import (
	"reflect"
	"testing"

	"github.com/jterrazz/studio-station/src/core/finding"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/core/section"
	"github.com/jterrazz/studio-station/src/overview"
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
