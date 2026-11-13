package calls

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	activitycalls "github.com/jterrazz/studio-station/apps/engine/activity/calls"
	appscontract "github.com/jterrazz/studio-station/apps/engine/apps/contract"
	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/background"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call/lanes"
	"github.com/jterrazz/studio-station/apps/engine/overview"
	"github.com/jterrazz/studio-station/apps/engine/ports"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	settingscontract "github.com/jterrazz/studio-station/apps/engine/settings/contract"
)

const (
	subject       = "status"
	defaultBudget = 2 * time.Second
	updateBudget  = 10 * time.Second
	ghBudget      = 5 * time.Second
	margin        = time.Second
	partBudget    = 60 * time.Second
)

type Reader[App, Item, Misplaced any] struct{ d Deps[App, Item, Misplaced] }

type snapshot[App, Item, Misplaced any] = overview.Snapshot[App, Item, Misplaced]

func New[App, Item, Misplaced any](d Deps[App, Item, Misplaced]) *Reader[App, Item, Misplaced] {
	return &Reader[App, Item, Misplaced]{d: d}
}

func (r *Reader[App, Item, Misplaced]) mac() bool { return r.d.ThisMac.Kind != catalog.Linux }

func (r *Reader[App, Item, Misplaced]) Watch(ctx context.Context) <-chan snapshot[App, Item, Misplaced] {
	return lanes.Stream(ctx, r.d.Pool, subject, r.watch)
}

func (r *Reader[App, Item, Misplaced]) Snapshot(ctx context.Context) (snapshot[App, Item, Misplaced], error) {
	snap := lanes.Last(r.Watch(ctx))
	if err := ctx.Err(); err != nil {
		return snapshot[App, Item, Misplaced]{}, err
	}
	return snap, snap.Err
}

func (r *Reader[App, Item, Misplaced]) Sample(ctx context.Context) (activity.Reading, error) {
	return activitycalls.Measure(ctx, r.d.Meter)
}

func (r *Reader[App, Item, Misplaced]) Forget() {
	// invariant: a reload of the whole machine drops every shared reading of
	// the run (the inventory, the walk, the shared commands), not only its own.
	r.d.Pool.Forget("")
}

type row struct {
	name   string
	budget time.Duration
	read   func(ctx context.Context) (overview.Check, error)
}

func pendingChecks(table []row) []overview.Check {
	checks := make([]overview.Check, len(table))
	for i, p := range table {
		checks[i] = overview.Check{Name: p.name, State: catalog.StatePending}
	}
	return checks
}

// invariant: a row past its budget answers late, whatever its tool says.
func fact(i settingscontract.Inspector, name string, f catalog.Fact, budget time.Duration, judge func(settings.Fact) overview.Check) row {
	if judge == nil {
		judge = func(got settings.Fact) overview.Check { return overview.FactRow(name, got) }
	}
	return row{name: name, budget: budget, read: func(ctx context.Context) (overview.Check, error) {
		got, err := i.Read(ctx, f)
		if err != nil {
			return overview.Check{}, err
		}
		return judge(got), nil
	}}
}

func (r *Reader[App, Item, Misplaced]) networkRows() []row {
	return []row{
		fact(r.d.Facts, "local ip", catalog.FactLocalIP, defaultBudget, nil),
		fact(r.d.Facts, "dns", catalog.FactDNS, defaultBudget, nil),
	}
}

func (r *Reader[App, Item, Misplaced]) protectionRows() []row {
	return []row{
		fact(r.d.Facts, "filevault", catalog.FactFileVault, defaultBudget, nil),
		fact(r.d.Facts, "firewall", catalog.FactFirewall, defaultBudget, nil),
		fact(r.d.Facts, "sip", catalog.FactSIP, defaultBudget, nil),
		fact(r.d.Facts, "gatekeeper", catalog.FactGatekeeper, defaultBudget, nil),
	}
}

func (r *Reader[App, Item, Misplaced]) updateRows() []row {
	rows := []row{fact(r.d.Facts, "macos", catalog.FactSystemUpdates, updateBudget, nil)}
	if u := r.d.Updates; u != nil {
		rows = append(rows, row{name: u.Manager(), budget: updateBudget, read: func(ctx context.Context) (overview.Check, error) {
			n, err := u.Behind(ctx)
			if ctx.Err() != nil {
				return overview.Check{}, ctx.Err()
			}
			return overview.AppUpdates(u.Manager(), n, errors.Is(err, appscontract.ErrMissing), err != nil), nil
		}})
	}
	return rows
}

