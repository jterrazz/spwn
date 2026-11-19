package calls

import (
	"context"
	activitycalls "github.com/jterrazz/studio-station/apps/engine/activity/calls"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/background"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call/lanes"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/overview"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

const (
	settingRestartAfterPowerCut = "restart-after-power-cut"
	settingNeverSleeps          = "never-sleeps"
	settingAutoLogin            = "autologin"
)

func (r *Reader[App, Item, Misplaced]) machineJobs() []job[App, Item, Misplaced] {
	type sn = snapshot[App, Item, Misplaced]
	return []job[App, Item, Misplaced]{
		{Key: subject + "/machine/os", Budget: defaultBudget + margin,
			Do: func(ctx context.Context) func(*sn) {
				name := r.kernel(ctx)
				return func(s *sn) { s.Machine.OS = name }
			},
			Late: func(*sn) {},
		},
		{Key: subject + "/machine/facts", Budget: defaultBudget + margin,
			Do: func(ctx context.Context) func(*sn) {
				uptime, battery, memory := r.system(ctx)
				return func(s *sn) {
					s.System.Uptime, s.System.Battery, s.System.MemoryTotal, s.System.Pending = uptime, battery, memory, false
				}
			},
			Late: func(s *sn) { s.System.Pending = false },
		},
		{Key: subject + "/machine/sample", Budget: defaultBudget + margin,
			Do: func(ctx context.Context) func(*sn) {
				reading, err := activitycalls.Measure(ctx, r.d.Meter)
				return func(s *sn) {
					if err != nil {
						s.Err = err
						return
					}
					s.System.Reading = reading
				}
			},
			Late: func(*sn) {},
		},
		{Key: subject + "/machine/setup", Budget: partBudget,
			Do: func(ctx context.Context) func(*sn) {
				steps, err := r.d.Setup.Check(ctx)
				st := overview.SetupOf(steps)
				if err != nil {
					st = overview.Setup{State: catalog.StateUnknown, Steps: []overview.Step{}}
				}
				return func(s *sn) { s.Setup = st }
			},
			Late: func(s *sn) { s.Setup = overview.Setup{State: catalog.StateUnknown, Steps: []overview.Step{}} },
		},
		{Key: subject + "/machine/daemon", Budget: defaultBudget + margin,
			Do: func(ctx context.Context) func(*sn) {
				d := r.engine(ctx)
				return func(s *sn) { s.Station.Engine = d }
			},
			Late: func(s *sn) { s.Station.Engine = overview.Check{Name: "engine", State: catalog.StateAbsent} },
		},
		{Key: subject + "/machine/power", Budget: defaultBudget + margin,
			Do: func(ctx context.Context) func(*sn) {
				if r.d.ThisMac.Kind != catalog.Mac || r.d.Awake == nil {
					return func(*sn) {}
				}
				p, ok := r.d.Awake.Session(ctx)
				return func(s *sn) { s.Awake, s.AwakeKnown = p, ok }
			},
			Late: func(*sn) {},
		},
	}
}

// invariant: the readiness reads the settings first, and the Mac's facts
// only when the Mac needs a readiness at all.
func (r *Reader[App, Item, Misplaced]) readinessJob() job[App, Item, Misplaced] {
	type sn = snapshot[App, Item, Misplaced]
	return job[App, Item, Misplaced]{Key: subject + "/readiness", Budget: partBudget,
		Do: func(ctx context.Context) func(*sn) {
			if r.d.ThisMac.Kind != catalog.Mac {
				return func(*sn) {}
			}
			checks := r.readiness(ctx)
			return func(s *sn) { s.Readiness = checks }
		},
		Late: func(*sn) {},
	}
}

func (r *Reader[App, Item, Misplaced]) readiness(ctx context.Context) []overview.Check {
	states := map[string]catalog.State{}
	if items, err := r.d.Settings.List(ctx); err == nil {
		for _, it := range items {
			name, state := r.d.Settings.Placed(it)
			states[name] = state
		}
	}
	on := func(name string) machines.ReadinessSetting {
		state, ok := states[name]
		if !ok {
			state = catalog.StateUnknown
		}
		return machines.ReadinessSetting{Name: name, State: state, Root: r.needsRoot(name)}
	}
	f := machines.ReadinessFacts{
		Mac:                   r.label(),
		RestartsAfterPowerCut: on(settingRestartAfterPowerCut), NeverSleeps: on(settingNeverSleeps), LogsIn: on(settingAutoLogin),
		EngineKnown: r.d.ThisMac.Agent != nil,
	}
	if f.NeverSleeps.State != catalog.StateOK && r.d.Servers != nil {
		f.MacsServer = r.d.Servers.Hosted(ctx)
	}
	if machines.Readiness(f) == nil {
		return []overview.Check{}
	}
	f.FileVault, f.AuthRestart = r.factState(ctx, catalog.FactFileVault), r.factState(ctx, catalog.FactAuthRestart)
	f.EngineAtLogin = f.EngineKnown && r.d.ThisMac.Agent()
	items := machines.Readiness(f)
	out := make([]overview.Check, 0, len(items))
	for _, it := range items {
		out = append(out, overview.Check{Name: it.Name, State: it.State, Value: it.Value, Detail: it.Fix})
	}
	return out
}

func (r *Reader[App, Item, Misplaced]) needsRoot(setting string) bool {
	it, _ := r.d.Catalog.Setting(setting)
	return it.NeedsRoot
}

func (r *Reader[App, Item, Misplaced]) factState(ctx context.Context, f catalog.Fact) catalog.State {
	ctx, cancel := context.WithTimeout(ctx, defaultBudget)
	defer cancel()
	got, err := r.d.Facts.Read(ctx, f)
	if err != nil || ctx.Err() != nil {
		return catalog.StateUnknown
	}
	return got.State
}

func (r *Reader[App, Item, Misplaced]) label() string {
	name, _ := r.d.Name.Get()
	if short, ok := strings.CutPrefix(name, r.d.ThisMac.User+"-"); ok && short != "" && r.d.ThisMac.User != "" {
		return short
	}
	return name
}

func (r *Reader[App, Item, Misplaced]) diskJob() job[App, Item, Misplaced] {
	type sn = snapshot[App, Item, Misplaced]
	return job[App, Item, Misplaced]{Key: subject + "/workspace/disk", Lane: lanes.Inline,
		// invariant: the disk row follows the run's one storage reading, so its
		// frame shows the last-known total until the walk answers.
		Partial: func(ctx context.Context, emit func(func(*sn))) func(*sn) {
			last := DiskReading[Misplaced]{Rows: []overview.Disk{}, Misplaced: []Misplaced{}}
			if r.d.Disk != nil {
				for d := range r.d.Disk.Watch(ctx) {
					last = d
					emit(func(s *sn) { s.Workspace.Disk, s.Workspace.Misplaced = d.Rows, d.Misplaced })
				}
			}
			return func(s *sn) {
				s.Workspace.Disk, s.Workspace.Misplaced, s.Workspace.DiskPending = last.Rows, last.Misplaced, false
			}
		},
		Late: func(s *sn) { s.Workspace.DiskPending = false },
	}
}

func (r *Reader[App, Item, Misplaced]) sectionJobs() []job[App, Item, Misplaced] {
	type sn = snapshot[App, Item, Misplaced]
	return []job[App, Item, Misplaced]{
		{Key: subject + "/background", Budget: 2*defaultBudget + margin,
			Do: func(ctx context.Context) func(*sn) {
				jobs := []background.Task{}
				if r.mac() && r.d.Jobs != nil {
					listed, _ := r.d.Jobs.List(ctx)
					jobs = background.Judge(listed, r.d.ThisMac.Root, r.d.ThisMac.Home)
				}
				return func(s *sn) { s.Background = jobs }
			},
			Late: func(*sn) {},
		},
		{Key: subject + "/tailnet", Budget: partBudget,
			Do: func(ctx context.Context) func(*sn) {
				st, err := r.d.Tailnet.Status(ctx)
				if err != nil {
					st = ports.Status{State: catalog.StateUnknown}
				}
				return func(s *sn) { s.Tailnet = st; s.Gateways = overview.Gateways(r.d.ThisMac.Kind, st) }
			},
			Late: func(s *sn) { s.Tailnet = ports.Status{State: catalog.StateUnknown} },
		},
		{Key: subject + "/tools", Lane: lanes.Inline,
			Do: func(ctx context.Context) func(*sn) {
				reports, err := r.d.Apps.Inventory(ctx)
				return func(s *sn) {
					s.Apps, s.AppsPending = reports, false
					if err != nil {
						s.Err = err
					}
				}
			},
			Late: func(s *sn) { s.AppsPending = false },
		},
		{Key: subject + "/config", Budget: partBudget,
			Do: func(ctx context.Context) func(*sn) {
				reports, err := r.d.Settings.List(ctx)
				return func(s *sn) {
					s.Settings, s.SettingsPending = reports, false
					if err != nil {
						s.Err = err
						return
					}
					if r.mac() {
						s.Environment.Protections = append(s.Environment.Protections, r.protections(reports)...)
					}
				}
			},
			Late: func(s *sn) { s.SettingsPending = false },
		},
	}
}

func (r *Reader[App, Item, Misplaced]) protections(reports []Item) []overview.Check {
	var out []overview.Check
	for _, p := range overview.Protections() {
		for _, rep := range reports {
			if pkg, state := r.d.Settings.Placed(rep); pkg == p[0] {
				out = append(out, overview.Protection(state, p[1]))
				break
			}
		}
	}
	return out
}
