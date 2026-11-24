package machines

import (
	"cmp"
	"slices"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
)

type Counts struct {
	Machines, Running, Stopped int
}

func MaySleep(f AllMachines, h MacView) string {
	// value: what a server made on a laptop is warned of.
	return Label(f, h) + " may sleep: a server pauses with it."
}

func Label(f AllMachines, h MacView) string {
	if h.ThisMac {
		return "This Mac"
	}
	owner := f.owner()
	if short, ok := strings.CutPrefix(h.Name, owner+"-"); ok && owner != "" && short != "" {
		return short
	}
	return h.Name
}

// value: the user every Mac's name starts with: This Mac's, or the one all
// the Macs share when the fleet was read on another Mac (SPEC §2.5).
func (f AllMachines) owner() string {
	users := map[string]bool{}
	for _, h := range f.Macs {
		if h.ThisMac {
			return h.User
		}
		users[h.User] = true
	}
	if len(users) != 1 {
		return ""
	}
	for u := range users {
		return u
	}
	return ""
}

func Elsewhere(f AllMachines) AllMachines {
	// invariant: a fleet another Mac's engine read, as this binary shows it:
	// none of its Macs is This Mac (SPEC §2.5, §8.1: rendered here).
	f.Macs = slices.Clone(f.Macs)
	for i := range f.Macs {
		f.Macs[i].ThisMac = false
	}
	return f
}

func Tally(f AllMachines) Counts {
	var c Counts
	for _, h := range f.Macs {
		c.Machines++
		if h.Reachable {
			c.Running++
		}
		for _, vm := range h.VMs {
			c.Machines++
			switch vm.State {
			case Running:
				c.Running++
			case Stopped, Suspended:
				c.Stopped++
			case Stale, Broken, UnknownState, Creating:
			}
		}
	}
	return c
}

func Ordered(vms []VM) []VM {
	// invariant: a Mac's servers, then its workspaces, each by name.
	out := slices.Clone(vms)
	slices.SortStableFunc(out, func(a, b VM) int {
		return cmp.Or(cmp.Compare(rank(a.Lifetime.Kind()), rank(b.Lifetime.Kind())), cmp.Compare(a.Name, b.Name))
	})
	return out
}

func rank(k machine.Kind) int {
	if k == machine.Server {
		return 0
	}
	return 1
}

func KindWord(k machine.Kind) string {
	if k == machine.Server {
		return "Server"
	}
	return "Workspace"
}

func Behind(h MacView, thisMac string) (fix string, differs bool) {
	if h.ThisMac || h.StationVersion == "" || thisMac == "" {
		return "", false
	}
	switch SkewOf(thisMac, h.StationVersion) {
	case Speaks:
		return "", false
	case HereBehind:
		return "update station here (" + catalog.InstallLine + ")", true
	case FarBehind:
	}
	return "update station there (" + catalog.InstallLine + ")", true
}

func Findings(f AllMachines) []finding.Finding {
	thisMac := ""
	for _, h := range f.Macs {
		if h.ThisMac {
			thisMac = h.StationVersion
		}
	}
	var out []finding.Finding
	add := func(m machine.Ref, sev finding.Severity, subject, text, fix string) {
		out = append(out, finding.Finding{Machine: m, Section: section.Overview, Severity: sev, Subject: subject, Text: text, Fix: finding.Fix{Command: fix}})
	}
	for _, h := range f.Macs {
		label, mac := Label(f, h), machine.Ref{Name: h.Name}
		on := ""
		if !h.ThisMac {
			on = " --on " + label
		}
		switch s := MacStandingOf(h); s {
		case MacAsleep, MacNotSeen:
			switch {
			case !macsServer(h) && s == MacAsleep:
			case h.LAN != "" && macsServer(h):
				add(mac, severityOf(s.Tone()), label, label+" does not answer", "station machines unlock "+label)
			default:
				add(mac, severityOf(s.Tone()), label, label+" does not answer", "")
			}
		case MacEngineNotInstalled:
			add(mac, severityOf(s.Tone()), label, label+" has no engine installed", "station engine start"+on)
		case MacEngineStopped:
			add(mac, severityOf(s.Tone()), label, label+" has its engine stopped", "station engine start"+on)
		case MacOnline:
		}
		if h.Reachable && h.Engine.Running() && h.Engine.LocalNetwork == LocalNetworkNotApproved {
			out = append(out, finding.Finding{Machine: mac, Section: section.Overview, Severity: finding.NeedsALook, Subject: label, Text: NotApproved(h.Name, h.Engine.Bundled), Fix: finding.Fix{Path: LocalNetworkFix(h.Name, h.Engine.Bundled)}})
		}
		if h.Reachable && h.Engine.Running() && h.Engine.Agent.Unserved() {
			out = append(out, finding.Finding{Machine: mac, Section: section.Overview, Severity: finding.NeedsALook, Subject: label, Text: h.Engine.Agent.Said(h.Name), Fix: finding.Fix{Command: h.Engine.Agent.Fix()}})
		}
		if fix, ok := Behind(h, thisMac); ok && h.Reachable {
			out = append(out, finding.Finding{Machine: mac, Section: section.Overview, Severity: finding.NeedsALook, Subject: label, Text: label + " runs " + h.StationVersion, Fix: finding.Fix{Path: fix}})
		}
		for _, vm := range Ordered(h.VMs) {
			if f, ok := vmFinding(machine.Ref{Name: vm.Name, On: h.Name}, vm, on, h.Engine.Bundled); ok {
				out = append(out, f)
			}
		}
	}
	return finding.Rank(out)
}

