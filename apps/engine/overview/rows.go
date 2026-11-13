package overview

import (
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/ports"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

func FactRow(name string, f settings.Fact) Check {
	return Check{Name: name, State: f.State, Value: f.Value, Detail: f.Detail}
}

func GitValue(name string, f settings.Fact, expected string) Check {
	// invariant: a global git value that is set but is not the one the owner's
	// preferences expect is degraded, not ok: commits would carry the wrong author.
	switch {
	case f.State != catalog.StateOK || f.Value == "":
		return Check{Name: name, State: catalog.StateAbsent}
	case f.Value != expected:
		return Check{Name: name, State: catalog.StateDegraded, Value: f.Value, Detail: "expected " + expected}
	}
	return Check{Name: name, State: catalog.StateOK, Value: f.Value}
}

func AppUpdates(name string, behind int, missing, failed bool) Check {
	switch {
	case missing:
		return Check{Name: name, State: catalog.StateAbsent, Value: "not installed"}
	case failed:
		return Check{Name: name, State: catalog.StateUnknown}
	case behind > 0:
		return Check{Name: name, State: catalog.StateDegraded, Value: strconv.Itoa(behind) + " outdated", Detail: "station apps update --manager " + name}
	}
	return Check{Name: name, State: catalog.StateOK, Value: "all current"}
}

func Listening(ls []ports.Listener) []Port {
	// invariant: one row per port and process, the system's own listeners left
	// out, in port order.
	out := []Port{}
	seen := map[string]bool{}
	for _, l := range ls {
		if l.Process == "" || slices.Contains(catalog.SystemListeners, l.Process) {
			continue
		}
		port := strconv.Itoa(l.Port)
		if key := port + "/" + l.Process; !seen[key] {
			seen[key] = true
			out = append(out, Port{PID: l.PID, Name: l.Process, Port: port})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return atoi(out[i].Port) < atoi(out[j].Port) })
	return out
}

func atoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

type EngineCheck struct {
	// value: the hypervisor station's daemon runs VMs with, and whether it is installed.
	Hypervisor string
	Installed  bool
	// value: whether the daemon's job definition is on disk.
	Agent bool
	// value: the version the daemon answered, or the error it did not answer with.
	Version string
	Err     error
}

func EngineRow(d EngineCheck, station string) Check {
	// invariant: the daemon matters where VMs can exist: a host with its hypervisor; one that
	// does not answer, or answers another version, is degraded, never an error.
	c := Check{Name: "engine", State: catalog.StateAbsent}
	if !d.Installed {
		c.Value = "not needed until " + d.Hypervisor + " is installed"
		return c
	}
	switch {
	case d.Err != nil && d.Agent:
		c.State, c.Value, c.Detail = catalog.StateDegraded, "stopped", "station engine start"
	case d.Err != nil:
		c.State, c.Value, c.Detail = catalog.StateDegraded, "not installed", "station engine start"
	case d.Version != station:
		c.State, c.Value, c.Detail = catalog.StateDegraded, "runs "+d.Version+", station is "+station, "station engine start"
	default:
		c.State, c.Value = catalog.StateOK, "running "+d.Version
	}
	return c
}
