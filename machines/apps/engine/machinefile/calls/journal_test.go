package calls

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/journal"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
)

func TestARunThatCouldNotWriteTheSkillKeepsAFailedLineForIt(t *testing.T) {
	want := journal.Line{Text: "spwn skill", Detail: "not written: permission denied", Outcome: journal.Failed}
	for name, lines := range map[string][]journal.Line{
		"apply":  keptApply(ApplyIn{Machinefile: "dev"}, machinefile.Applied{SkillFailed: "permission denied"}, nil),
		"update": keptUpgrade(UpgradeIn{Machinefile: "dev"}, machinefile.Upgraded{SkillFailed: "permission denied"}, nil),
	} {
		if len(lines) != 2 || lines[1].Text != want.Text || lines[1].Detail != want.Detail || lines[1].Outcome != want.Outcome {
			t.Errorf("%s keeps %+v; want its own line, then %+v", name, lines, want)
		}
	}
	for name, lines := range map[string][]journal.Line{
		"apply":  keptApply(ApplyIn{Machinefile: "dev"}, machinefile.Applied{}, nil),
		"update": keptUpgrade(UpgradeIn{Machinefile: "dev"}, machinefile.Upgraded{}, nil),
	} {
		if len(lines) != 1 {
			t.Errorf("%s that wrote the skill keeps %+v; want its own line alone", name, lines)
		}
	}
}
