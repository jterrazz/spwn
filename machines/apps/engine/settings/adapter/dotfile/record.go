package dotfile

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/store"
)

const recordVersion = 1

// value: Machines' record of every fragment it wrote into a file of the
// owner's, one file beside the ledger: by setting and file, what it made
// (the file, its folders), the digest of a block's text, and the JSON
// pointer and digest of each value a merge wrote. Off takes back only
// what it holds.

type record struct {
	V     int       `json:"v"`
	Files []written `json:"files,omitempty"`
}

type written struct {
	Setting string `json:"setting"`
	// value: the file as its package names it, `~/.claude/settings.json`.
	Path string `json:"path"`
	Made bool   `json:"made,omitempty"`
	// value: the folders made for it, deepest first.
	Folders []string `json:"folders,omitempty"`
	Block   string   `json:"block,omitempty"`
	Values  []entry  `json:"values,omitempty"`
}

// value: one thing a merge wrote: a value at its pointer, an element of
// the array at its pointer, or an object or an array it made there.

type entry struct {
	Pointer string `json:"pointer"`
	Kind    kind   `json:"kind"`
	Digest  string `json:"digest,omitempty"`
}

type kind string

const (
	kindValue   kind = "value"
	kindElement kind = "element"
	kindObject  kind = "object"
	kindArray   kind = "array"
)

func (r *record) of(setting, file string) (w written, at int) {
	i := slices.IndexFunc(r.Files, func(f written) bool { return f.Setting == setting && f.Path == file })
	if i < 0 {
		return written{Setting: setting, Path: file}, -1
	}
	return r.Files[i], i
}

func (r *record) put(w written) {
	if _, i := r.of(w.Setting, w.Path); i >= 0 {
		r.Files[i] = w
		return
	}
	r.Files = append(r.Files, w)
}

func (r *record) drop(setting, file string) {
	r.Files = slices.DeleteFunc(r.Files, func(w written) bool { return w.Setting == setting && w.Path == file })
}

// value: the bound of the record, past which it is no record Machines wrote.
const maxRecord = 1 << 20

// value: how long a change of a fragment waits for the lock, as the
// authorized keys' writer does.
const lockWait = 10 * time.Second

func (a *Applier) read() (record, error) {
	info, err := a.files.Stat(a.o.Record)
	if errors.Is(err, fs.ErrNotExist) {
		return record{}, nil
	}
	if err != nil {
		return record{}, &store.UnreadableError{Path: a.o.Record, Cause: err}
	}
	if info.Size() > maxRecord {
		return record{}, a.untrusted(errors.New("it is past 1 MiB"))
	}
	data, err := a.files.ReadFile(a.o.Record)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return record{}, nil
	case err != nil:
		return record{}, &store.UnreadableError{Path: a.o.Record, Cause: err}
	case len(data) > maxRecord:
		return record{}, a.untrusted(errors.New("it is past 1 MiB"))
	}
	var r record
	if err := json.Unmarshal(data, &r); err != nil {
		return record{}, &store.UnreadableError{Path: a.o.Record, Cause: err}
	}
	if err := store.Current(r.V, recordVersion); err != nil {
		return record{}, &store.UnreadableError{Path: a.o.Record, Cause: err}
	}
	if err := r.trusted(); err != nil {
		return record{}, a.untrusted(err)
	}
	return r, nil
}

func (a *Applier) untrusted(why error) error {
	return &store.UnreadableError{Path: a.o.Record, Cause: fmt.Errorf("%w: Machines takes nothing back by a record it cannot trust; mend the entry, or delete the file and off takes back nothing", why)}
}

// value: a JSON pointer as a merge writes one: each key under `/`, its
// `~` escaped as `~0` or `~1` alone.
var pointer = regexp.MustCompile(`^(/([^~/]|~[01])*)+$`)

