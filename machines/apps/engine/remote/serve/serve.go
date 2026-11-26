package serve

import (
	"bufio"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"io"
	"slices"
	"sync"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/goes"
	"github.com/jterrazz/studio-station/apps/engine/core/guide"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/secret"
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
	// the server returns once every request has answered. A caller whose
	// hello names another release than this station's is answered its hello,
	// and every request after it is refused unrun.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	out := &writer{w: w}
	var running sync.WaitGroup
	defer running.Wait()
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 0, 64<<10), maxFrame)
	caller := ""
	answering := requests{ends: map[string]*request{}}
	for s.Scan() {
		var f api.Frame
		if err := json.Unmarshal(s.Bytes(), &f); err != nil {
			continue
		}
		if f.Cancel {
			answering.end(f.ID)
			continue
		}
		if f.Hello != nil && api.SkewOf(calls.Hello().Version, f.Hello.Version) != api.Speaks {
			caller = f.Hello.Version
		}
		if caller != "" && f.Hello == nil {
			out.write(api.Frame{ID: f.ID, Error: call.ToWire(tooOld(caller), word)})
			continue
		}
		rctx, done := answering.begin(ctx, f.ID)
		running.Go(func() {
			defer done()
			named := cmp.Or(f.Call+f.Watch, "hello")
			if err := goes.TryOf("stream.answer", named, func() error { answer(rctx, f, out, calls, word); return nil }); err != nil {
				out.write(api.Frame{ID: f.ID, Error: call.ToWire(broken(err, named, ""), word)})
			}
		})
	}
	return s.Err()
}

// invariant: the requests of one stream still answering, by id, each ended
// by a cancel that names it; a request that ended is no longer there.
type requests struct {
	mu   sync.Mutex
	ends map[string]*request
}

type request struct{ cancel context.CancelFunc }

func (r *requests) begin(ctx context.Context, id string) (answering context.Context, done func()) {
	answering, cancel := context.WithCancel(ctx)
	this := &request{cancel: cancel}
	r.mu.Lock()
	r.ends[id] = this
	r.mu.Unlock()
	return answering, func() {
		r.mu.Lock()
		if r.ends[id] == this {
			delete(r.ends, id)
		}
		r.mu.Unlock()
		cancel()
	}
}

func (r *requests) end(id string) {
	r.mu.Lock()
	this := r.ends[id]
	r.mu.Unlock()
	if this != nil {
		this.cancel()
	}
}

func answer(ctx context.Context, f api.Frame, out *writer, calls Calls, word call.Wording) {
	reply := func(r api.Frame) {
		r.V, r.ID = api.Version, f.ID
		out.write(r)
	}
	p := progress{id: f.ID, out: out, word: word}
	if f.Hello == nil {
		if err := call.FrameBy(f.Call+f.Watch, f.By); err != nil {
			reply(api.Frame{Error: call.ToWire(err, word)})
			return
		}
		ctx = call.WithBy(ctx, f.By)
		if f.Far {
			ctx = call.WithFar(ctx)
		}
	}
	switch {
	case f.Hello != nil:
		h := calls.Hello()
		reply(api.Frame{Hello: &h})
	case f.Call != "":
		var run call.Progress = p
		g := &guided{progress: p}
		if gc, ok := calls.(GuidedCalls); ok {
			meta, acts := gc.Guide(f.Target.Ref(), f.Call)
			g.family, g.acts, g.repairs = meta.Guide, acts, repairOf(meta, f.In)
			if g.family != "" && g.acts != nil {
				run = g
			}
		}
		var v any
		err := broken(goes.TryOf("stream.call", f.Call, func() error {
			var err error
			v, err = calls.Invoke(ctx, f.Target.Ref(), f.Call, f.In, run)
			return err
		}), f.Call, "")
		g.end(err)
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
		err := broken(goes.TryOf("stream.watch", f.Watch, func() error { return watched(ctx, f, calls, p, word, reply) }), f.Watch, "")
		if err != nil {
			reply(api.Frame{Error: call.ToWire(err, word)})
			return
		}
		reply(api.Frame{Done: true})
	}
}

// invariant: a watch's updates, each replied as it comes; the error that
// ends it is answered by the caller, as the watch's last frame.
func watched(ctx context.Context, f api.Frame, calls Calls, p call.Progress, word call.Wording, reply func(api.Frame)) error {
	updates, err := calls.Watch(ctx, f.Target.Ref(), f.Watch, f.In, p)
	if err != nil {
		return err
	}
	for u := range updates {
		if u.Err != nil {
			return u.Err
		}
		raw, err := call.Encode(u.Value, word)
		if err != nil {
			return err
		}
		reply(api.Frame{Out: orNull(raw)})
	}
	return nil
}

