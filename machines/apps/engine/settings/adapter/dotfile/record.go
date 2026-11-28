package dotfile

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

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

func (a *Applier) read() (record, error) {
	data, err := a.files.ReadFile(a.o.Record)
	if errors.Is(err, fs.ErrNotExist) {
		return record{}, nil
	}
	if err != nil {
		return record{}, &store.UnreadableError{Path: a.o.Record, Cause: err}
	}
	var r record
	if err := json.Unmarshal(data, &r); err != nil {
		return record{}, &store.UnreadableError{Path: a.o.Record, Cause: err}
	}
	if err := store.Current(r.V, recordVersion); err != nil {
		return record{}, &store.UnreadableError{Path: a.o.Record, Cause: err}
	}
	return r, nil
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
func (a *Applier) locked(edit func(r *record) error) error {
	if err := a.files.MkdirAll(path.Dir(a.o.Record), 0o700); err != nil {
		return &store.UnwritableError{Path: a.o.Record, Cause: err}
	}
	release, err := a.files.Lock(a.o.Record + ".lock")
	if err != nil {
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
