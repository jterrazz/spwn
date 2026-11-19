package calls

import (
	"context"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/engine/contract"
)

type Deps struct {
	Control Control
	// value: writes this Mac's engine job and brings the engine to this
	// station; nil where no job can be written.
	Installer Installer
	Kind      machine.Kind
	// value: station's releases on GitHub and the engine's last asks of
	// them; the version of the engine that answers, and this Mac's arch,
	// the binary a release carries for it.
	Releases      contract.Releases
	Cache         contract.ReleaseCache
	Version, Arch string
	Now           func() time.Time
}

type Installer interface {
	Install(ctx context.Context) (engine.Installed, error)
}

type Control interface {
	// value: refuses a verb off a host, or before the engine's job is installed.
	Check() error
	Read(ctx context.Context) (engine.Reading, error)
	Plan(ctx context.Context) (engine.Work, error)
	Stop(ctx context.Context, w engine.Work, by string) (engine.StopReport, error)
	Start(ctx context.Context) (engine.StartReport, error)
}

const (
	StartFailed        fail.Code = "engine.start-failed"
	ReleaseUnreachable fail.Code = "engine.release-unreachable"
)

var Applies machine.Applies = func(p machine.Profile) machine.Verdict {
	// value: the engine's calls apply where an engine runs: a Mac.
	if !p.Caps.Engine {
		return machine.No(p.Ref.Name + " runs no engine: only a Mac does")
	}
	return machine.Yes
}

func All(d Deps) []call.Handler {
	return []call.Handler{call.Bind(Read, d), call.Bind(Start, d), call.Bind(Stop, d), call.Bind(ReadRelease, d), call.Bind(RefreshRelease, d)}
}
