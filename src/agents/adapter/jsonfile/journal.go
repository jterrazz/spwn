package jsonfile

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/agents"
	"github.com/jterrazz/studio-station/src/agents/contract"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/store"
)

const Version = 1

type Files interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
	ReadDir(path string) ([]fs.DirEntry, error)
	Rename(from, to string) error
	Remove(path string) error
}

type Clock interface {
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) error
}

type Journal struct {
	// invariant: one file per session, <dir>/<session>.json, mode 0600,
	// each written whole to a staged file then renamed over the last, so
	// a reader never sees half a session.
	files Files
	dir   string
	clock Clock
	every time.Duration
}

var _ contract.Journal = (*Journal)(nil)

func New(files Files, dir string, clock Clock, every time.Duration) *Journal {
	// value: every is how often Watch looks at the folder.
	return &Journal{files: files, dir: dir, clock: clock, every: every}
}

type document struct {
	V     int          `json:"v,omitempty"`
	Agent agents.Agent `json:"agent"`
}

func (j *Journal) file(session string) string { return path.Join(j.dir, session+".json") }

func (j *Journal) all() ([]agents.Agent, error) {
	entries, err := j.files.ReadDir(j.dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, &store.UnreadableError{Path: j.dir, Cause: err}
	}
	var out []agents.Agent
	for _, e := range entries {
		session, ok := strings.CutSuffix(e.Name(), ".json")
		if !ok || e.IsDir() || !agents.PlainSession(session) {
			continue
		}
		at := j.file(session)
		data, err := j.files.ReadFile(at)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, &store.UnreadableError{Path: at, Cause: err}
		}
		var doc document
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, &store.UnreadableError{Path: at, Cause: err, Line: store.LineOf(data, err)}
		}
		if _, err := store.Current(doc.V, Version); err != nil {
			return nil, &store.UnreadableError{Path: at, Cause: err}
		}
		doc.Agent.Session = session
		out = append(out, doc.Agent)
	}
	slices.SortFunc(out, func(a, b agents.Agent) int { return strings.Compare(a.Session, b.Session) })
	return out, nil
}

func (j *Journal) Read(context.Context) ([]agents.Agent, error) {
	held, err := j.all()
	if err != nil {
		return nil, err
	}
	now := j.clock.Now()
	return slices.DeleteFunc(held, func(a agents.Agent) bool { return a.Expired(now) }), nil
}

func (j *Journal) Write(_ context.Context, a agents.Agent) error {
	if !agents.PlainSession(a.Session) {
		return errors.New("no session is named " + a.Session)
	}
	if len(a.Steps) > catalog.AgentStepsKept {
		a.Steps = slices.Clone(a.Steps[len(a.Steps)-catalog.AgentStepsKept:])
	}
	data, err := json.MarshalIndent(document{V: Version, Agent: a}, "", "  ")
	if err != nil {
		return err
	}
	if err := j.files.MkdirAll(j.dir, 0o700); err != nil {
		return err
	}
	staged := path.Join(j.dir, "."+a.Session+".json.writing")
	if err := j.files.WriteFile(staged, append(data, '\n'), 0o600); err != nil {
		return err
	}
	if err := j.files.Rename(staged, j.file(a.Session)); err != nil {
		_ = j.files.Remove(staged)
		return err
	}
	return j.forget()
}

func (j *Journal) Forget(_ context.Context, session string) error {
	if !agents.PlainSession(session) {
		return nil
	}
	if err := j.files.Remove(j.file(session)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func (j *Journal) forget() error {
	held, err := j.all()
	if err != nil {
		return err
	}
	now := j.clock.Now()
	for _, a := range held {
		if a.Expired(now) {
			if err := j.files.Remove(j.file(a.Session)); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return err
			}
		}
	}
	return nil
}

func (j *Journal) Watch(ctx context.Context) (<-chan []agents.Agent, error) {
	if _, err := j.Read(ctx); err != nil {
		return nil, err
	}
	out := make(chan []agents.Agent)
	go func() {
		defer close(out)
		var last []byte
		for {
			if now, err := j.Read(ctx); err == nil {
				if data, _ := json.Marshal(now); last == nil || !bytes.Equal(data, last) {
					select {
					case out <- now:
						last = data
					case <-ctx.Done():
						return
					}
				}
			}
			if j.clock.Sleep(ctx, j.every) != nil {
				return
			}
		}
	}()
	return out, nil
}
