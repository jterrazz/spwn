package wire

import (
	"context"
	"errors"
	"io"

	"github.com/jterrazz/studio-station/src/agents"
	"github.com/jterrazz/studio-station/src/awake"

	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/fail"
	coremachine "github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/engine"
	enginecalls "github.com/jterrazz/studio-station/src/engine/calls"
	"github.com/jterrazz/studio-station/src/engine/process"
	"github.com/jterrazz/studio-station/src/platform/local"
	"github.com/jterrazz/studio-station/src/remote/api"
	"github.com/jterrazz/studio-station/src/surface/cli/words"
)

// value: a client of the station.api/v2 server at file: this Mac's engine
// socket, or inside a VM the socket its engine offers there.
func socketAPI(h local.System, file string) engineAPI {
	return engineAPI{c: api.NewClient(func(ctx context.Context) (io.ReadWriteCloser, error) {
		return h.Sockets.DialUnix(ctx, file, api.ConnectBudget)
	})}
}

func (m machine) engineAPI() engineAPI { return socketAPI(m.sys, engine.SocketPath(m.paths.State)) }

type engineAPI struct{ c api.Client }

// invariant: an engine that does not answer is the engine's DownError,
// whichever call found it so.
func (e engineAPI) do(ctx context.Context, name string, in, out any) error {
	err := e.c.Call(ctx, name, coremachine.Ref{}, in, out, nil)
	if down := new(api.Down); errors.As(err, &down) {
		return &enginecalls.DownError{}
	}
	return err
}

func (e engineAPI) AgentsHeld(ctx context.Context) (agents.Held, error) {
	var h agents.Held
	err := e.do(ctx, engine.CallAgentsHeld, nil, &h)
	return h, err
}

func (e engineAPI) Hello(ctx context.Context) (engine.Hello, error) {
	var h engine.Hello
	err := e.do(ctx, engine.CallHello, nil, &h)
	return h, err
}

func (e engineAPI) VMs(ctx context.Context) ([]engine.VMView, error) {
	var views []engine.VMView
	err := e.do(ctx, engine.CallVMs, nil, &views)
	return views, err
}

func (e engineAPI) StopAll(ctx context.Context) (engine.StopAll, error) {
	var all engine.StopAll
	err := e.do(ctx, engine.CallStopAll, nil, &all)
	return all, err
}

func (e engineAPI) AwakeView(ctx context.Context) (awake.View, error) {
	var v awake.View
	err := e.do(ctx, engine.CallAwakeView, nil, &v)
	return v, err
}

func word(e *fail.Error) (message, hint string) {
	if w, ok := words.Codes()[e.Code]; ok {
		return w(e)
	}
	return "", ""
}

// invariant: the engine serves its own calls and this machine's registry,
// gated by this machine's profile and worded by this binary's table; a call
// for another machine rides far, the run's Router.
func (m machine) engineServer(d *process.Engine, registry *call.Registry, far call.Router) func(ctx context.Context, conn process.Stream, guest string) {
	serving := process.Serving{Registry: registry, ThisMachine: m.profile, Word: word, Sleep: m.sys.Clock.Sleep, Far: far}
	s := process.Server(d, serving)
	return func(ctx context.Context, conn process.Stream, guest string) {
		if guest == "" {
			s.Conn(ctx, conn)
			return
		}
		serving.Guest = guest
		process.Server(d, serving).Conn(ctx, conn)
	}
}
