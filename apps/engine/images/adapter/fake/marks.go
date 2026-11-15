package fake

import (
	"context"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/images"
	"github.com/jterrazz/studio-station/apps/engine/images/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

type Marks struct {
	// invariant: the builds one Mac of the world marks.
	w   *world.World
	mac string
}

var _ contract.BuildMarks = (*Marks)(nil)

func MarksOf(w *world.World, mac string) *Marks { return &Marks{w: w, mac: mac} }

func NewMarks() *Marks {
	w := &world.World{ThisMac: "this-mac"}
	w.Mac("this-mac")
	return MarksOf(w, "this-mac")
}

func (b *Marks) Mark(_ context.Context, m images.Making) error {
	b.w.Lock()
	defer b.w.Unlock()
	marks := &b.w.Mac(b.mac).Building
	*marks = append(slices.DeleteFunc(*marks, func(o images.Making) bool { return o.Image == m.Image }), m)
	return nil
}

func (b *Marks) Clear(_ context.Context, image string) error {
	b.w.Lock()
	defer b.w.Unlock()
	marks := &b.w.Mac(b.mac).Building
	*marks = slices.DeleteFunc(*marks, func(o images.Making) bool { return o.Image == image })
	return nil
}

func (b *Marks) Marked(context.Context) ([]images.Making, error) {
	b.w.Lock()
	defer b.w.Unlock()
	return slices.Clone(b.w.Mac(b.mac).Building), nil
}
