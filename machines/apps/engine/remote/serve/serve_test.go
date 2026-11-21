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

var laptop = machine.Profile{Ref: machine.Ref{Name: "macbook-pro"}, Kind: machine.Computer}

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
	if err := c.Call(t.Context(), "machines.stop", machine.Ref{Name: "mac-mini"}, stopIn{VM: "infra", Confirm: "infra"}, &out, nil); err != nil || out.Wanted != "stopped there" {
		t.Fatalf("a call for mac-mini: %+v, %v, want the far answer", out, err)
	}
	out = stopOut{}
	if err := c.Call(t.Context(), "machines.stop", laptop.Ref, stopIn{VM: "linux-1", Confirm: "linux-1"}, &out, nil); err != nil || out.Wanted != "stopped" {
		t.Fatalf("a call for this Mac: %+v, %v, want it run here", out, err)
	}
	if strings.Join(far.asked, ",") != "mac-mini machines.stop" {
		t.Fatalf("far was asked %v, want mac-mini's stop alone", far.asked)
	}
}

func streamed(t *testing.T, caller string) (frames map[string]api.Frame, ran int) {
	t.Helper()
	registry, err := call.NewRegistry([]call.Handler{call.Bind(call.Call[struct{}, struct{}, struct{}]{
		Name: "ports.declare", Where: call.AtTarget, Applies: machine.Anywhere,
		Run: func(context.Context, struct{}, struct{}, call.Progress) (struct{}, error) {
			ran++
			return struct{}{}, nil
		},
	}, struct{}{})})
	if err != nil {
		t.Fatal(err)
	}
	in := `{"v":2,"id":"1","hello":{"version":"` + caller + `"}}` + "\n" + `{"v":2,"id":"2","call":"ports.declare"}` + "\n"
	var out strings.Builder
	if err := serve.Stream(t.Context(), strings.NewReader(in), &out, serve.Local(registry, laptop, "v0.21.0"), func(err error) *fail.Error {
		var f *fail.Error
		errors.As(err, &f)
		return f
	}); err != nil {
		t.Fatal(err)
	}
	frames = map[string]api.Frame{}
	for line := range strings.Lines(out.String()) {
		var f api.Frame
		if err := json.Unmarshal([]byte(line), &f); err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		frames[f.ID] = f
	}
	return frames, ran
}

func TestACallerOnAnotherReleaseIsAnsweredItsHelloAndRefusedEveryCall(t *testing.T) {
	for _, caller := range []string{"v0.0.9", "v0.20.0", "v3.2.0", "v0.22.0"} {
		frames, ran := streamed(t, caller)
		if frames["1"].Hello == nil || frames["1"].Hello.Version != "v0.21.0" {
			t.Errorf("the hello of a caller at %s was answered %+v, want this station's hello", caller, frames["1"])
		}
		refused := frames["2"].Error
		if ran != 0 || refused == nil || refused.Code != call.CallerTooOld || refused.Detail["version"] != caller {
			t.Errorf("a call from a caller at %s ran %d times and answered %+v, want it refused unrun as %s", caller, ran, refused, call.CallerTooOld)
		}
	}
	for _, caller := range []string{"v0.21.0", "v0.21.0-2-gabcdef1", "dev"} {
		if frames, ran := streamed(t, caller); ran != 1 || frames["2"].Error != nil {
			t.Errorf("a call from a caller at %s ran %d times and answered %+v, want it run", caller, ran, frames["2"].Error)
		}
	}
}

type said struct{ details []string }

func (s *said) Step(_, detail string) { s.details = append(s.details, detail) }
func (s *said) Done(string, error)    {}

func TestAnErrorAndAProgressLeaveWithTheirTokensWithheld(t *testing.T) {
	token := "s" + "k-" + strings.Repeat("Ab3dE", 8)
	ref, spec := "refs/heads/feat/redaction-pre-verify1-2026-11-13", "specs/usecases/machinefile/apply/a-failing-line-withholds-its-token.spec.yaml"
	fetch := call.Bind(call.Call[struct{}, struct{}, struct{}]{
		Name: "machinefile.fetch", Where: call.InEngine, Applies: machine.Anywhere,
		Run: func(_ context.Context, _ struct{}, _ struct{}, p call.Progress) (struct{}, error) {
			p.Step("Fetching", "fetching https://owner:"+token+"@example.test/repo")
			return struct{}{}, &fail.Error{Message: "fetch refused for https://owner:" + token + "@example.test/repo at " + ref + " of " + spec, Hint: "token=" + token, Subject: token, Detail: map[string]string{"url": "?token=" + token}}
		},
	}, struct{}{})
	s := serve.Server{Calls: func(name string) (call.Handler, bool) { return fetch, name == "machinefile.fetch" }, ThisMachine: laptop}
	var p said
	err := dialed(t.Context(), s).Call(t.Context(), "machinefile.fetch", machine.Ref{}, struct{}{}, nil, &p)
	var e *fail.Error
	if !errors.As(err, &e) {
		t.Fatalf("the failing call answered %v, want its error", err)
	}
	heard := strings.Join(p.details, "|") + "|" + e.Message + "|" + e.Hint + "|" + e.Subject + "|" + e.Detail["url"]
	if strings.Contains(heard, token) || !strings.Contains(heard, "withheld") {
		t.Fatalf("a call's words over the socket: %s, want the token withheld", heard)
	}
	if !strings.HasSuffix(e.Message, " at "+ref+" of "+spec) {
		t.Errorf("the error's ref and path were not kept: %q", e.Message)
	}
}

