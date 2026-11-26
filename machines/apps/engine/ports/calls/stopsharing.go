package calls

import (
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
	"github.com/jterrazz/studio-station/apps/engine/ports"
	"github.com/jterrazz/studio-station/apps/engine/ports/contract"
)

type StopSharingIn struct {
	// value: the VM whose port it is; empty inside a VM, which stops its own.
	VM   string
	Port string
	// value: the Mac the VM runs on, when the caller has read it; empty finds it.
	Mac string
}

type StopSharingOut struct {
	Reservation ports.Reservation
	// value: the Mac that released it when it is not this one.
	Mac string
}

var StopSharing = call.Call[Deps, StopSharingIn, StopSharingOut]{
	// invariant: a stop of sharing asks nothing (ADR-097: a share gives it
	// back), and releases the reserved port on the VM's Mac, whose engine
	// remembers the choice (Remember).
	Name: "ports.stop-sharing", Section: section.Ports, Where: call.InEngine, Applies: machine.Anywhere,
	Run: func(ctx context.Context, d Deps, in StopSharingIn, _ call.Progress) (StopSharingOut, error) {
		port, err := d.Ports.CheckPort(in.Port)
		if err != nil {
			return StopSharingOut{}, err
		}
		vm, inside := d.Ports.Placed()
		if !inside {
			if vm = in.VM; vm == "" {
				return StopSharingOut{}, fail.New(fail.Refused, ports.CodeNeedsVM, in.Port, nil)
			}
		}
		t := ports.Target{VM: vm, Port: port}
		mac := in.Mac
		if mac == "" {
			if mac, err = d.Ports.CheckRelease(ctx, t); err != nil {
				return StopSharingOut{}, err
			}
		}
		r, err := d.Ports.Release(ctx, t, mac)
		return StopSharingOut{Reservation: r, Mac: mac}, err
	},
}

func Remember(prefs contract.PortsPrefs, vm string, port int, stopped bool) error {
	// invariant: the one writer of the owner's stops of sharing, kept in the
	// ports group of the VM's Mac: stopped adds the port, a share takes it back.
	g, err := prefs.Get()
	if err != nil {
		return err
	}
	key := vm + ":" + strconv.Itoa(port)
	had := g != nil && slices.Contains(g.NotShared, key)
	if had == stopped {
		return nil
	}
	if g == nil {
		g = &ports.PortsGroup{}
	}
	if stopped {
		g.NotShared = append(g.NotShared, key)
	} else {
		g.NotShared = slices.DeleteFunc(g.NotShared, func(k string) bool { return k == key })
	}
	if len(g.NotShared) == 0 {
		g.NotShared = nil
	}
	return prefs.Set(g)
}

func ForgetStops(prefs contract.PortsPrefs, vm string) error {
	// invariant: a VM deleted leaves none of the owner's stops of sharing.
	return forgetStops(prefs, func(k string) bool { return strings.HasPrefix(k, vm+":") })
}

func ForgetStopsBut(prefs contract.PortsPrefs, vms []string) error {
	// invariant: a stop of sharing of a VM that does not exist is forgotten,
	// so a VM made again under its name starts with none of them.
	return forgetStops(prefs, func(k string) bool {
		at := strings.LastIndex(k, ":")
		return at < 0 || !slices.Contains(vms, k[:at])
	})
}

func forgetStops(prefs contract.PortsPrefs, gone func(key string) bool) error {
	g, err := prefs.Get()
	if err != nil || g == nil {
		return err
	}
	kept := slices.DeleteFunc(slices.Clone(g.NotShared), gone)
	if len(kept) == len(g.NotShared) {
		return nil
	}
	if len(kept) == 0 {
		kept = nil
	}
	g.NotShared = kept
	return prefs.Set(g)
}
