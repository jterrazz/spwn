package wire

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/devices"
	devicesjsonfile "github.com/jterrazz/studio-station/apps/engine/devices/adapter/jsonfile"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/remote/api"
	"github.com/jterrazz/studio-station/apps/engine/remote/serve"
)

type piped struct {
	io.Reader
	io.Writer
	closers []io.Closer
}

func (p piped) Close() error {
	for _, c := range p.closers {
		_ = c.Close()
	}
	return nil
}

// value: the peer of an engine whose hello says the book it holds; nil is
// an engine that does not answer.
func peerOf(t *testing.T, held *devices.Head) devicesjsonfile.Peer {
	if held == nil {
		return enginePeer{engineAPI{c: api.NewClient(func(context.Context) (io.ReadWriteCloser, error) { return nil, errors.New("no socket") })}}
	}
	hello := call.Bind(call.Call[struct{}, struct{}, engine.Hello]{
		Name: engine.CallHello, Where: call.InEngine, Applies: coremachine.Anywhere,
		Run: func(context.Context, struct{}, struct{}, call.Progress) (engine.Hello, error) {
			return engine.Hello{Book: held.String()}, nil
		},
	}, struct{}{})
	s := serve.Server{Calls: func(name string) (call.Handler, bool) { return hello, name == engine.CallHello }}
	return enginePeer{engineAPI{c: api.NewClient(func(context.Context) (io.ReadWriteCloser, error) {
		inR, inW := io.Pipe()
		outR, outW := io.Pipe()
		go s.Conn(t.Context(), piped{Reader: inR, Writer: outW, closers: []io.Closer{inR, outW}})
		return piped{Reader: outR, Writer: inW, closers: []io.Closer{outR, inW}}, nil
	})}}
}

func TestTheHeadAProcessTakesIsTheOneTheRunningEngineHolds(t *testing.T) {
	if got, ok, err := peerOf(t, nil).Held(t.Context()); err != nil || ok || got != (devices.Head{}) {
		t.Fatalf("no engine answering: %+v, %v, %v; want nothing held and no error", got, ok, err)
	}
	head := devices.Book{Generation: 5, Devices: []devices.Device{{Name: "mini", Key: "ssh-ed25519 AAAAmini"}}}.Head()
	for _, want := range []devices.Head{{}, head} {
		if got, ok, err := peerOf(t, &want).Held(t.Context()); err != nil || !ok || got != want {
			t.Fatalf("an engine holding %+v: %+v, %v, %v; want it held", want, got, ok, err)
		}
	}
}
