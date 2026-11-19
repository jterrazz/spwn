package wire

import (
	"context"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	corejournal "github.com/jterrazz/studio-station/apps/engine/core/journal"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	journalcalls "github.com/jterrazz/studio-station/apps/engine/journal/calls"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/remote/each"
	journalcmd "github.com/jterrazz/studio-station/apps/engine/surface/cli/journal"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"

	"github.com/spf13/cobra"
)

func journalCommand(env kit.Env, m machine) *cobra.Command {
	return journalcmd.Command(journalcmd.Deps{ThisMachine: m.where}, env)
}

func (m machine) journalDeps() journalcalls.Deps {
	return journalcalls.Deps{Journal: m.stores.journal, Now: m.sys.Clock.Now, Macs: journalMacs{m}, Known: knownTo(m.cat)}
}

// value: the journal's reach: every Mac as This Mac lists it, each Mac's
// lines through this run's Router.
type journalMacs struct{ m machine }

func (j journalMacs) Machines(ctx context.Context) (machines.AllMachines, error) {
	return settingsAllMachines(j).Machines(ctx)
}

func (j journalMacs) Kept(ctx context.Context, macs []string, in journalcalls.KeptIn) map[string]journalcalls.Answer {
	r := j.m.far.router
	got := each.All(ctx, macs, func(ctx context.Context, mac string) ([]corejournal.Line, error) {
		out, err := call.Invoke(ctx, r, coremachine.Ref{Name: mac}, journalcalls.Kept, in, nil)
		if err == nil && out.Lines == nil {
			out.Lines = []corejournal.Line{}
		}
		return out.Lines, err
	})
	out := make(map[string]journalcalls.Answer, len(macs))
	for mac, a := range got {
		if a.Err != nil {
			out[mac] = journalcalls.Answer{Why: kit.Screened(a.Err, "journal").Message}
			continue
		}
		out[mac] = journalcalls.Answer{Lines: a.Value}
	}
	return out
}

// value: whether the catalog this run reads still knows a line's package
// or image.
func knownTo(cat catalog.Catalog) func(corejournal.Subject) bool {
	return func(s corejournal.Subject) bool {
		switch s.Kind {
		case corejournal.Setting:
			return slices.ContainsFunc(cat.Settings, func(c catalog.Setting) bool { return c.Name == s.Name })
		case corejournal.Image:
			return slices.ContainsFunc(cat.Images, func(c catalog.Image) bool { return c.Name == s.Name })
		case corejournal.App, corejournal.Skill, corejournal.Machine:
		}
		return slices.ContainsFunc(cat.Apps, func(c catalog.App) bool { return c.Name == s.Name })
	}
}
