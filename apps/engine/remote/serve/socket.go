package serve

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/remote/api"
)

const stall = 5 * time.Second

type Feed func(ctx context.Context, in json.RawMessage, send func(out any) error) error

type Server struct {
	// value: the calls this server runs, by name: the registry's, and the
	// host's own.
	Calls func(name string) (call.Handler, bool)
	// value: what the server hosts under a watch name of its own: a stream
	// sends each update and returns when it ends; an error it returns is the
	// watch's refusal, after an update as before.
	Streams map[string]Feed
	// value: the machine the calls run on: the gate of every call frame.
	ThisMachine machine.Profile
	// value: words a fault that carries only its code.
	Word func(*fail.Error) (message, hint string)
	// value: the hint of an error no table words.
	Hint string
	// value: the clock a stall is measured by; nil writes unguarded.
	Sleep func(ctx context.Context, d time.Duration) error
	// value: how long a write may block before the connection is dropped.
	Stall time.Duration
	// value: where a call frame for another machine runs: that machine's
	// engine, over this run's connections; nil runs every frame on this machine.
	Far call.Router
}

func (s Server) Conn(ctx context.Context, conn io.ReadWriteCloser) {
	// invariant: the calls of a connection are answered in the order they came;
	// a watch takes the connection over until either side ends it, and a later
	// line on it is read and ignored.
	defer func() { _ = conn.Close() }()
	w := &writer{w: conn}
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var f api.Frame
		if err := json.Unmarshal(scanner.Bytes(), &f); err != nil || f.V != api.Version || (f.Call == "") == (f.Watch == "") {
			if w.frame(api.Frame{ID: f.ID, Error: s.fault(badFrame(f))}) != nil {
				return
			}
			continue
		}
		if f.Watch != "" {
			s.watch(ctx, conn, scanner, w, f)
			return
		}
		if w.frame(s.call(ctx, w, f)) != nil {
			return
		}
	}
}

func badFrame(f api.Frame) *fail.Error {
	return fail.New(fail.Refused, call.BadFrame, api.Name, map[string]string{"v": strconv.Itoa(f.V)})
}

func (s Server) handler(name string) (call.Handler, error) {
	if s.Calls != nil {
		if h, ok := s.Calls(name); ok {
			return h, nil
		}
	}
	return nil, fail.New(fail.NotFound, call.Unknown, name, nil)
}

// invariant: a frame whose target is another machine runs there, its
// answer relayed as it came; any other frame runs on this machine.
func (s Server) call(ctx context.Context, w *writer, f api.Frame) api.Frame {
	if t := f.Target.Ref(); s.Far != nil && !s.thisMachine(t) {
		var out json.RawMessage
		if err := s.Far.Invoke(ctx, t, f.Call, f.In, &out, progress{id: f.ID, out: w, word: s.wording}); err != nil {
			return api.Frame{ID: f.ID, Error: s.fault(err)}
		}
		return api.Frame{ID: f.ID, Out: orNull(out)}
	}
	h, err := s.handler(f.Call)
	if err != nil {
		return api.Frame{ID: f.ID, Error: s.fault(err)}
	}
	out, err := h.Invoke(ctx, s.ThisMachine, f.In, progress{id: f.ID, out: w, word: s.wording})
	if err != nil {
		return api.Frame{ID: f.ID, Error: s.fault(err)}
	}
	if len(out) == 0 {
		out = json.RawMessage("null")
	}
	return api.Frame{ID: f.ID, Out: out}
}

func (s Server) thisMachine(t machine.Ref) bool {
	return t == (machine.Ref{}) || t == s.ThisMachine.Ref || t.Name == s.ThisMachine.Ref.Name
}

func (s Server) watch(ctx context.Context, conn io.ReadWriteCloser, lines *bufio.Scanner, w *writer, f api.Frame) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		for lines.Scan() {
		}
		cancel()
	}()
	send := s.guarded(ctx, conn, func(out any) error {
		data, err := json.Marshal(out)
		if err != nil {
			return err
		}
		return w.frame(api.Frame{ID: f.ID, Out: data})
	})
	var err error
	if t := f.Target.Ref(); s.Far != nil && !s.thisMachine(t) {
		err = s.far(ctx, t, w, f, send)
	} else if stream, ok := s.Streams[f.Watch]; ok {
		err = stream(ctx, f.In, send)
	} else {
		err = s.updates(ctx, f, send)
	}
	if err != nil {
		_ = w.frame(api.Frame{ID: f.ID, Error: s.fault(err)})
	}
}

// invariant: a watch whose target is another machine is that machine's
// watch, found as a call's target is and relayed update by update; its end
// there ends it here, and its end here ends it there.
func (s Server) far(ctx context.Context, t machine.Ref, w *writer, f api.Frame, send func(any) error) error {
	updates, err := s.Far.Watch(ctx, t, f.Watch, f.In, progress{id: f.ID, out: w, word: s.wording})
	if err != nil {
		return err
	}
	for u := range updates {
		if u.Err != nil {
			return u.Err
		}
		if err := send(u.Value); err != nil {
			return err
		}
	}
	return nil
}

func (s Server) updates(ctx context.Context, f api.Frame, send func(any) error) error {
	h, err := s.handler(f.Watch)
	if err != nil {
		return err
	}
	updates, err := h.Watch(ctx, s.ThisMachine, f.In, nil)
	if err != nil {
		return err
	}
	for u := range updates {
		if u.Err != nil {
			return u.Err
		}
		if err := send(u.Value); err != nil {
			return err
		}
	}
	return nil
}

// invariant: a write that does not finish within the stall budget closes
// the connection: the server never blocks on a client that does not read.
func (s Server) guarded(ctx context.Context, conn io.Closer, write func(any) error) func(any) error {
	limit := s.Stall
	if limit <= 0 {
		limit = stall
	}
	if s.Sleep == nil {
		return write
	}
	return func(v any) error {
		wctx, done := context.WithCancel(ctx)
		defer done()
		go func() {
			if s.Sleep(wctx, limit) == nil && wctx.Err() == nil {
				_ = conn.Close()
			}
		}()
		return write(v)
	}
}

func (s Server) fault(err error) *call.WireError { return call.ToWire(err, s.wording) }

func (s Server) wording(err error) *fail.Error {
	var e *fail.Error
	if !errors.As(err, &e) {
		e = &fail.Error{Type: fail.Failed, Code: "internal", Message: err.Error()}
	}
	out := *e
	if out.Type == "" {
		out.Type = fail.Failed
		if out.Exit == fail.ExitUsage {
			out.Type = fail.Refused
		}
	}
	if out.Message == "" && s.Word != nil {
		out.Message, out.Hint = s.Word(e)
	}
	if out.Message == "" {
		out.Message = e.Error()
	}
	if out.Hint == "" {
		out.Hint = s.Hint
	}
	return &out
}
