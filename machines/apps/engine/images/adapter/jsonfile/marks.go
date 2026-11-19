package jsonfile

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/images"
	"github.com/jterrazz/studio-station/apps/engine/images/contract"
)

type Marks struct {
	files Files
	dir   string
}

var _ contract.BuildMarks = (*Marks)(nil)

func NewMarks(files Files, dir string) *Marks { return &Marks{files: files, dir: dir} }

func (b *Marks) file(image string) string { return path.Join(b.dir, image+".json") }

func (b *Marks) Mark(_ context.Context, m images.Making) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	if err := b.files.MkdirAll(b.dir, 0o700); err != nil {
		return err
	}
	return b.files.WriteFile(b.file(m.Image), data, 0o600)
}

func (b *Marks) Clear(_ context.Context, image string) error {
	if err := b.files.Remove(b.file(image)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func (b *Marks) Marked(context.Context) ([]images.Making, error) {
	entries, err := b.files.ReadDir(b.dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []images.Making
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := b.files.ReadFile(path.Join(b.dir, e.Name()))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		var m images.Making
		if err := json.Unmarshal(data, &m); err != nil {
			continue
		}
		out = append(out, m)
	}
	slices.SortFunc(out, func(a, c images.Making) int { return strings.Compare(a.Image, c.Image) })
	return out, nil
}