func (r *Reader[App, Item, Misplaced]) identityRows() []row {
	want, _ := r.d.Identity.Get()
	git := func(name string, expected string) func(settings.Fact) overview.Check {
		return func(got settings.Fact) overview.Check { return overview.GitValue(name, got, expected) }
	}
	return []row{
		fact(r.d.Owner, "git-email", catalog.FactGitEmail, defaultBudget, git("git-email", want.Email())),
		fact(r.d.Owner, "git-name", catalog.FactGitName, defaultBudget, git("git-name", want.Name())),
		fact(r.d.Owner, "git-signing", catalog.FactGitSigning, defaultBudget, nil),
		fact(r.d.Owner, "gpg-key", catalog.FactGPGKey, defaultBudget, nil),
		fact(r.d.Owner, "github", catalog.FactGitHub, ghBudget, nil),
		fact(r.d.Owner, "ssh-key", catalog.FactSSHKey, defaultBudget, nil),
	}
}

// invariant: the frame is what a read knows from preferences alone: every
// section and every row a table declares, each pending; a job per row then
// fills it, so a slow tool delays only its own row.
func (r *Reader[App, Item, Misplaced]) frame() snapshot[App, Item, Misplaced] {
	h := r.d.ThisMac
	snap := snapshot[App, Item, Misplaced]{
		GeneratedAt:     r.d.Clock.Now().UTC(),
		Station:         overview.Station{Version: h.Version, Commit: h.Commit, Engine: overview.Check{Name: "engine", State: catalog.StatePending}},
		Tailnet:         ports.Status{State: catalog.StatePending},
		Setup:           overview.Setup{State: catalog.StatePending, Steps: []overview.Step{}},
		System:          overview.System{Pending: true, Reading: activity.Reading{Top: activity.Busiest{CPU: []activity.Process{}, Memory: []activity.Process{}}}},
		Identity:        pendingChecks(r.identityRows()),
		Background:      []background.Task{},
		Gateways:        []overview.Check{},
		Environment:     overview.Environment{Network: pendingChecks(r.networkRows())},
		Workspace:       overview.Workspace[Misplaced]{Disk: []overview.Disk{}, Misplaced: []Misplaced{}, DiskPending: true},
		AppsPending:     true,
		SettingsPending: true,
	}
	if r.mac() {
		snap.Environment.Ports = []overview.Port{}
		snap.Environment.Protections = pendingChecks(r.protectionRows())
		snap.Environment.Updates = pendingChecks(r.updateRows())
	}
	thisMac, err := r.d.Name.Get()
	snap.Machine = overview.Machine{
		Name: thisMac, Registered: err == nil && thisMac != "", Kind: h.Kind,
		Hostname: h.Hostname, OS: h.OS, Arch: h.Arch, User: h.User, Shell: overview.ShellName(h.Shell),
	}
	if !snap.Machine.Registered {
		snap.Machine.Name = ""
	}
	return snap
}

type job[App, Item, Misplaced any] = lanes.Job[snapshot[App, Item, Misplaced]]

func (r *Reader[App, Item, Misplaced]) watch(ctx context.Context) <-chan snapshot[App, Item, Misplaced] {
	jobs := r.machineJobs()
	jobs = append(jobs, r.rowJobs("environment/network", r.networkRows(), func(sn *snapshot[App, Item, Misplaced]) []overview.Check { return sn.Environment.Network })...)
	if r.mac() {
		jobs = append(jobs, r.rowJobs("environment/protections", r.protectionRows(), func(sn *snapshot[App, Item, Misplaced]) []overview.Check { return sn.Environment.Protections })...)
		jobs = append(jobs, r.rowJobs("environment/updates", r.updateRows(), func(sn *snapshot[App, Item, Misplaced]) []overview.Check { return sn.Environment.Updates })...)
		jobs = append(jobs, job[App, Item, Misplaced]{Key: subject + "/environment/ports", Budget: defaultBudget + margin,
			Do: func(ctx context.Context) func(*snapshot[App, Item, Misplaced]) {
				found := r.listening(ctx)
				return func(sn *snapshot[App, Item, Misplaced]) { sn.Environment.Ports = found }
			},
			Late: func(*snapshot[App, Item, Misplaced]) {},
		})
	}
	jobs = append(jobs, r.diskJob())
	jobs = append(jobs, r.rowJobs("identity", r.identityRows(), func(sn *snapshot[App, Item, Misplaced]) []overview.Check { return sn.Identity })...)
	jobs = append(jobs, r.sectionJobs()...)
	jobs = append(jobs, r.readinessJob())
	return lanes.Run(ctx, r.d.Pool, r.frame(), jobs, clone[App, Item, Misplaced])
}

