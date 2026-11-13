package calls

import (
	"context"
	"path"
	"strings"

	model "github.com/jterrazz/studio-station/apps/engine/machines"

	background "github.com/jterrazz/studio-station/apps/engine/background/contract"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
)

const agentPrefix = "com.jterrazz.station.vm."

func (s *Machinist) LogPath(name string) string { return path.Join(s.o.Logs, name+".log") }

func (s *Machinist) KnownHostsPath(name string) string { return path.Join(s.o.KnownHosts, name) }

func (s *Machinist) KeyPath() string { return s.img.KeyPath() }

func (s *Machinist) VMUser() string {
	cfg, _ := s.prefs()
	return s.vmUser(cfg)
}

func (s *Machinist) ThisMacName() string {
	cfg, _ := s.prefs()
	return s.thisMac(cfg)
}

func (s *Machinist) ThisMacLabel() string {
	// value: this Mac's name without the owner's user before it, as the
	// fleet's other Macs are named ("mac-mini").
	cfg, _ := s.prefs()
	name, user := s.thisMac(cfg), cfg.Macs[cfg.ThisMachine].User
	if user == "" {
		user = s.h.User()
	}
	if short, ok := strings.CutPrefix(name, user+"-"); ok && short != "" {
		return short
	}
	return name
}

func (s *Machinist) oldAgent(name string) string {
	return path.Join(s.h.Home(), "Library", "LaunchAgents", agentPrefix+name+".plist")
}

func (s *Machinist) session() background.Domain { return background.Domain{UID: s.h.UID()} }

func (s *Machinist) oldJob(name string) background.Ref {
	return background.Ref{Domain: s.session(), Label: agentPrefix + name, File: s.oldAgent(name)}
}

// invariant: a VM still under a round-3 per-VM agent has that agent booted out
// and its plist removed by the verb that stops or deletes it, so the setup
// migration's hint can always be followed.
func (s *Machinist) retireAgent(ctx context.Context, name string) {
	file := s.oldAgent(name)
	if !s.h.Exists(file) {
		return
	}
	_ = s.o.Jobs.Remove(ctx, s.oldJob(name))
}

type AgentMigration struct{ Retired []string }

func (s *Machinist) MigrateAgents(ctx context.Context) (AgentMigration, error) {
	// invariant: a migration never cuts a running VM: one VM running under an old
	// agent refuses the whole step before any bootout.
	dir := path.Join(s.h.Home(), "Library", "LaunchAgents")
	entries, readErr := s.h.Names(dir)
	if readErr != nil {
		return AgentMigration{}, nil //nolint:nilerr // outside: no LaunchAgents folder holds no agent to migrate.
	}
	var names []string
	for _, n := range entries {
		if strings.HasPrefix(n, agentPrefix) && strings.HasSuffix(n, ".plist") {
			names = append(names, strings.TrimSuffix(strings.TrimPrefix(n, agentPrefix), ".plist"))
		}
	}
	if len(names) == 0 {
		return AgentMigration{}, nil
	}
	var running []string
	for _, n := range names {
		if s.engineState(ctx, n) == contract.DiskRunning {
			running = append(running, n)
		}
	}
	if len(running) > 0 {
		return AgentMigration{}, &model.RunningUnderAgentError{Names: running}
	}
	overrides := s.o.Jobs.Overrides(ctx, s.session())
	records, _ := s.records()
	for _, n := range names {
		_ = s.o.Jobs.Remove(ctx, s.oldJob(n))
		for _, r := range records {
			if r.Name != n || r.Lifetime != model.Persistent {
				continue
			}
			r.Want = model.WantRunning
			if enabled, listed := overrides[agentPrefix+n]; listed && !enabled {
				r.Want = model.WantStopped
			}
			_ = s.writeRecord(r)
		}
	}
	return AgentMigration{Retired: names}, nil
}
