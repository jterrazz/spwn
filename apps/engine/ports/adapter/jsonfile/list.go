package jsonfile

import (
	"encoding/json"
	"errors"
	"io/fs"
	"path"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/store"
)

type Files interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
	Rename(from, to string) error
}

// invariant: a list file is {"schema": "<name>/v<N>", "<key>": [...]}: the
// schema is its version field, and a file with no schema is at version 1.
type listFile[T any] struct {
	files   Files
	path    string
	schema  string
	key     string
	version int
}

func (l listFile[T]) load() ([]T, error) {
	data, err := l.files.ReadFile(l.path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	var schema string
	if raw, ok := doc["schema"]; ok {
		if err := json.Unmarshal(raw, &schema); err != nil {
			return nil, &store.UnreadableError{Path: l.path, Cause: err}
		}
	}
	if _, err := store.Current(versionOf(schema), l.version); err != nil {
		return nil, &store.UnreadableError{Path: l.path, Cause: err}
	}
	var items []T
	if raw, ok := doc[l.key]; ok {
		if err := json.Unmarshal(raw, &items); err != nil {
			return nil, err
		}
	}
	return items, nil
}

func versionOf(schema string) int {
	_, v, ok := strings.Cut(schema, "/v")
	if !ok {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return n
}

// invariant: the file is written whole to a temporary file then renamed, so a
// reader (the CLI while the engine writes) never sees half a file.
func (l listFile[T]) save(items []T) error {
	if items == nil {
		items = []T{}
	}
	body, err := json.MarshalIndent(items, "  ", "  ")
	if err != nil {
		return err
	}
	data := []byte("{\n  \"schema\": " + string(quote(l.schema)) + ",\n  " + string(quote(l.key)) + ": " + string(body) + "\n}")
	if err := l.files.MkdirAll(path.Dir(l.path), 0o700); err != nil {
		return err
	}
	tmp := l.path + ".tmp"
	if err := l.files.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return l.files.Rename(tmp, l.path)
}

func quote(s string) json.RawMessage {
	raw, _ := json.Marshal(s)
	return raw
}
