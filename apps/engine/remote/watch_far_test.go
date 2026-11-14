package remote_test

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/remote"
	"github.com/jterrazz/studio-station/apps/engine/remote/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/remote/api"
	"github.com/jterrazz/studio-station/apps/engine/remote/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/remote/serve"
)

const fullMini = "jterrazz-mac-mini"

// value: a machine's activity that says whose it is by its CPU alone.
func sampled(cpu ...int) *call.Registry {
	registry, err := call.NewRegistry([]call.Handler{call.BindWatch(call.Watch[struct{}, struct{}, contracttest.Sample]{
		Name: "activity.read", Where: call.AtTarget, Applies: machine.Anywhere,
		Run: func(ctx context.Context, _ struct{}, _ struct{}, _ call.Progress) (<-chan call.Update[contracttest.Sample], error) {
			out := make(chan contracttest.Sample, len(cpu))
			for _, c := range cpu {
				out <- contracttest.Sample{CPU: c}
			}
			close(out)
			return call.Updates(ctx, out), nil
		},
	}, struct{}{})})
	if err != nil {
		panic(err)
	}
	return registry
}

// value: the laptop's engine socket: its own activity reads 99, the mini's
// (reached by its full name, asked by its short one) 12 then 40, and its VM
// linux-1's guest 7.
func laptopSocket(t *testing.T) (s serve.Server, macs, engine *fake.Dialer) {
	t.Helper()
	return laptopSocketOver(t, sampled(12, 40))
}

func laptopSocketOver(t *testing.T, minis *call.Registry) (s serve.Server, macs, engine *fake.Dialer) {
	t.Helper()
	here := sampled(99)
	macs, engine = fake.New(), fake.New()
	macs.Serve(fullMini, fake.Mac{Registry: minis, Profile: contracttest.Mini, Version: "v3.3.0"})
	engine.Serve(laptop.Ref.Name, fake.Mac{Registry: sampled(7), Profile: linux1, Version: "v3.3.0"})
	r := remote.NewRouter(here, laptop, remote.Reach{
		Macs: macs, Engine: engine, Version: "v3.3.0",
		Label: func(mac string) string { return strings.TrimPrefix(mac, "jterrazz-") },
		Locate: func(name string) (machine.Ref, bool) {
			switch name {
			case linux1.Ref.Name:
				return linux1.Ref, true
			case mini, fullMini:
				return machine.Ref{Name: fullMini}, true
			}
			return machine.Ref{Name: name}, false
		},
	})
	t.Cleanup(r.Close)
	s = serve.Server{
		Calls:       here.Lookup,
		ThisMachine: laptop,
		Streams: map[string]serve.Feed{"page": func(context.Context, json.RawMessage, func(any) error) error {
			t.Error("a watch of page for another machine ran this Mac's page stream")
			return nil
		}},
		Far: r,
	}
	return s, macs, engine
}

// value: every frame one watch line is answered with, until the server ends
// the connection, within a bound.
func watched(t *testing.T, s serve.Server, line string) []api.Frame {
	t.Helper()
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	go func() {
		s.Conn(t.Context(), struct {
			io.Reader
			io.Writer
			io.Closer
		}{inR, outW, closers{inR, outW}})
	}()
	go func() { _, _ = inW.Write([]byte(line + "\n")) }()
	got := make(chan []api.Frame, 1)
	go func() {
		var frames []api.Frame
		lines := bufio.NewScanner(outR)
		for lines.Scan() {
			var f api.Frame
			if json.Unmarshal(lines.Bytes(), &f) == nil {
				frames = append(frames, f)
			}
		}
		got <- frames
	}()
	bound, stop := context.WithTimeout(t.Context(), 5*time.Second)
	defer stop()
	select {
	case frames := <-got:
		_ = inW.Close()
		return frames
	case <-bound.Done():
		_ = inW.Close()
		t.Fatalf("%s: the watch never ended", line)
		return nil
	}
}

type closers []io.Closer

func (c closers) Close() error {
	for _, one := range c {
		_ = one.Close()
	}
	return nil
}

