package calls

import (
	"context"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
)

// value: an engine whose guest answers every line at once, the deadline of
// each line kept.
type boundEngine struct {
	Engine
	until []time.Time
}

func (e *boundEngine) GuestExec(ctx context.Context, _, _, _ string) (GuestRan, error) {
	until, _ := ctx.Deadline()
	e.until = append(e.until, until)
	return GuestRan{}, nil
}

func TestABootPhaseOutlastsItsItemsAndItsLines(t *testing.T) {
	// invariant: the boot phase's apply is given the items' share and the
	// lines' bound together, so lines that run long end as a broken
	// run.boot step inside it, never as the apply cut short.
	e := &boundEngine{}
	s := &Machinist{engineAPI: e}
	want := catalog.ItemsBound + catalog.RunBound
	ref, cancel := context.WithTimeout(context.Background(), want)
	defer cancel()
	due, _ := ref.Deadline()
	if err := s.applyBoot(context.Background(), "web", model.Template{Name: "studio-linux"}); err != nil {
		t.Fatal(err)
	}
	if off := e.until[0].Sub(due); off < 0 || off > time.Minute {
		left := e.until[0].Sub(due) + want
		t.Fatalf("the boot phase had %v; want %v, the items' share and the lines' bound", left, want)
	}
}
