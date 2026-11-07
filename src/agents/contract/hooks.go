package contract

import (
	"context"

	"github.com/jterrazz/studio-station/src/agents"
)

type Hooks interface {
	// value: a strategy, one adapter per app: the hook that tells station
	// what the app's agent does, in the app's own settings file inside the VM.
	App() agents.App
	// invariant: the program that starts the app's agent in a folder, its
	// first prompt typed after it starts, never on its command line.
	Launch() []string
	// invariant: installing adds station's hook and keeps every other key of
	// the app's settings file as the owner left it; a second install changes
	// nothing.
	Install(ctx context.Context) error
	// invariant: the hook is installed when the app's settings file runs it on
	// every event station reads.
	Inspect(ctx context.Context) (installed bool, err error)
	// invariant: removing takes station's hook out and nothing else; removing
	// a hook not installed changes nothing.
	Remove(ctx context.Context) error
	// invariant: a payload the app's hook hands over reads as its event; a
	// payload of an event or a version the adapter does not know is ignored,
	// never an error, as a hook never fails the agent.
	Event(ctx context.Context, payload []byte) (agents.Event, bool)
}