// invariant: a Mac that hosts a server and stops answering may wait at its
// disk unlock after a power cut: its readiness fix is the unlock, from your
// LAN (SPEC §7.6).
func macsServer(h MacView) bool {
	return slices.ContainsFunc(h.VMs, func(vm VM) bool { return vm.Lifetime == AlwaysOn })
}

func Down(vm VM) bool {
	return StandingOf(vm) == StandingDown
}

func severityOf(t verdict.Tone) finding.Severity {
	// invariant: a finding's severity is its machine's standing's tone: bad
	// is broken, any other tone needs a look, so a finding never says broken
	// of a machine its standing does not.
	if t == verdict.Bad {
		return finding.Broken
	}
	return finding.NeedsALook
}

// invariant: what a VM's finding says is read from its facts, its severity
// from its standing (severityOf). A running VM its Mac's Local Network
// or its Mac's refused VM user blocks says so first, the cause of whatever
// else fails; a VM whose
// create failed, its start the cure, or its delete when no start can reach
// its guest; a server in a loop, broken, powered
// off from inside, or whose check fails since it passed, a VM
// ended unexpectedly or broken, a running one whose SSH fails, and any other
// VM down each have one; a check that has not answered yet is none. The
// engine never restarts a server by itself: the restart is the owner's
// gesture.
func vmFinding(ref machine.Ref, vm VM, on string, bundled bool) (finding.Finding, bool) {
	s := StandingOf(vm)
	f := finding.Finding{Machine: ref, Section: section.Overview, Severity: severityOf(s.Tone()), Subject: vm.Name}
	server := vm.Lifetime == AlwaysOn
	switch {
	case vm.State == Running && vm.Blocked == LocalNetworkBlock:
		f.Text, f.Fix = vm.Name+" is unreachable: "+NotApproved(ref.On, bundled), finding.Fix{Path: LocalNetworkFix(ref.On, bundled)}
	case vm.State == Running && vm.Blocked == VMUserBlock:
		f.Text, f.Fix = vm.Name+" is unreachable: "+VMUserRefused(ref.On), finding.Fix{Path: VMUserFix}
	case s == StandingNotCreated && vm.Birth.Gone:
		f.Text, f.Again = vm.Name+" was not created: it failed at "+vm.Birth.Step.Noun(), true
		f.Fix = finding.Fix{Command: "station machines delete " + vm.Name + on}
	case s == StandingNotCreated:
		f.Text, f.Again = vm.Name+" was not created: it failed at "+vm.Birth.Step.Noun(), true
		f.Fix = finding.Fix{Call: "machines.start", Args: map[string]string{"VM": vm.Name}, On: ref.On, Command: "station machines start " + vm.Name + on}
	case server && vm.Restarts.Looping:
		f.Text = vm.Name + " is in a loop: " + vm.Restarts.InALoop()
		f.Fix = finding.Fix{Call: "machines.restart", Args: map[string]string{"Name": vm.Name}, On: ref.On, Confirm: vm.Name}
	case server && vm.State == Stopped && vm.Detail == PoweredOffInside:
		f.Text, f.Fix = vm.Name+" "+PoweredOffInside, finding.Fix{Command: "station machines start " + vm.Name + on}
	case server && vm.State == Running && vm.CheckStatus.State == CheckFailing:
		f.Text = vm.Name + " check failing since " + vm.CheckStatus.Since.Local().Format("15:04")
		f.Fix = finding.Fix{Call: "machines.restart", Args: map[string]string{"Name": vm.Name}, On: ref.On, Confirm: vm.Name}
		f.Detail = downDetail(ref.On, vm)
	case vm.State == Stale:
		f.Text, f.Fix = vm.Name+" ended unexpectedly", finding.Fix{Command: "station machines start " + vm.Name + on}
	case server && vm.State == Broken:
		f.Text = vm.Name + " is broken: " + vm.Detail
		f.Fix = finding.Fix{Call: "machines.restart", Args: map[string]string{"Name": vm.Name}, On: ref.On, Confirm: vm.Name}
	case vm.State == Broken:
		f.Text, f.Fix = vm.Name+" "+vm.Detail+", stopped", finding.Fix{Command: "station machines delete " + vm.Name + on}
	case vm.State == Running && !HealthOf(vm).SSH && vm.Guest == "absent":
		f.Text, f.Fix = vm.Name+" fails "+StepSSH, finding.Fix{Command: "station machines restart " + vm.Name + on}
	case s == StandingDown:
		f.Text, f.Fix = vm.Name+" is down", finding.Fix{Command: "station machines start " + vm.Name + on}
	default:
		return finding.Finding{}, false
	}
	return f, true
}

