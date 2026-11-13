package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
	machinefilecalls "github.com/jterrazz/studio-station/apps/engine/machinefile/calls"
	"github.com/jterrazz/studio-station/apps/engine/machines"
)

type NewIn struct {
	// value: the machine file the new VM follows; empty for station-linux.
	From string
	// value: the Mac the sheet is opened from; empty for none.
	On string
}

type NewMachine struct {
	// value: what a create from From makes asked nothing else: the file,
	// the name it takes over the fleet as read, its size and lifetime.
	From     string
	Name     string
	Size     machines.Resources
	Lifetime machines.Lifetime
	// value: the lifetimes a workspace may be given, in the sheet's order.
	Lifetimes []machines.Lifetime
	// value: the least memory a create takes, machines.MinMemoryBytes,
	// and the most characters its purpose holds.
	MinMemory  int64
	PurposeMax int
	// value: the Mac the sheet opens on: the one asked for when it takes a
	// VM, else the roomiest, else This Mac (machines.Preselected).
	Mac string
	// value: every Mac of the fleet, in machines.list's order.
	On []NewOn
	// value: the server the sheet offers; nil when From is a user's file,
	// which no server follows.
	Server *NewServer
}

type NewOn struct {
	Mac     string
	ThisMac bool
	// value: its room, its presets judged; nil for a Mac silent, asleep or
	// that never said its memory.
	Room *machines.Room
	// value: the most memory a create takes there, its free memory; 0
	// without a room.
	MaxMemory int64
	// invariant: a create may be sent there now (machines.Takes); Verdict
	// says why not.
	Ready   bool
	Verdict verdict.Verdict
	// value: a server of NewServer's size there; nil without a room or a
	// server.
	Server *NewServerOn
}

type NewServer struct {
	// value: the built-in image a server is made of, and its size.
	Image string
	Size  machines.Resources
}

type NewServerOn struct {
	Fit machines.Fit
	// value: a laptop's warning that a server sleeps with it; plain and
	// empty elsewhere.
	Verdict verdict.Verdict
}

var NewSheet = call.Watch[Deps, NewIn, NewMachine]{
	// invariant: each update is the sheet over the fleet so far, a Mac more
	// as each answers; a machine file no workspace is made from is refused
	// before the first.
	Name: "machines.new", Where: call.InProcess, Applies: machine.Anywhere,
	Errors: []fail.Code{ServerTemplate, machinefilecalls.NotFound, machinefilecalls.NotATemplate},
	Run: func(ctx context.Context, d Deps, in NewIn, _ call.Progress) (<-chan call.Update[NewMachine], error) {
		unasked, err := d.Machines.Unasked(ctx, in.From)
		if err != nil {
			return nil, err
		}
		updates := make(chan call.Update[NewMachine])
		send := func(u call.Update[NewMachine]) {
			select {
			case updates <- u:
			case <-ctx.Done():
			}
		}
		sheet := func(f machines.AllMachines) call.Update[NewMachine] {
			name, err := d.Machines.NameFor(ctx, unasked.Kind, f)
			if err != nil {
				return call.Update[NewMachine]{Err: err}
			}
			return call.Update[NewMachine]{Value: newMachine(unasked, name, in.On, f)}
		}
		go func() {
			defer close(updates)
			all, err := d.Machines.AllMachinesAsItFills(ctx, false, func(f machines.AllMachines) { send(sheet(f)) })
			if err != nil {
				send(call.Update[NewMachine]{Err: err})
				return
			}
			send(sheet(all))
		}()
		return updates, nil
	},
}

func newMachine(u machines.Unasked, name, asked string, f machines.AllMachines) NewMachine {
	r := u.Request
	out := NewMachine{
		From: r.From, Name: name, Size: r.Size, Lifetime: r.Lifetime,
		Lifetimes: []machines.Lifetime{machines.Ephemeral, machines.Persistent},
		MinMemory: machines.MinMemoryBytes, PurposeMax: machines.PurposeMax,
		Mac: machines.Preselected(f, asked),
	}
	if u.Builtin != "" {
		out.Server = &NewServer{Image: u.Builtin, Size: machines.ServerSize()}
	}
	for _, h := range f.Macs {
		out.On = append(out.On, newOn(f, h, out.Server))
	}
	return out
}

func newOn(f machines.AllMachines, h machines.MacView, server *NewServer) NewOn {
	o := NewOn{Mac: h.Name, ThisMac: h.ThisMac, Verdict: verdict.Verdict{Tone: verdict.Plain}}
	label := machines.Label(f, h)
	c, known := machines.CapacityOf(h)
	switch {
	case h.Asleep:
		o.Verdict.Sentence = label + " is asleep"
		return o
	case !h.Reachable:
		o.Verdict.Sentence = label + " does not answer"
		return o
	case !known:
		o.Verdict.Sentence = label + " has not said its memory"
		return o
	}
	room := machines.RoomOf(h.Name, c)
	o.Room, o.MaxMemory, o.Ready = &room, c.Free, machines.Takes(h)
	if !o.Ready {
		o.Verdict = verdict.Verdict{Tone: verdict.Bad, Sentence: "No room on " + label + ". Choose the other one, or reclaim an idle machine first."}
	}
	if server != nil {
		o.Server = &NewServerOn{Fit: machines.FitOf(server.Size, c), Verdict: verdict.Verdict{Tone: verdict.Plain}}
		if h.Laptop {
			o.Server.Verdict = verdict.Verdict{Tone: verdict.Warn, Sentence: machines.MaySleep(f, h)}
		}
	}
	return o
}