func tooOld(version string) error {
	return fail.New(fail.Refused, call.CallerTooOld, "", map[string]string{"version": version})
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
}

func (w *writer) write(f api.Frame) { _ = w.frame(f) }

func (w *writer) frame(f api.Frame) error {
	f.V = api.Version
	data, err := json.Marshal(f)
	if err != nil {
		return err
	}
	secret.Tripwire(data)
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
	p.out.write(api.Frame{V: api.Version, ID: p.id, Progress: &api.Progress{Task: task, Detail: secret.Kept(detail)}})
}

func (p progress) Done(task string, err error) {
	p.out.write(api.Frame{V: api.Version, ID: p.id, Progress: &api.Progress{Task: task, Done: true, Error: call.ToWire(err, p.word)}})
}

type Acts interface {
	// value: the engine's act board: a guided call's phase kept under its
	// act, its id answered, and its first Guide when the phase put it; its
	// end, with the failure it ended on, nil when it was made; and the
	// refusal of a repair its Guide offered once it ended.
	Phase(family string, ph guide.Phase) (string, *guide.Guide)
	End(id string, ended guide.Ended, failed *call.WireError)
	Refused(id string, refusal *call.WireError)
}

type GuidedCalls interface {
	// value: a stream's calls that are the window's door onto an engine:
	// the call as this machine's registry declares it (its Guide and its
	// Repairs) and the board its acts are kept on; a zero Meta and no
	// board for a call that runs at another machine.
	Guide(target machine.Ref, name string) (call.Meta, Acts)
}

func repairOf(m call.Meta, in json.RawMessage) string {
	if m.Repairs == nil {
		return ""
	}
	family, subject := m.Repairs(in)
	if family == "" || subject == "" {
		return ""
	}
	return guide.ID(family, subject)
}

type guided struct {
	progress
	family string
	acts   Acts
	// value: the act this call is the repair of, empty for a call that
	// repairs none.
	repairs string
	act     string
	step    string
	// value: the step each task was told at, which its end belongs to.
	told map[string]string
	// value: the act's first Guide, until the frame that first names the
	// act carries it.
	first *guide.Guide
}

func (g *guided) Phase(ph guide.Phase) {
	act, first := g.acts.Phase(g.family, ph)
	g.act, g.step = act, ph.Step
	if first != nil {
		g.first = first
	}
}

func (g *guided) Step(task, detail string) {
	if g.told == nil {
		g.told = map[string]string{}
	}
	g.told[task] = g.step
	first := g.first
	g.first = nil
	g.out.write(api.Frame{V: api.Version, ID: g.id, Progress: &api.Progress{Task: task, Detail: secret.Kept(detail), Act: g.act, Step: g.step, Guide: first}})
}

func (g *guided) Done(task string, err error) {
	// invariant: a task's end belongs to the step that told it, though the
	// next step may already be reached.
	step, told := g.told[task]
	if !told {
		step = g.step
	}
	g.out.write(api.Frame{V: api.Version, ID: g.id, Progress: &api.Progress{Task: task, Done: true, Error: call.ToWire(err, g.word), Act: g.act, Step: step}})
}

func (g *guided) end(err error) {
	// invariant: a call cut by its context did not fail: it ends interrupted
	// at its step, as an engine that stopped under it does. A repair the
	// engine refused is told to the act it repairs.
	switch {
	case g.act == "" && g.repairs != "" && g.acts != nil && err != nil:
		if w := call.ToWire(err, g.word); refusal(err, w) {
			g.acts.Refused(g.repairs, w)
		}
	case g.act == "":
	case err == nil:
		g.acts.End(g.act, guide.EndedDone, nil)
	case errors.Is(err, context.Canceled):
		g.acts.End(g.act, guide.EndedInterrupted, nil)
	default:
		g.acts.End(g.act, guide.EndedFailed, call.ToWire(err, g.word))
	}
}

func refusal(err error, w *call.WireError) bool {
	// invariant: a question the call asks first, the owner's no, and a call
	// cut by its context are no refusal of the repair.
	if errors.Is(err, context.Canceled) || w == nil {
		return false
	}
	return !slices.Contains([]fail.Type{fail.NeedsConfirmation, fail.Declined, fail.Interrupted}, w.Type)
}
