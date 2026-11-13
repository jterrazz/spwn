package serve

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"sync"
	"sync/atomic"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/remote/api"
)

type Calls interface {
	// value: what a server runs a frame with: this binary's Router, the
	// call's In decoded from the frame and its Out handed back as it is.
	Hello() api.Hello
	Invoke(ctx context.Context, target machine.Ref, name string, in json.RawMessage, p call.Progress) (any, error)
	Watch(ctx context.Context, target machine.Ref, name string, in json.RawMessage, p call.Progress) (<-chan call.Update[any], error)
}

const maxFrame = 64 << 20

func Stream(ctx context.Context, r io.Reader, w io.Writer, calls Calls, word call.Wording) error {
	// invariant: serves frames until the caller closes its side; each
	// request runs on its own, its answers written whole, one line each, and
	// the server returns once every request has answered.
	// invariant: a caller whose hello names a version older than
	// api.WordsSince, or names none as an older station's does, is answered
	// in the words it reads (ADR-032) from that hello on; a caller that sends
	// no hello is answered in today's.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	out := &writer{w: w}
	var running sync.WaitGroup
	defer running.Wait()
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 0, 64<<10), maxFrame)
	for s.Scan() {
		var f api.Frame
		if err := json.Unmarshal(s.Bytes(), &f); err != nil {
			continue
		}
		if f.Hello != nil {
			out.older.Store(api.Older(f.Hello.Version, calls.Hello().Version))
		}
		running.Go(func() { answer(ctx, f, out, calls, word) })
	}
	return s.Err()
}

func answer(ctx context.Context, f api.Frame, out *writer, calls Calls, word call.Wording) {
	reply := func(r api.Frame) {
		r.V, r.ID = api.Version, f.ID
		out.write(r)
	}
	p := progress{id: f.ID, out: out, word: word}
	switch {
	case f.Hello != nil:
		h := calls.Hello()
		reply(api.Frame{Hello: &h})
	case f.Call != "":
		v, err := calls.Invoke(ctx, f.Target.Ref(), f.Call, f.In, p)
		if err != nil {
			reply(api.Frame{Error: call.ToWire(err, word)})
			return
		}
		raw, err := call.Encode(v, word)
		if err != nil {
			reply(api.Frame{Error: call.ToWire(err, word)})
			return
		}
		reply(api.Frame{Out: orNull(raw)})
	case f.Watch != "":
		updates, err := calls.Watch(ctx, f.Target.Ref(), f.Watch, f.In, p)
		if err != nil {
			reply(api.Frame{Error: call.ToWire(err, word)})
			return
		}
		for u := range updates {
			if u.Err != nil {
				reply(api.Frame{Error: call.ToWire(u.Err, word)})
				continue
			}
			raw, err := call.Encode(u.Value, word)
			if err != nil {
				reply(api.Frame{Error: call.ToWire(err, word)})
				continue
			}
			reply(api.Frame{Out: orNull(raw)})
		}
		reply(api.Frame{Done: true})
	}
}

func orNull(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage("null")
	}
	return raw
}

type writer struct {
	mu sync.Mutex
	w  io.Writer
	// value: the caller reads only the words before api.WordsSince.
	older atomic.Bool
}

func (w *writer) write(f api.Frame) { _ = w.frame(f) }

func (w *writer) frame(f api.Frame) error {
	f.V = api.Version
	data, err := json.Marshal(f)
	if err != nil {
		return err
	}
	if w.older.Load() {
		data = api.OlderWords(data)
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	_, err = w.w.Write(append(data, '\n'))
	return err
}

type progress struct {
	id   string
	out  *writer
	word call.Wording
}

func (p progress) Step(task, detail string) {
	p.out.write(api.Frame{V: api.Version, ID: p.id, Progress: &api.Progress{Task: task, Detail: detail}})
}

func (p progress) Done(task string, err error) {
	p.out.write(api.Frame{V: api.Version, ID: p.id, Progress: &api.Progress{Task: task, Done: true, Error: call.ToWire(err, p.word)}})
}
