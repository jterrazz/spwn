package process

import (
	"encoding/json"
	"slices"
	"sync"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/guide"
	"github.com/jterrazz/studio-station/apps/engine/machines"
)

const (
	// value: how long an act that ended stays on the board, so a window
	// that was hidden still reads how it ended.
	ActLinger = 10 * time.Minute
)

type ActFile interface {
	// value: the board's one file, read and written whole; a file never
	// written reads as no bytes.
	Read() ([]byte, error)
	Write(data []byte) error
}

type ActOwners interface {
	Self() (machines.Process, error)
	Runs(p machines.Process) bool
}

type actEntry struct {
	guide.Act
	// value: the process the act's call runs in: an unended act whose
	// process no longer runs was cut by the engine stopping under it.
	Owner *machines.Process `json:"owner,omitempty"`
}

type actsKept struct {
	Acts []actEntry `json:"acts"`
}

type Acts struct {
	// invariant: the file is the board: each write reads it, changes its
	// own act and writes it back whole, then puts what it holds, so
	// two engines of one Mac (a world's commands) never drop each other's.
	mu      sync.Mutex
	file    ActFile
	owners  ActOwners
	now     func() time.Time
	put     func(guide.Acts) error
	project func(guide.Act) (guide.Guide, bool)
}

func OpenActs(file ActFile, owners ActOwners, now func() time.Time, put func(guide.Acts) error, project func(guide.Act) (guide.Guide, bool)) *Acts {
	// invariant: at the engine's start an act whose process no longer runs
	// ends interrupted at the step it reached; the board puts what it
	// holds only when it holds an act. project is an act's Guide as the
	// engine serves it, false for a family no projection knows.
	b := &Acts{file: file, owners: owners, now: now, put: put, project: project}
	b.mu.Lock()
	defer b.mu.Unlock()
	k := b.read()
	moved := false
	for i, e := range k.Acts {
		if e.Ended == "" && (e.Owner == nil || !owners.Runs(*e.Owner)) {
			k.Acts[i].Ended, k.Acts[i].At, k.Acts[i].Owner = guide.EndedInterrupted, b.at(), nil
			moved = true
		}
	}
	if moved {
		b.write(k)
	}
	if len(k.Acts) > 0 {
		_ = b.put(actsOf(k))
	}
	return b
}

func (b *Acts) Phase(family string, ph guide.Phase) (string, *guide.Guide) {
	// invariant: the phase that puts an act on the board answers its first
	// Guide, projected from the act as put, so the progress frame that
	// names the act carries what the board already holds.
	id := guide.ID(family, ph.Subject)
	b.mu.Lock()
	defer b.mu.Unlock()
	k := b.read()
	i := slices.IndexFunc(k.Acts, func(e actEntry) bool { return e.ID == id })
	if i >= 0 && k.Acts[i].Ended != "" {
		k.Acts, i = slices.Delete(k.Acts, i, i+1), -1
	}
	put := i < 0
	if put {
		var owner *machines.Process
		if self, err := b.owners.Self(); err == nil {
			owner = &self
		}
		k.Acts = append(k.Acts, actEntry{Act: guide.Act{ID: id, Family: family, Subject: ph.Subject}, Owner: owner})
		i = len(k.Acts) - 1
	}
	e := &k.Acts[i]
	if len(ph.Rail) > 0 {
		e.Rail = slices.Clone(ph.Rail)
	}
	e.Step, e.Line, e.At = ph.Step, ph.Line, b.at()
	act := e.Act
	b.save(k)
	if !put {
		return id, nil
	}
	if g, known := b.project(act); known {
		return id, &g
	}
	return id, nil
}

func (b *Acts) End(id string, ended guide.Ended, failed *call.WireError) {
	b.mu.Lock()
	defer b.mu.Unlock()
	k := b.read()
	i := slices.IndexFunc(k.Acts, func(e actEntry) bool { return e.ID == id && e.Ended == "" })
	if i < 0 {
		return
	}
	e := &k.Acts[i]
	e.Ended, e.Line, e.At, e.Owner = ended, "", b.at(), nil
	if failed != nil {
		e.Code, e.Cause = string(failed.Code), failed.Message
	}
	b.save(k)
}

func (b *Acts) Refused(id string, refusal *call.WireError) {
	// invariant: a repair the engine refused ends its act failed again, on
	// the refusal's code and words, from the moment it was refused; an act
	// that still runs, or that left the board, takes no refusal.
	b.mu.Lock()
	defer b.mu.Unlock()
	k := b.read()
	i := slices.IndexFunc(k.Acts, func(e actEntry) bool { return e.ID == id && e.Ended != "" })
	if i < 0 || refusal == nil {
		return
	}
	e := &k.Acts[i]
	e.Ended, e.Code, e.Cause, e.At = guide.EndedFailed, string(refusal.Code), refusal.Message, b.at()
	b.save(k)
}

func (b *Acts) at() time.Time { return b.now().UTC().Truncate(time.Second) }

func (b *Acts) save(k actsKept) {
	// invariant: an act that ended leaves the board once it lingered, as a
	// fact this write records, never as a clock a projection reads.
	k.Acts = slices.DeleteFunc(k.Acts, func(e actEntry) bool {
		return e.Ended != "" && b.now().Sub(e.At) >= ActLinger
	})
	b.write(k)
	_ = b.put(actsOf(k))
}

func (b *Acts) read() actsKept {
	var k actsKept
	data, err := b.file.Read()
	if err != nil || len(data) == 0 || json.Unmarshal(data, &k) != nil {
		return actsKept{}
	}
	return k
}

func (b *Acts) write(k actsKept) {
	data, err := json.Marshal(k)
	if err != nil {
		return
	}
	_ = b.file.Write(append(data, '\n'))
}

func actsOf(k actsKept) guide.Acts {
	out := guide.Acts{Acts: make([]guide.Act, len(k.Acts))}
	for i, e := range k.Acts {
		out.Acts[i] = e.Act
	}
	return out
}
