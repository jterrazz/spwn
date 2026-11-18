package contract

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/agents"
)

type Session interface {
	// value: the multiplexer of a VM the agents run in, each in a
	// terminal of its own the owner attaches to.
	//
	// invariant: Open runs argv in folder in a new terminal of the multiplexer,
	// marked as app's agent and named by its name, and answers its address
	// once the program is ready for its first prompt.
	Open(ctx context.Context, app agents.App, name, folder string, argv []string) (string, error)
	// invariant: Send types text into the terminal at that address, then
	// Enter; the text travels on stdin, never on a command line.
	Send(ctx context.Context, terminal, text string) error
	// invariant: Close ends the program of the terminal at that address and
	// takes the terminal away; one already gone changes nothing.
	Close(ctx context.Context, terminal string) error
	// value: the live terminals Open made, with what each agent says of
	// itself through the multiplexer.
	Terminals(ctx context.Context) ([]agents.Terminal, error)
}

type Multiplexer interface {
	// invariant: Attach puts the owner's terminal on the terminal at that
	// address of machine until they detach; an empty machine is this one.
	Attach(ctx context.Context, machine, terminal string) error
}