func clone[App, Item, Misplaced any](sn snapshot[App, Item, Misplaced]) snapshot[App, Item, Misplaced] {
	sn.Identity = slices.Clone(sn.Identity)
	sn.Environment.Network = slices.Clone(sn.Environment.Network)
	sn.Environment.Protections = slices.Clone(sn.Environment.Protections)
	sn.Environment.Updates = slices.Clone(sn.Environment.Updates)
	sn.Gateways = slices.Clone(sn.Gateways)
	sn.Readiness = slices.Clone(sn.Readiness)
	return sn
}

func (r *Reader[App, Item, Misplaced]) rowJobs(key string, table []row, rows func(*snapshot[App, Item, Misplaced]) []overview.Check) []job[App, Item, Misplaced] {
	out := make([]job[App, Item, Misplaced], 0, len(table))
	for i, p := range table {
		out = append(out, job[App, Item, Misplaced]{
			Key: subject + "/" + key + "/" + p.name, Budget: p.budget + margin,
			Do: func(ctx context.Context) func(*snapshot[App, Item, Misplaced]) {
				check := read(ctx, p)
				return func(sn *snapshot[App, Item, Misplaced]) { rows(sn)[i] = check }
			},
			Late: func(sn *snapshot[App, Item, Misplaced]) { rows(sn)[i] = overview.Late(p.name, p.budget) },
		})
	}
	return out
}

func read(ctx context.Context, p row) overview.Check {
	ctx, cancel := context.WithTimeout(ctx, p.budget)
	defer cancel()
	check, err := p.read(ctx)
	if err != nil || ctx.Err() != nil {
		return overview.Late(p.name, p.budget)
	}
	return check
}

func (r *Reader[App, Item, Misplaced]) listening(ctx context.Context) []overview.Port {
	if r.d.Listeners == nil {
		return []overview.Port{}
	}
	ctx, cancel := context.WithTimeout(ctx, defaultBudget)
	defer cancel()
	ls, err := r.d.Listeners.Read(ctx)
	if err != nil || ctx.Err() != nil {
		return []overview.Port{}
	}
	return overview.Listening(ls)
}

func (r *Reader[App, Item, Misplaced]) kernel(ctx context.Context) string {
	ctx, cancel := context.WithTimeout(ctx, defaultBudget)
	defer cancel()
	if got, err := r.d.Facts.Read(ctx, catalog.FactKernel); err == nil && got.State == catalog.StateOK && got.Value != "" {
		return got.Value
	}
	return r.d.ThisMac.OS
}

func (r *Reader[App, Item, Misplaced]) system(ctx context.Context) (uptime, battery string, memory int64) {
	ctx, cancel := context.WithTimeout(ctx, defaultBudget)
	defer cancel()
	if s, err := r.d.Meter.Sample(ctx); err == nil {
		if !s.Boot.IsZero() {
			uptime = activity.Uptime(s.At.Sub(s.Boot))
		}
		memory = s.MemTotal
	}
	if r.d.Battery != nil {
		if b, err := r.d.Battery.Read(); err == nil && b.HasBattery {
			battery = batteryLine(b)
		}
	}
	return uptime, battery, memory
}

func batteryLine(b awake.Reading) string {
	state := "not charging"
	switch {
	case b.Charging:
		state = "charging"
	case b.OnBattery():
		state = "on battery"
	case b.Percent >= 100:
		state = "charged"
	}
	return strconv.Itoa(b.Percent) + "% " + state
}

func (r *Reader[App, Item, Misplaced]) engine(ctx context.Context) overview.Check {
	absent := overview.Check{Name: "engine", State: catalog.StateAbsent}
	if r.d.ThisMac.Kind != catalog.Mac || r.d.Engine == nil {
		return absent
	}
	d := overview.EngineCheck{Hypervisor: r.d.Hypervisor.Name(), Installed: r.d.Hypervisor.Installed()}
	if d.Installed {
		d.Version, d.Err = r.d.Engine.Ping(ctx)
		d.Agent = d.Err != nil && r.d.ThisMac.Agent != nil && r.d.ThisMac.Agent()
	}
	return overview.EngineRow(d, r.d.ThisMac.Version)
}
