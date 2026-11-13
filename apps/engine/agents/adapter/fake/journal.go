package fake

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"sync"

	"github.com/jterrazz/studio-station/apps/engine/agents"
	"github.com/jterrazz/studio-station/apps/engine/agents/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

type Journal struct {
	// invariant: the journal of one VM of the world, read on the world's
	// clock, so an agent event of the world is read at the next Read; a
	// write through it, or Changed, wakes its watches.
	inVM
	mu       sync.Mutex
	watchers []chan struct{}
}

var _ contract.Journal = (*Journal)(nil)

func JournalOf(w *world.World, vm string) *Journal {
	return &Journal{inVM: inVM{w: w, vm: vm}}
}

func NewJournal() *Journal { return &Journal{inVM: alone()} }

// value: what a scripted fault of Journal.Read or Journal.Write answers.
var errJournal = errors.New("the agents' journal could not be read")

func (j *Journal) World() *world.World { return j.w }

func (j *Journal) read() []agents.Agent {
	var out []agents.Agent
	held := j.agents().Journal
	for _, name := range slices.Sorted(maps.Keys(held)) {
		if !held[name].Expired(j.w.Now) {
			out = append(out, held[name])
		}
	}
	return out
}

func (j *Journal) Read(context.Context) ([]agents.Agent, error) {
	j.w.Lock()
	defer j.w.Unlock()
	if j.w.Fails("Journal.Read") {
		return nil, errJournal
	}
	return j.read(), nil
}

func (j *Journal) Write(_ context.Context, a agents.Agent) error {
	j.w.Lock()
	defer j.w.Unlock()
	if j.w.Fails("Journal.Write") {
		return errJournal
	}
	if !agents.PlainSession(a.Session) {
		return errors.New("no session is named " + a.Session)
	}
	held := j.agents()
	if held.Journal == nil {
		held.Journal = map[string]agents.Agent{}
	}
	if len(a.Steps) > catalog.AgentStepsKept {
		a.Steps = slices.Clone(a.Steps[len(a.Steps)-catalog.AgentStepsKept:])
	}
	held.Journal[a.Session] = a
	maps.DeleteFunc(held.Journal, func(_ string, kept agents.Agent) bool { return kept.Expired(j.w.Now) })
	j.Changed()
	return nil
}

func (j *Journal) Forget(_ context.Context, session string) error {
	j.w.Lock()
	defer j.w.Unlock()
	if j.w.Fails("Journal.Write") {
		return errJournal
	}
	delete(j.agents().Journal, session)
	j.Changed()
	return nil
}

func (j *Journal) Changed() {
	j.mu.Lock()
	defer j.mu.Unlock()
	for _, w := range j.watchers {
		select {
		case w <- struct{}{}:
		default:
		}
	}
}

func (j *Journal) Watch(ctx context.Context) (<-chan []agents.Agent, error) {
	changed := make(chan struct{}, 1)
	j.mu.Lock()
	j.watchers = append(j.watchers, changed)
	j.mu.Unlock()
	out := make(chan []agents.Agent)
	go func() {
		defer close(out)
		defer func() {
			j.mu.Lock()
			defer j.mu.Unlock()
			j.watchers = slices.DeleteFunc(j.watchers, func(w chan struct{}) bool { return w == changed })
		}()
		var last []byte
		for {
			j.w.Lock()
			now := j.read()
			j.w.Unlock()
			if data, _ := json.Marshal(now); last == nil || !bytes.Equal(data, last) {
				select {
				case out <- now:
					last = data
				case <-ctx.Done():
					return
				}
			}
			select {
			case <-changed:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}
