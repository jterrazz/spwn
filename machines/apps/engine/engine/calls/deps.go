package calls

import (
	"context"
	"encoding/json"
	"sync"
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
	// value: held from the read of what the engine kept to the write of its
	// next asks, one for the Mac, shared by every ask of it.
	Asking *sync.Mutex
	// value: the revision of station.page/v1 the engine's pages speak.
	Page int
	// value: engine.state asked of this Mac's engine, for a door that is
	// not the engine itself.
	Forward func(ctx context.Context, in engine.StateIn) (<-chan json.RawMessage, error)
	// value: the views engine.state projects of the store beside its
	// slices; nil projects none.
	Views Views
}

type Installer interface {
	Install(ctx context.Context) (engine.Installed, error)
}

type Control interface {
	// value: refuses a verb off a host, or before the engine's job is installed.
	Check() error
	Read(ctx context.Context) (engine.Reading, error)
	Plan(ctx context.Context) (engine.Work, error)
	Log(ctx context.Context) (engine.Log, error)
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
	return []call.Handler{call.Bind(Read, d), call.Bind(Start, d), call.Bind(Stop, d), call.Bind(ReadRelease, d), call.Bind(RefreshRelease, d), call.Bind(ReadLog, d), call.BindWatch(StateWatch, d)}
}
