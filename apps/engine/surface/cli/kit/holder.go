package kit

import (
	"context"
	"errors"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

type Held struct {
	// value: a VM of All machines, the Mac that holds it, and whether it runs.
	VM, Mac      string
	MacIsThisMac bool
	// value: whether that Mac answered the list.
	Reachable bool
	Running   bool
	Kind      machine.Kind
}

func Holder(ctx context.Context, env Env, vm string) (Held, bool) {
	list, err := call.Read(ctx, env.Calls, env.Self, machinescalls.List, machinescalls.ListIn{Unwritten: true}, nil)
	if err != nil {
		return Held{}, false
	}
	for _, h := range list.All.Macs {
		for _, v := range h.VMs {
			if v.Name == vm {
				return Held{VM: vm, Mac: h.Name, MacIsThisMac: h.ThisMac, Reachable: h.ThisMac || h.Reachable, Running: v.State == machines.Running, Kind: v.Lifetime.Kind()}, true
			}
		}
	}
	return Held{}, false
}

func (h Held) OnTarget(env Env) bool {
	// value: whether the Mac the command runs on is the one that holds the
	// VM, named in full or as the interface names it (mac-mini).
	t := env.Target()
	if t.Name == "" || t.Name == env.Self.Name {
		return h.MacIsThisMac
	}
	return t.Name == h.Mac || t.Name == page.MacName(h.Mac, env.Self.Name)
}

// value: SPEC §8.1: when the Mac that does not answer last answered, as All
// machines says it (`last answer 03:12`); nothing for a Mac never heard, or
// for a VM of it, which is named by where it is.
func lastAnswer(ctx context.Context, env Env, err error) []string {
	var e *fail.Error
	if !errors.As(err, &e) || e.Code != call.Unreachable {
		return nil
	}
	list, lErr := call.Read(ctx, env.Calls, env.Self, machinescalls.List, machinescalls.ListIn{Unwritten: true}, nil)
	if lErr != nil {
		return nil
	}
	for _, h := range list.All.Macs {
		if h.Name == e.Subject && !h.Reachable && !h.Answered.IsZero() {
			return []string{"last answer " + page.LastAnswer(h.Answered, env.Now())}
		}
	}
	return nil
}

func OnItsMac(ctx context.Context, env Env, vm string, err error) error {
	// value: SPEC §8.1: an action on a VM sent to its Mac, which does not
	// answer, names the VM where it is (`infra is on mac-mini, which does not
	// answer`); any other error is kept.
	var e *fail.Error
	if err == nil || !errors.As(err, &e) || e.Code != call.Unreachable {
		return err
	}
	h, ok := Holder(ctx, env, vm)
	if !ok || h.MacIsThisMac || h.Mac != e.Subject {
		return err
	}
	detail := map[string]string{"mac": h.Mac}
	for k, v := range e.Detail {
		detail[k] = v
	}
	return fail.New(e.Type, call.VMMacUnreachable, vm, detail)
}
