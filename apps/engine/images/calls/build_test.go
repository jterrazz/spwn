package calls

import (
	"context"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/images"
)

// value: a build VM that takes every file and answers every line at once,
// the deadline of each line kept.
type boundVM struct{ until []time.Time }

func (*boundVM) Put(context.Context, Login, string, []byte, uint32) error { return nil }
func (*boundVM) Close() error                                             { return nil }
func (v *boundVM) Run(ctx context.Context, _ Login, _, _ string) (Ran, error) {
	until, _ := ctx.Deadline()
	v.until = append(v.until, until)
	return Ran{}, nil
}

func TestTheImagePhaseOutlastsItsItemsAndItsLines(t *testing.T) {
	// invariant: the bake is given the items' share and the run.image
	// lines' bound together, as a first start's boot phase is.
	vm := &boundVM{}
	s := &Builder{o: BuilderOptions{User: "admin"}}
	want := catalog.ItemsBound + catalog.RunBound
	ref, cancel := context.WithTimeout(context.Background(), want)
	defer cancel()
	due, _ := ref.Deadline()
	if _, err := s.bakeRun(context.Background(), vm, catalog.Image{Bake: "images/linux-bake.sh"}, images.Template{Name: "studio-linux"}, Login{User: "admin"}); err != nil {
		t.Fatal(err)
	}
	if off := vm.until[0].Sub(due); off < 0 || off > time.Minute {
		left := vm.until[0].Sub(due) + want
		t.Fatalf("the bake had %v; want %v, the items' share and the lines' bound", left, want)
	}
}
