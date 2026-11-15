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
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/words"
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

type Aimed struct {
	// value: where a call about the target machine is sent, the VM it names
	// in its input, the machine's name and its kind.
	At       machine.Ref
	VM, Name string
	Kind     machine.Kind
}

func Aim(ctx context.Context, env Env, self string) Aimed {
	// invariant: a VM's accounts are the business of the engine of its own
	// Mac, which knows what it granted and holds a sign-in's exchange; a
	// Mac's are its own.
	t := env.Target()
	name := t.Name
	if name == "" {
		name = self
	}
	if h, ok := Holder(ctx, env, name); ok {
		return Aimed{At: machine.Ref{Name: h.Mac}, VM: name, Name: name, Kind: h.Kind}
	}
	return Aimed{At: t, Name: name, Kind: machine.Computer}
}

func NeedsTerminal(subject string) error {
	return fail.New(fail.NeedsTerminal, words.NeedsTerminal, subject, nil)
}