func downDetail(mac string, vm VM) string {
	// invariant: a server whose check fails names its Mac and when the check
	// began to fail, and says SSH answers only when its connection opened.
	parts := []string{mac, "Since " + vm.CheckStatus.Since.Local().Format("15:04")}
	if vm.SSH {
		parts = append(parts, "SSH still answers")
	}
	return strings.Join(parts, ". ") + "."
}

func Newer(a, b string) bool {
	// value: whether release a is past release b, both vMAJOR.MINOR.PATCH
	// with the pre-release words semver allows; an unreadable one is never
	// newer.
	pa, pb := release(a), release(b)
	if pa == nil || pb == nil {
		return false
	}
	if c := slices.Compare(pa.core, pb.core); c != 0 {
		return c > 0
	}
	return preceded(pb.pre, pa.pre)
}

func Released(v string) bool {
	// value: whether v reads as a release Newer orders.
	return release(v) != nil
}

type semver struct {
	core []int
	pre  []string
}

func release(v string) *semver {
	// invariant: a station version is vMAJOR.MINOR.PATCH with the pre-release
	// words semver allows after a dash (v4.0.0-alpha.1); a build suffix or any
	// other form is no version.
	rest, ok := strings.CutPrefix(v, "v")
	if !ok {
		return nil
	}
	core, pre, dashed := strings.Cut(rest, "-")
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return nil
	}
	out := &semver{}
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil
		}
		out.core = append(out.core, n)
	}
	if dashed {
		if pre == "" {
			return nil
		}
		out.pre = strings.Split(pre, ".")
	}
	return out
}

// value: whether the pre-release words a come before b under semver: a
// release has none and comes after every pre-release of its numbers; numeric
// words compare as numbers and sort before the others; a shorter run of equal
// words comes first.
func preceded(a, b []string) bool {
	switch {
	case len(a) == 0:
		return false
	case len(b) == 0:
		return true
	}
	for i := range min(len(a), len(b)) {
		x, xerr := strconv.Atoi(a[i])
		y, yerr := strconv.Atoi(b[i])
		var c int
		switch {
		case xerr == nil && yerr == nil:
			c = cmp.Compare(x, y)
		case xerr == nil:
			c = -1
		case yerr == nil:
			c = 1
		default:
			c = strings.Compare(a[i], b[i])
		}
		if c != 0 {
			return c < 0
		}
	}
	return len(a) < len(b)
}

type Need struct {
	Broken             bool
	Subject, Text, Fix string
}

func Needs(f AllMachines) []Need {
	fs := Findings(f)
	out := make([]Need, 0, len(fs))
	for _, x := range fs {
		out = append(out, Need{Broken: x.Severity == finding.Broken, Subject: x.Subject, Text: x.Text, Fix: x.Fix.Command + x.Fix.Path})
	}
	return out
}
