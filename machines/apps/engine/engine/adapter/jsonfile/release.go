package jsonfile

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"path"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/engine/contract"
)

const ReleaseVersion = 1

type Files interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
	Rename(from, to string) error
}

// value: the engine's last asks of the latest release, one file:
// `~/.cache/station/release.json`.

type ReleaseCache struct {
	files Files
	at    string
}

var _ contract.ReleaseCache = (*ReleaseCache)(nil)

func NewReleaseCache(files Files, at string) *ReleaseCache {
	return &ReleaseCache{files: files, at: at}
}

type releaseFile struct {
	V       int            `json:"v"`
	Read    time.Time      `json:"read,omitzero"`
	Latest  *latest        `json:"latest,omitempty"`
	Refused *refusedAnswer `json:"refused,omitempty"`
}

type latest struct {
	Tag string `json:"tag"`
	URL string `json:"url"`
}

type refusedAnswer struct {
	At      time.Time `json:"at"`
	Message string    `json:"message"`
}

func (c *ReleaseCache) Read(context.Context) (engine.ReleaseKept, bool, error) {
	data, err := c.files.ReadFile(c.at)
	if errors.Is(err, fs.ErrNotExist) {
		return engine.ReleaseKept{}, false, nil
	}
	if err != nil {
		return engine.ReleaseKept{}, false, &store.UnreadableError{Path: c.at, Cause: err}
	}
	var f releaseFile
	if err = json.Unmarshal(data, &f); err != nil {
		return engine.ReleaseKept{}, false, &store.UnreadableError{Path: c.at, Cause: err}
	}
	if err = store.Current(f.V, ReleaseVersion); err != nil {
		return engine.ReleaseKept{}, false, &store.UnreadableError{Path: c.at, Cause: err}
	}
	k := engine.ReleaseKept{Read: f.Read}
	if f.Latest != nil {
		k.Latest = &engine.Tagged{Tag: f.Latest.Tag, URL: f.Latest.URL}
	}
	if f.Refused != nil {
		k.Refused = &engine.ReleaseRefusal{At: f.Refused.At, Message: f.Refused.Message}
	}
	return k, true, nil
}

func (c *ReleaseCache) Write(_ context.Context, k engine.ReleaseKept) error {
	// invariant: the file is written whole to a temporary file renamed over
	// it, so a reader never sees half of one.
	f := releaseFile{V: ReleaseVersion, Read: k.Read}
	if k.Latest != nil {
		f.Latest = &latest{Tag: k.Latest.Tag, URL: k.Latest.URL}
	}
	if k.Refused != nil {
		f.Refused = &refusedAnswer{At: k.Refused.At, Message: k.Refused.Message}
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	if err := c.files.MkdirAll(path.Dir(c.at), 0o755); err != nil {
		return &store.UnwritableError{Path: c.at, Cause: err}
	}
	tmp := c.at + ".tmp"
	if err := c.files.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return &store.UnwritableError{Path: c.at, Cause: err}
	}
	if err := c.files.Rename(tmp, c.at); err != nil {
		return &store.UnwritableError{Path: c.at, Cause: err}
	}
	return nil
}
