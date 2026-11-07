package calls

import (
	"context"

	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/machines"
)

type Deps struct {
	Machines Machines
	// value: this Mac's images, a missing one built before a create the
	// owner confirmed.
	Images ImageBuilds
	Kind   machine.Kind
	// value: this Mac reports a battery: a server made here stops whenever
	// it sleeps.
	Laptop bool
	// value: what starts an agent in a workspace just made.
	Agents AgentStarter
}

type AgentStarter interface {
	// value: starts an agent in vm's terminal session once it answers, task
	// typed as its first prompt.
	StartAgent(ctx context.Context, vm, task string) error
}

type ImageBuilds interface {
	// value: builds image on this Mac, each step's label to step as it starts.
	Build(ctx context.Context, image string, step func(label string)) error
}

type Machines interface {
	// value: sofar sees the fleet after each Mac's answer, the return holds every Mac.
	AllMachinesAsItFills(ctx context.Context, thisMacOnly bool, sofar func(machines.AllMachines)) (machines.AllMachines, error)
	// value: rewrites the owner's ssh blocks for the fleet just read.
	SSHBlocks(ctx context.Context, f machines.AllMachines) error
	// value: keeps when each other Mac answered, and its VMs then.
	Remember(ctx context.Context, f machines.AllMachines) error
	CheckCreate(ctx context.Context, r machines.CreateRequest) (machines.CreateRequest, error)
	Create(ctx context.Context, r machines.CreateRequest, progress func(machines.Step)) (machines.Created, error)
	// value: refuses every action while this Mac's engine is stopped.
	EngineUp(ctx context.Context) error
	CheckStart(name string) error
	Start(ctx context.Context, name string) error
	CheckStop(name string) error
	// value: the lifetime of a VM of this Mac, which says its kind; empty
	// for no VM of it.
	LifetimeOf(name string) machines.Lifetime
	// value: whether the VM of this Mac runs now.
	Running(ctx context.Context, name string) (bool, error)
	Stop(ctx context.Context, name string) error
	// value: refuses a name that is no VM of this host, worded for verb.
	CheckVM(name, verb string) error
	Delete(ctx context.Context, name string) (machines.DeleteReport, error)
	// value: whether name is a VM rather than a Mac.
	CheckRestart(name string) (vm bool, err error)
	Restart(ctx context.Context, name string) (machines.RestartReport, error)
	CheckMac() error
	// value: the registered name of another Mac, which name may say short;
	// refuses a name that is no other Mac, worded for verb.
	CheckRemoteMac(name, verb string) (string, error)
	AddMac(ctx context.Context, name string, h machines.MacRecord) error
	// value: what a Mac's station says once ssh reaches it; a Mac that does
	// not answer, or has no station, is the error.
	Reach(ctx context.Context, name string) (machines.Added, error)
	ForgetMac(ctx context.Context, name string) error
	// value: refuses off a terminal, where nobody can type the password.
	Unlock(ctx context.Context, name string, interactive bool) error
	// value: whether the Mac's station answers now.
	Answers(ctx context.Context, name string) bool
	// value: waits, in the budget of a Mac's boot, for its station to answer.
	AwaitAnswer(ctx context.Context, name string) bool
	// value: the VMs of that Mac, as it lists them now.
	VMsOf(ctx context.Context, name string) ([]machines.VM, error)
	// value: starts a VM of that Mac through its station; one that runs
	// already is left as it is.
	StartOn(ctx context.Context, mac, vm string) error
	// value: refuses a name that is no known machine.
	CheckRemote(name string) error
	SSH(ctx context.Context, name string) (machines.SSHTarget, error)
	CheckEdit(name string, c machines.Change) error
	Edit(ctx context.Context, name string, c machines.Change) (machines.Edited, error)
}

func All(d Deps) []call.Handler {
	return []call.Handler{
		call.BindWatch(List, d), call.Bind(CreateWorkspace, d), call.Bind(CreateServer, d), call.Bind(Start, d), call.Bind(Stop, d), call.Bind(Delete, d),
		call.Bind(Restart, d), call.Bind(RestartMac, d), call.Bind(Add, d), call.Bind(Forget, d), call.Bind(Unlock, d), call.Bind(SSH, d), call.Bind(Edit, d),
	}
}
