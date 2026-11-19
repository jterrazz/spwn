package jsonfile

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/journal"
	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/journal/contract"
)

const (
	Schema  = "station.journal"
	Version = 1
)

type Files interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
	Rename(from, to string) error
	// invariant: an exclusive lock on path, made when missing, held until
	// release; a second holder waits for it.
	Lock(path string) (release func(), err error)
}

type Journal struct {
	// value: the journal, one file: `~/.local/state/station/journal.jsonl`,
	// its first line `{"schema": "station.journal/v1"}`, then one line per
	// line kept, beside the lock its writers take, `journal.jsonl.lock`.
	files Files
	at    string
}

var (
	_ contract.Journal = (*Journal)(nil)
)

func New(files Files, at string) *Journal { return &Journal{files: files, at: at} }

type head struct {
	Schema string `json:"schema"`
}

func (j *Journal) read() ([]journal.Line, error) {
	data, err := j.files.ReadFile(j.at)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, &store.UnreadableError{Path: j.at, Cause: err}
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	var out []journal.Line
	n := 0
	for sc.Scan() {
		n++
		text := sc.Bytes()
		if len(bytes.TrimSpace(text)) == 0 {
			continue
		}
		if n == 1 {
			var h head
			if err := json.Unmarshal(text, &h); err != nil {
				return nil, &store.UnreadableError{Path: j.at, Cause: err, Line: n}
			}
			_, version, _ := strings.Cut(h.Schema, "/v")
			v, _ := strconv.Atoi(version)
			if err := store.Current(v, Version); err != nil {
				return nil, &store.UnreadableError{Path: j.at, Cause: err}
			}
			continue
		}
		var l journal.Line
		if err := json.Unmarshal(text, &l); err != nil {
			return nil, &store.UnreadableError{Path: j.at, Cause: err, Line: n}
		}
		out = append(out, l)
	}
	if err := sc.Err(); err != nil {
		return nil, &store.UnreadableError{Path: j.at, Cause: err}
	}
	return out, nil
}

func (j *Journal) Lines(_ context.Context, since, before time.Time) ([]journal.Line, error) {
	lines, err := j.read()
	if err != nil {
		return nil, err
	}
	return journal.Between(lines, since, before), nil
}

func (j *Journal) Write(_ context.Context, now time.Time, lines ...journal.Line) error {
	// invariant: a write happens under the journal's lock, read then written
	// whole to a temporary file renamed over it, so two writers at once (the
	// engine's night and a gesture) never lose each other's line, and a
	// reader never sees half a file.
	if err := j.files.MkdirAll(path.Dir(j.at), 0o700); err != nil {
		return &store.UnwritableError{Path: j.at, Cause: err}
	}
	release, err := j.files.Lock(j.at + ".lock")
	if err != nil {
		return &store.UnwritableError{Path: j.at, Cause: err}
	}
	defer release()
	kept, err := j.read()
	if err != nil {
		return err
	}
	kept = journal.Kept(append(kept, lines...), now)
	var b bytes.Buffer
	first, _ := json.Marshal(head{Schema: Schema + "/v" + strconv.Itoa(Version)})
	b.Write(first)
	b.WriteByte('\n')
	for i := len(kept) - 1; i >= 0; i-- {
		data, err := json.Marshal(kept[i])
		if err != nil {
			return err
		}
		b.Write(data)
		b.WriteByte('\n')
	}
	tmp := j.at + ".tmp"
	if err := j.files.WriteFile(tmp, b.Bytes(), 0o600); err != nil {
		return &store.UnwritableError{Path: j.at, Cause: err}
	}
	if err := j.files.Rename(tmp, j.at); err != nil {
		return &store.UnwritableError{Path: j.at, Cause: err}
	}
	return nil
}
