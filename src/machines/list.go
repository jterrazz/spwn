package machines

import (
	"cmp"
	"slices"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/src/core/finding"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/core/section"
)

type Counts struct {
	Machines, Running, Stopped int
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
	if h.ThisMac || h.StationVersion == "" || thisMac == "" || h.StationVersion == thisMac {
		return "", false
	}
	if newer(h.StationVersion, thisMac) {
		return "update station here (scripts/install.sh)", true
	}
	return "update station there (scripts/install.sh)", true
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
		switch {
		case Reach(h) == ReachAsleep && !macsServer(h):
		case !h.Reachable && h.LAN != "" && macsServer(h):
			add(mac, finding.Broken, label, label+" does not answer", "station machines unlock "+label)
		case !h.Reachable:
			add(mac, finding.Broken, label, label+" does not answer", "")
		case h.Engine.Known && !h.Engine.Running && !h.Engine.Installed && h.ThisMac:
			add(mac, finding.Broken, label, label+" has no engine installed", "station engine start")
		case h.Engine.Known && !h.Engine.Running:
			add(mac, finding.Broken, label, label+" has its engine stopped", "station engine start"+on)
		}
		if fix, ok := Behind(h, thisMac); ok && h.Reachable {
			out = append(out, finding.Finding{Machine: mac, Section: section.Overview, Severity: finding.NeedsALook, Subject: label, Text: label + " runs " + h.StationVersion, Fix: finding.Fix{Path: fix}})
		}
		for _, vm := range Ordered(h.VMs) {
			ref := machine.Ref{Name: vm.Name, On: h.Name}
			if f, ok := serverFinding(ref, vm, on); ok {
				out = append(out, f)
				continue
			}
			switch vm.State {
			case Stale:
				add(ref, finding.NeedsALook, vm.Name, vm.Name+" ended unexpectedly", "station machines start "+vm.Name+on)
			case Broken:
				add(ref, finding.Broken, vm.Name, vm.Name+" "+vm.Detail+", stopped", "station machines delete "+vm.Name+on)
			case Running:
				if h := HealthOf(vm); !h.SSH && vm.Guest == "absent" {
					add(ref, finding.Broken, vm.Name, vm.Name+" fails "+StepSSH, "station machines restart "+vm.Name+on)
				}
			case Stopped, Suspended, UnknownState, Creating:
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
	// value: a server is down when it is in a loop, powered off from inside,
	// or not running while its record wants it running (SPEC §10.1's red
	// badge).
	if vm.Lifetime != AlwaysOn {
		return false
	}
	switch vm.State {
	case Broken, Stale:
		return true
	case Stopped:
		return vm.Restarts.Looping || vm.Detail == PoweredOffInside || Wanted(vm) == "running"
	case Running:
		return vm.Restarts.Looping
	case Suspended, UnknownState, Creating:
	}
	return false
}

// invariant: a server in a loop or powered off from inside needs you (red);
// one whose check fails since it passed needs a look, never a restart.
func serverFinding(ref machine.Ref, vm VM, on string) (finding.Finding, bool) {
	if vm.Lifetime != AlwaysOn {
		return finding.Finding{}, false
	}
	f := finding.Finding{Machine: ref, Section: section.Overview, Subject: vm.Name}
	switch {
	case vm.Restarts.Looping:
		f.Severity, f.Text = finding.Broken, vm.Name+" is in a loop: "+vm.Restarts.InALoop()
	case vm.State == Stopped && vm.Detail == PoweredOffInside:
		f.Severity, f.Text, f.Fix = finding.Broken, vm.Name+" "+PoweredOffInside, finding.Fix{Command: "station machines start " + vm.Name + on}
	case vm.State == Running && vm.CheckStatus.State == CheckFailing:
		f.Severity, f.Text = finding.NeedsALook, vm.Name+" check failing since "+vm.CheckStatus.Since.Local().Format("15:04")
	default:
		return finding.Finding{}, false
	}
	return f, true
}

// value: whether release a is past release b, both vMAJOR.MINOR.PATCH with
// anything after the patch ignored; an unreadable one is never newer.
func newer(a, b string) bool {
	pa, pb := release(a), release(b)
	if pa == nil || pb == nil {
		return false
	}
	return slices.Compare(pa, pb) > 0
}

func release(v string) []int {
	core, ok := strings.CutPrefix(v, "v")
	if !ok {
		return nil
	}
	core, _, _ = strings.Cut(core, "-")
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return nil
	}
	out := make([]int, 0, 3)
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil
		}
		out = append(out, n)
	}
	return out
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
