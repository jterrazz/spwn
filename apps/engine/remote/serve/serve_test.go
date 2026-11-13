package serve_test

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/remote/api"
	"github.com/jterrazz/studio-station/apps/engine/remote/serve"
)

type pipe struct {
	io.Reader
	io.Writer
	closers []io.Closer
}

func (p pipe) Close() error {
	for _, c := range p.closers {
		_ = c.Close()
	}
	return nil
}

// value: a client whose every connection is a fresh pipe the server serves.
func dialed(ctx context.Context, s serve.Server) api.Client {
	return api.NewClient(func(context.Context) (io.ReadWriteCloser, error) {
		inR, inW := io.Pipe()
		outR, outW := io.Pipe()
		go s.Conn(ctx, pipe{Reader: inR, Writer: outW, closers: []io.Closer{inR, outW}})
		return pipe{Reader: outR, Writer: inW, closers: []io.Closer{outR, inW}}, nil
	})
}

type stopIn struct {
	VM      string
	Confirm string
}

type stopOut struct{ Wanted string }

type steps struct{ seen []string }

func (s *steps) Step(task, _ string)       { s.seen = append(s.seen, task) }
func (s *steps) Done(task string, _ error) { s.seen = append(s.seen, task+" done") }

var laptop = machine.Profile{Ref: machine.Ref{Name: "jterrazz-macbook-pro"}, Kind: machine.Computer}

func server(t *testing.T) serve.Server {
	t.Helper()
	stop := call.Bind(call.Call[struct{}, stopIn, stopOut]{
		Name: "machines.stop", Where: call.InEngine, Applies: machine.Anywhere, Confirm: confirm.Delete,
		Run: func(_ context.Context, _ struct{}, in stopIn, p call.Progress) (stopOut, error) {
			if err := confirm.Require(confirm.TypeName, in.VM, in.Confirm); err != nil {
				return stopOut{}, err
			}
			p.Step("Stopping "+in.VM, "graceful, 60 s")
			p.Done("Stopping "+in.VM, nil)
			return stopOut{Wanted: "stopped"}, nil
		},
	}, struct{}{})
	return serve.Server{
		Calls:       func(name string) (call.Handler, bool) { return stop, name == "machines.stop" },
		ThisMachine: laptop,
		Streams: map[string]serve.Feed{"page": func(ctx context.Context, _ json.RawMessage, send func(any) error) error {
			for i := 0; ; i++ {
				if err := send(map[string]int{"page": i}); err != nil {
					return err
				}
				if ctx.Err() != nil {
					return ctx.Err()
				}
			}
		}},
	}
}

func TestACallAnswersItsProgressThenItsOut(t *testing.T) {
	c := dialed(t.Context(), server(t))
	var out stopOut
	var p steps
	err := c.Call(t.Context(), "machines.stop", machine.Ref{Name: "infra"}, stopIn{VM: "infra", Confirm: "infra"}, &out, &p)
	if err != nil || out.Wanted != "stopped" || strings.Join(p.seen, ",") != "Stopping infra,Stopping infra done" {
		t.Fatalf("stopping infra over the socket: %+v, %v, progress %v", out, err, p.seen)
	}
}

func TestARefusalComesBackAsTheErrorItWasRaisedAs(t *testing.T) {
	c := dialed(t.Context(), server(t))
	err := c.Call(t.Context(), "machines.stop", machine.Ref{}, stopIn{VM: "infra", Confirm: "inf"}, nil, nil)
	var e *fail.Error
	if !errors.As(err, &e) || e.Type != fail.NeedsConfirmation || e.Code != confirm.Mismatch || e.Subject != "infra" || e.Detail["typed"] != "inf" || fail.ExitOf(err) != fail.ExitUsage {
		t.Fatalf("a mistyped confirmation over the socket: %#v, want needs-confirmation %s naming infra", err, confirm.Mismatch)
	}
	err = c.Call(t.Context(), "machines.nope", machine.Ref{}, nil, nil, nil)
	if !errors.As(err, &e) || e.Code != call.Unknown || e.Subject != "machines.nope" {
		t.Fatalf("an unknown call: %v, want %s", err, call.Unknown)
	}
}

func TestALineThatIsNoFrameIsRefusedAndTheConnectionAnswersTheNext(t *testing.T) {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	go server(t).Conn(t.Context(), pipe{Reader: inR, Writer: outW, closers: []io.Closer{inR, outW}})
	go func() {
		_, _ = inW.Write([]byte(`{"v":1,"id":"1","op":"ping"}` + "\n" + `{"v":2,"id":"2","call":"machines.stop","in":{"VM":"infra","Confirm":"infra"}}` + "\n"))
	}()
	r := bufio.NewReader(outR)
	var answers []string
	for len(answers) < 2 {
		line, err := r.ReadBytes('\n')
		if err != nil {
			t.Fatal(err)
		}
		var f api.Frame
		_ = json.Unmarshal(line, &f)
		switch {
		case f.Error != nil:
			answers = append(answers, f.ID+" "+string(f.Error.Code))
		case f.Out != nil:
			answers = append(answers, f.ID+" "+string(f.Out))
		}
	}
	_ = inW.Close()
	if got := strings.Join(answers, " | "); got != `1 call.bad-frame | 2 {"Wanted":"stopped"}` {
		t.Fatalf("answers %s", got)
	}
}

func TestAWatcherThatDoesNotReadIsDropped(t *testing.T) {
	ended := make(chan struct{})
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	s := server(t)
	s.Sleep = func(context.Context, time.Duration) error { return nil }
	go func() {
		s.Conn(t.Context(), pipe{Reader: inR, Writer: outW, closers: []io.Closer{inR, outW}})
		close(ended)
	}()
	go func() { _, _ = inW.Write([]byte(`{"v":2,"id":"1","watch":"page"}` + "\n")) }()
	<-ended
	if _, err := io.ReadAll(outR); err != nil {
		t.Fatalf("a watcher that stopped reading still holds its connection: %v", err)
	}
}

type farRouter struct{ asked []string }

func (f *farRouter) Invoke(_ context.Context, target machine.Ref, name string, _, out any, _ call.Progress) error {
	f.asked = append(f.asked, target.Name+" "+name)
	raw, ok := out.(*json.RawMessage)
	if !ok {
		return errors.New("not raw")
	}
	*raw = json.RawMessage(`{"Wanted":"stopped there"}`)
	return nil
}

func (f *farRouter) Watch(context.Context, machine.Ref, string, any, call.Progress) (<-chan call.Update[any], error) {
	return nil, errors.New("no watch")
}

func TestACallForAnotherMachineRidesFarAndOneForThisMachineRunsHere(t *testing.T) {
	far := &farRouter{}
	s := server(t)
	s.Far = far
	c := dialed(t.Context(), s)
	var out stopOut
	if err := c.Call(t.Context(), "machines.stop", machine.Ref{Name: "jterrazz-mac-mini"}, stopIn{VM: "infra", Confirm: "infra"}, &out, nil); err != nil || out.Wanted != "stopped there" {
		t.Fatalf("a call for mac-mini: %+v, %v, want the far answer", out, err)
	}
	out = stopOut{}
	if err := c.Call(t.Context(), "machines.stop", laptop.Ref, stopIn{VM: "linux-1", Confirm: "linux-1"}, &out, nil); err != nil || out.Wanted != "stopped" {
		t.Fatalf("a call for this Mac: %+v, %v, want it run here", out, err)
	}
	if strings.Join(far.asked, ",") != "jterrazz-mac-mini machines.stop" {
		t.Fatalf("far was asked %v, want mac-mini's stop alone", far.asked)
	}
}