func outs(frames []api.Frame) string {
	var said []string
	for _, f := range frames {
		switch {
		case f.Error != nil:
			said = append(said, string(f.Error.Code)+" "+f.Error.Detail["label"])
		case f.Out != nil:
			said = append(said, string(f.Out))
		}
	}
	return strings.Join(said, " | ")
}

func TestAWatchFrameForAnotherMachineIsThatMachinesWatch(t *testing.T) {
	cases := []struct{ target, want string }{
		{``, `{"CPU":99}`},
		{`,"target":{"name":"mac-mini"}`, `{"CPU":12} | {"CPU":40}`},
		{`,"target":{"name":"linux-1"}`, `{"CPU":7}`},
	}
	for _, c := range cases {
		s, macs, engine := laptopSocket(t)
		frames := watched(t, s, `{"v":2,"id":"1","watch":"activity.read"`+c.target+`,"in":{}}`)
		if got := outs(frames); got != c.want {
			t.Errorf("activity watched%s: %s; want %s", c.target, got, c.want)
		}
		dialed := append(macs.Dialed(), engine.Dialed()...)
		switch {
		case strings.Contains(c.target, "mac-mini") && !slices.Equal(dialed, []string{fullMini}):
			t.Errorf("the mini by its short name dialed %v; want its full name, as a call's target", dialed)
		case strings.Contains(c.target, "linux-1") && !slices.Equal(dialed, []string{laptop.Ref.Name}):
			t.Errorf("linux-1 dialed %v; want its Mac's engine", dialed)
		case !strings.Contains(c.target, "mac-mini") && !strings.Contains(c.target, "linux-1") && len(dialed) != 0:
			t.Errorf("This Mac%s dialed %v; want it read here", c.target, dialed)
		}
	}
}

func TestAWatchOfAnotherMachineUnderAStreamsNameIsStillThatMachines(t *testing.T) {
	s, _, _ := laptopSocket(t)
	frames := watched(t, s, `{"v":2,"id":"1","watch":"page","target":{"name":"mac-mini"},"in":{}}`)
	if got := outs(frames); got != "remote.far-too-old mac-mini" {
		t.Errorf("page watched on the mini: %s; want the mini's refusal, which serves no page", got)
	}
}

func TestAWatchOfAMacThatDoesNotAnswerIsRefusedNamedNeverHeld(t *testing.T) {
	s, macs, _ := laptopSocket(t)
	macs.Down(fullMini)
	frames := watched(t, s, `{"v":2,"id":"1","watch":"activity.read","target":{"name":"mac-mini"},"in":{}}`)
	if got := outs(frames); got != "remote.unreachable mac-mini" || frames[0].Error.Message == "" {
		t.Errorf("activity of a silent mini: %s (%+v); want one worded refusal naming mac-mini, then the end", got, frames)
	}
}

func TestAFarWatchRefusedAfterItsFirstUpdateEndsWithTheRefusal(t *testing.T) {
	registry, err := call.NewRegistry([]call.Handler{call.BindWatch(call.Watch[struct{}, struct{}, contracttest.Sample]{
		Name: "activity.read", Where: call.AtTarget, Applies: machine.Anywhere,
		Run: func(_ context.Context, _ struct{}, _ struct{}, _ call.Progress) (<-chan call.Update[contracttest.Sample], error) {
			out := make(chan call.Update[contracttest.Sample], 2)
			out <- call.Update[contracttest.Sample]{Value: contracttest.Sample{CPU: 12}}
			out <- call.Update[contracttest.Sample]{Err: fail.New(fail.Refused, "machines.being-created", "", nil)}
			close(out)
			return out, nil
		},
	}, struct{}{})})
	if err != nil {
		t.Fatal(err)
	}
	s, _, _ := laptopSocketOver(t, registry)
	frames := watched(t, s, `{"v":2,"id":"1","watch":"activity.read","target":{"name":"mac-mini"},"in":{}}`)
	if got := outs(frames); !strings.HasPrefix(got, `{"CPU":12} | machines.being-created`) {
		t.Errorf("a mini's watch refused after its first update: %s; want the update, then its refusal", got)
	}
}