var digest = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// invariant: a record is trusted whole or not at all: each file names its
// setting and a path a setting may write, its block's digest and each
// folder are of the form Machines writes, and each entry is of a kind it
// writes, at a pointer below the file's top, with a digest exactly when
// it is a value or an element. One entry that is not refuses every
// take-back, so off never deletes by half a record it cannot read.
func (r record) trusted() error {
	for _, w := range r.Files {
		switch {
		case w.Setting == "":
			return fmt.Errorf("an entry for %s names no setting", w.Path)
		case catalog.Confined(w.Path, catalog.WriteFile) != "":
			return fmt.Errorf("the entry of %s names a file no setting writes: %s", w.Setting, catalog.Confined(w.Path, catalog.WriteFile))
		case w.Block != "" && !digest.MatchString(w.Block):
			return fmt.Errorf("the entry of %s for %s holds the block digest %q, of no form Machines writes", w.Setting, w.Path, w.Block)
		}
		for _, dir := range w.Folders {
			if !path.IsAbs(dir) || path.Clean(dir) != dir {
				return fmt.Errorf("the entry of %s for %s names the folder %q, no path Machines makes", w.Setting, w.Path, dir)
			}
		}
		for i, e := range w.Values {
			if problem := e.untrusted(); problem != "" {
				return fmt.Errorf("value %d of %s for %s %s", i+1, w.Path, w.Setting, problem)
			}
		}
	}
	return nil
}

func (e entry) untrusted() string {
	if !pointer.MatchString(e.Pointer) {
		return fmt.Sprintf("is at %q, no JSON pointer below the file's top", e.Pointer)
	}
	switch e.Kind {
	case kindValue, kindElement:
		if !digest.MatchString(e.Digest) {
			return fmt.Sprintf("holds the digest %q, of no form Machines writes", e.Digest)
		}
	case kindObject, kindArray:
		if e.Digest != "" {
			return "is a made " + string(e.Kind) + " with a digest, which Machines never writes"
		}
	default:
		return fmt.Sprintf("is of the kind %q, none Machines writes", e.Kind)
	}
	return ""
}

// invariant: a record with no file left is no record: its file is removed.
func (a *Applier) save(r record) error {
	if len(r.Files) == 0 {
		err := a.files.Remove(a.o.Record)
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	slices.SortFunc(r.Files, func(x, y written) int {
		return strings.Compare(x.Setting+"\x00"+x.Path, y.Setting+"\x00"+y.Path)
	})
	r.V = recordVersion
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	if err := a.files.MkdirAll(path.Dir(a.o.Record), 0o700); err != nil {
		return &store.UnwritableError{Path: a.o.Record, Cause: err}
	}
	if err := a.files.Replace(a.o.Record, append(data, '\n'), 0o600); err != nil {
		return &store.UnwritableError{Path: a.o.Record, Cause: err}
	}
	return nil
}

// invariant: every change of a fragment, from its file's read to its
// record's write, holds the one lock beside the record, so two settings
// writing into one file at once never drop each other's values.
func (a *Applier) locked(ctx context.Context, edit func(r *record) error) error {
	if err := a.files.MkdirAll(path.Dir(a.o.Record), 0o700); err != nil {
		return &store.UnwritableError{Path: a.o.Record, Cause: err}
	}
	// invariant: the lock is waited for at most lockWait: one held longer
	// by another run is refused, nothing touched, never waited for forever.
	held, cancel := context.WithTimeout(ctx, lockWait)
	defer cancel()
	release, err := a.files.Hold(held, a.o.Record+".lock")
	switch {
	case errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil:
		return fmt.Errorf("%s.lock is held by another run writing a fragment, nothing touched: apply again once it ends", a.o.Record)
	case err != nil:
		return fmt.Errorf("%s.lock: %w", a.o.Record, err)
	}
	defer release()
	r, err := a.read()
	if err != nil {
		return err
	}
	before, _ := json.Marshal(r)
	if err := edit(&r); err != nil {
		return errors.Join(err, a.saveIfMoved(before, r))
	}
	return a.saveIfMoved(before, r)
}

func (a *Applier) saveIfMoved(before []byte, r record) error {
	if after, _ := json.Marshal(r); bytes.Equal(after, before) {
		return nil
	}
	return a.save(r)
}
