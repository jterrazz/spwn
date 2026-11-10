package calls

import (
	"context"
	"errors"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/confirm"
	"github.com/jterrazz/studio-station/src/core/fail"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/machines"
)

type ListIn struct {
	// value: this host only, without asking the others.
	ThisMac bool
	// value: the fleet read only, the owner's client blocks left as they are.
	Unwritten bool
}

type ListOut struct {
	All machines.AllMachines
	// value: why the owner's client blocks for the fleet were not rewritten.
	Unrefreshed error
}

var List = call.Watch[Deps, ListIn, ListOut]{
	// invariant: each update is the fleet so far, a row more as each Mac
	// answers; the last holds every Mac, and the ssh blocks are rewritten
	// for it alone.
	Name: "machines.list", Where: call.InProcess, Applies: machine.Anywhere,
	Run: func(ctx context.Context, d Deps, in ListIn, _ call.Progress) (<-chan call.Update[ListOut], error) {
		updates := make(chan call.Update[ListOut])
		send := func(u call.Update[ListOut]) {
			select {
			case updates <- u:
			case <-ctx.Done():
			}
		}
		go func() {
			defer close(updates)
			allMachines, err := d.Machines.AllMachinesAsItFills(ctx, in.ThisMac, func(f machines.AllMachines) { send(call.Update[ListOut]{Value: ListOut{All: f}}) })
			if err != nil {
				send(call.Update[ListOut]{Err: err})
				return
			}
			out := ListOut{All: allMachines}
			if !in.ThisMac && !in.Unwritten {
				out.Unrefreshed = errors.Join(d.Machines.SSHBlocks(ctx, allMachines), d.Machines.Remember(ctx, allMachines))
			}
			send(call.Update[ListOut]{Value: out})
		}()
		return updates, nil
	},
}

type CreateIn struct {
	Request machines.CreateRequest
	// value: the answer to "build the missing image first?", asked only when
	// the image is not built on this Mac.
	Confirm string
}

var CreateWorkspace = call.Call[Deps, CreateIn, machines.Created]{
	// invariant: a workspace, throwaway or permanent; always on is a server's.
	Name: "machines.create-workspace", Where: call.InEngine, Applies: machines.CanCreate, Needs: machines.VMNeeds(), Confirm: confirm.BuildImage,
	Errors: []fail.Code{confirm.YesNeeded},
	Run: func(ctx context.Context, d Deps, in CreateIn, p call.Progress) (machines.Created, error) {
		r := in.Request
		r.Server = false
		if image := missingImage(ctx, d, r); image != "" {
			if err := confirm.Ask(confirm.BuildImage, d.Kind, image, in.Confirm); err != nil {
				return machines.Created{}, err
			}
			if err := build(ctx, d, image, p); err != nil {
				return machines.Created{}, err
			}
		}
		return create(ctx, d, r, p)
	},
}

type CreateServerIn struct {
	Name, Image, Network string
	Size                 machines.Resources
	// value: its health check, a URL or host:port; empty for none.
	Check string
	// value: the answer to "build the missing image first?", asked only when
	// the image is not built on this Mac.
	Confirm string
}

var CreateServer = call.Call[Deps, CreateServerIn, machines.Created]{
	// invariant: a server is always on, behind its Mac, and named; station
	// sets up nothing inside it (SPEC §7.1).
	Name: "machines.create-server", Where: call.InEngine, Applies: machines.CanCreate, Needs: machines.VMNeeds(), Confirm: confirm.BuildImage,
	Errors: []fail.Code{confirm.YesNeeded},
	Run: func(ctx context.Context, d Deps, in CreateServerIn, p call.Progress) (machines.Created, error) {
		r := machines.CreateRequest{Image: in.Image, Name: in.Name, Server: true, Network: in.Network, Size: in.Size, Check: in.Check}
		if image := missingImage(ctx, d, r); image != "" {
			if err := confirm.Ask(confirm.BuildImage, d.Kind, image, in.Confirm); err != nil {
				return machines.Created{}, err
			}
			if err := build(ctx, d, image, p); err != nil {
				return machines.Created{}, err
			}
		}
		created, err := create(ctx, d, r, p)
		created.OnALaptop = err == nil && d.Laptop
		return created, err
	},
}

// value: the image a create would be refused for alone, not built on this
// Mac; empty when it is built, when station only fetches it, or when another
// refusal comes first.
func missingImage(ctx context.Context, d Deps, r machines.CreateRequest) string {
	_, err := d.Machines.CheckCreate(ctx, r)
	if unbuilt := new(machines.NoImageError); errors.As(err, &unbuilt) && d.Images != nil {
		if catalog.FetchedOnly(unbuilt.Image) {
			return ""
		}
		return unbuilt.Image
	}
	return ""
}

func build(ctx context.Context, d Deps, image string, p call.Progress) error {
	last := ""
	err := d.Images.Build(ctx, image, func(label string) {
		if last != "" {
			p.Done(last, nil)
		}
		last = label
		p.Step(label, "")
	})
	if last != "" {
		p.Done(last, err)
	}
	return err
}

func create(ctx context.Context, d Deps, r machines.CreateRequest, p call.Progress) (machines.Created, error) {
	last := ""
	created, err := d.Machines.Create(ctx, r, func(s machines.Step) {
		if last != "" {
			p.Done(last, nil)
		}
		last = s.Label
		p.Step(s.Label, "")
	})
	if last != "" {
		p.Done(last, err)
	}
	return created, err
}
