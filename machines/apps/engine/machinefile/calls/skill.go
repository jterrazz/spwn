package calls

import (
	"context"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/apps"
	appscalls "github.com/jterrazz/studio-station/apps/engine/apps/calls"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
)

// invariant: the `spwn` skill is the last step of an apply (a create's boot
// phase included, never an image's bake, which is no machine yet) and of an
// update, the night's included, each run to its end: it says
// the machine as its Machinefile resolves (kind, system, mounts, port
// rights) and the apps that read present now (the
// run's shared reading dropped first, so what it installed is seen), each
// with its agent line; the skill's adapter writes it only where its bytes
// changed. An interrupted run leaves the last one as it was.
// The skill is best-effort: its run's effects are done when it is written,
// so a skill that cannot be written (its folder refused, the apps or the
// resolution unread) fails nothing; it answers why, the run's own outcome
// and exit staying what they were.
func spwnSkill(ctx context.Context, d Deps, name string) (failed string) {
	if d.Skill == nil {
		return ""
	}
	select {
	case <-ctx.Done():
		return ""
	default:
	}
	r, err := resolveValues(ctx, d, name)
	if err != nil {
		return why(err)
	}
	reports, err := call.Read(ctx, d.Calls, machine.Ref{}, appscalls.List, appscalls.ListIn{Fresh: true}, nil)
	if err != nil {
		return why(err)
	}
	var present []catalog.App
	for _, report := range reports {
		if slices.Contains([]apps.State{apps.OK, apps.UpdateAvailable, apps.NotManaged}, apps.StateOf(report)) {
			present = append(present, report.App)
		}
	}
	where := machinefile.Where{Name: d.Name, Kind: r.Kind, System: r.System, Values: r.Values}
	if _, err := d.Skill.Write(ctx, machinefile.Skill(where, present, catalog.SpwnDrive)); err != nil {
		return why(err)
	}
	return ""
}

func why(err error) string {
	line, _, _ := strings.Cut(err.Error(), "\n")
	return line
}
