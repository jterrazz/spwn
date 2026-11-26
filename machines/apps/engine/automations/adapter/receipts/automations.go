package receipts

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"spwn.sh/packages/automation"
	"spwn.sh/packages/project"

	"github.com/jterrazz/studio-station/apps/engine/automations"
	"github.com/jterrazz/studio-station/apps/engine/automations/contract"
)

type Automations struct{}

var _ contract.Automations = Automations{}

// value: where a project's manifest and the receipts of its runs lie, from its root.
const (
	manifestFile = "spwn.yaml"
	receiptsDir  = ".spwn"
	receiptsFile = "runs.jsonl"
)

type declared struct {
	group string
	auto  automations.Automation
}

func declaredIn(root string) ([]declared, error) {
	// invariant: the automations are read as the manifest declares them,
	// under each group it keeps them in; no root, or none with a manifest,
	// declares none.
	if root == "" {
		return nil, nil
	}
	p, err := project.Load(filepath.Join(root, manifestFile))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		var unreadable *fs.PathError
		if errors.As(err, &unreadable) {
			return nil, err
		}
		return nil, &automations.InvalidProjectError{Path: filepath.Join(root, manifestFile), Reason: err.Error()}
	}
	var out []declared
	for group, w := range p.Manifest.Worlds {
		for name, a := range w.Automations {
			out = append(out, declared{group: group, auto: automations.Automation{Group: group, Name: name, Agent: a.Agent, Trigger: trigger(a.On)}})
		}
	}
	slices.SortFunc(out, func(a, b declared) int {
		if c := strings.Compare(a.auto.Name, b.auto.Name); c != 0 {
			return c
		}
		return strings.Compare(a.group, b.group)
	})
	return out, nil
}

func trigger(on project.Trigger) automations.Trigger {
	switch {
	case on.Cron != "":
		return automations.Trigger{Kind: automations.Cron, Spec: on.Cron}
	case on.FS != nil:
		return automations.Trigger{Kind: automations.FS, Spec: on.FS.Path}
	}
	return automations.Trigger{}
}

func receiptsOf(root string) ([]automation.Receipt, int, error) {
	if root == "" {
		return nil, 0, nil
	}
	return automation.ReadReceipts(filepath.Join(root, receiptsDir, receiptsFile))
}

func read(all []declared, rows []automation.Receipt) []automations.Automation {
	out := make([]automations.Automation, 0, len(all))
	for _, d := range all {
		a := d.auto
		a.Group = d.group
		for _, r := range rows {
			if r.World != d.group || r.Automation != a.Name {
				continue
			}
			a.Fires++
			if !r.OK {
				a.Failed++
			}
			if r.Fired.After(a.LastFired) {
				a.LastFired, a.LastOK = r.Fired, r.OK
			}
		}
		out = append(out, a)
	}
	return automations.Mark(out)
}

func (Automations) List(_ context.Context, root string) ([]automations.Automation, error) {
	all, err := declaredIn(root)
	if err != nil || len(all) == 0 {
		return []automations.Automation{}, err
	}
	rows, _, err := receiptsOf(root)
	if err != nil {
		return nil, err
	}
	return read(all, rows), nil
}

func (a Automations) Automation(ctx context.Context, root, ref string) (automations.Automation, error) {
	all, err := a.List(ctx, root)
	if err != nil {
		return automations.Automation{}, err
	}
	return automations.Select(all, ref)
}

func (a Automations) Log(ctx context.Context, root, ref string, last int) (automations.Runs, error) {
	var chosen automations.Automation
	if ref != "" {
		found, err := a.Automation(ctx, root, ref)
		if err != nil {
			return automations.Runs{}, err
		}
		chosen = found
	}
	rows, unreadable, err := receiptsOf(root)
	if err != nil {
		return automations.Runs{}, err
	}
	var out []automations.Run
	for _, r := range rows {
		if ref == "" || (r.World == chosen.Group && r.Automation == chosen.Name) {
			out = append(out, run(r))
		}
	}
	total := len(out)
	if last > 0 && len(out) > last {
		out = out[len(out)-last:]
	}
	return automations.Runs{Runs: out, Total: total, Unreadable: unreadable}, nil
}

func run(r automation.Receipt) automations.Run {
	return automations.Run{Group: r.World, Name: r.Automation, Trigger: r.Trigger, Cause: r.Reason, Scheduled: r.Scheduled, At: r.Fired, DurationMS: r.DurationMS, OK: r.OK, Error: r.Error, Output: r.Output, Missed: r.Missed, Paths: len(r.EventPaths)}
}