// value: a caller's side that tells when the server has read its end.
type closing struct {
	r     io.Reader
	ended chan struct{}
}

func (c closing) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	if errors.Is(err, io.EOF) {
		select {
		case <-c.ended:
		default:
			close(c.ended)
		}
	}
	return n, err
}

// value: a watch whose frame answers at once and whose last value only once
// the test lets it.
type lateWatch struct{ release chan struct{} }

func (lateWatch) Hello() api.Hello { return api.Hello{} }

func (lateWatch) Invoke(context.Context, machine.Ref, string, json.RawMessage, call.Progress) (any, error) {
	return nil, errors.New("no call here")
}

func (w lateWatch) Watch(ctx context.Context, _ machine.Ref, _ string, _ json.RawMessage, _ call.Progress) (<-chan call.Update[any], error) {
	out := make(chan call.Update[any])
	go func() {
		defer close(out)
		for i, value := range []string{"frame", "last"} {
			if i > 0 {
				select {
				case <-w.release:
				case <-ctx.Done():
					return
				}
			}
			select {
			case out <- call.Update[any]{Value: value}:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}

func TestAWatchAnswersItsLastValueAfterItsCallerClosedItsSide(t *testing.T) {
	in := closing{r: strings.NewReader(`{"v":2,"id":"1","watch":"disk.read"}` + "\n"), ended: make(chan struct{})}
	outR, outW := io.Pipe()
	w := lateWatch{release: make(chan struct{})}
	served := make(chan error, 1)
	go func() {
		served <- serve.Stream(t.Context(), in, outW, w, nil)
		_ = outW.Close()
	}()
	bounded, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	select {
	case <-in.ended:
	case <-bounded.Done():
		t.Fatal("the server never read its caller's end")
	}
	close(w.release)
	lines := make(chan []string, 1)
	go func() {
		var got []string
		s := bufio.NewScanner(outR)
		for s.Scan() {
			var f api.Frame
			if json.Unmarshal(s.Bytes(), &f) == nil {
				got = append(got, string(f.Out)+doneMark(f.Done))
			}
		}
		lines <- got
	}()
	var got []string
	select {
	case got = <-lines:
	case <-bounded.Done():
		t.Fatal("the server never ended once its watch had")
	}
	if want := []string{`"frame"`, `"last"`, "done"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("the watch answered %v after its caller's end, want %v: a slow last value would be lost", got, want)
	}
	if err := <-served; err != nil {
		t.Fatalf("serve: %v", err)
	}
}

func doneMark(done bool) string {
	if done {
		return "done"
	}
	return ""
}

func TestAWatchOverTheSocketEndsOnItsDoneBeforeItsConnectionCloses(t *testing.T) {
	read := call.BindWatch(call.Watch[struct{}, struct{}, string]{
		Name: "overview.read", Where: call.AtTarget, Applies: machine.Anywhere,
		Run: func(context.Context, struct{}, struct{}, call.Progress) (<-chan call.Update[string], error) {
			out := make(chan call.Update[string], 2)
			out <- call.Update[string]{Value: "pending"}
			out <- call.Update[string]{Value: "whole"}
			close(out)
			return out, nil
		},
	}, struct{}{})
	s := serve.Server{Calls: func(name string) (call.Handler, bool) { return read, name == "overview.read" }, ThisMachine: laptop}
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	go s.Conn(t.Context(), pipe{Reader: inR, Writer: outW, closers: []io.Closer{inR, outW}})
	go func() { _, _ = inW.Write([]byte(`{"v":2,"id":"1","watch":"overview.read"}` + "\n")) }()
	var got []string
	lines := bufio.NewScanner(outR)
	for lines.Scan() {
		var f api.Frame
		if json.Unmarshal(lines.Bytes(), &f) == nil {
			got = append(got, f.ID+" "+string(f.Out)+doneMark(f.Done))
		}
	}
	if want := []string{`1 "pending"`, `1 "whole"`, "1 done"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("the socket answered %v, want %v: a client cannot tell a watch's end from a dropped connection", got, want)
	}
}
