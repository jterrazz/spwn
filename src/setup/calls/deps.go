package calls

import (
	"context"

	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/setup"
)

type Deps struct {
	Setup Setup
	// value: the owner's terminal, to name the machine when no name is
	// given; nil where nobody is there to answer.
	Ask Ask
}

type Setup interface {
	CheckName(name string) error
	Init(ctx context.Context, name string, interactive bool, ask func() (string, error)) (setup.Report, error)
}

type Ask interface {
	Name(ctx context.Context) (string, error)
}

func All(d Deps) []call.Handler {
	return []call.Handler{call.Bind(Init, d)}
}
