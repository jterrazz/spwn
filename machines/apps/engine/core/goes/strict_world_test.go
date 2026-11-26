//go:build world

package goes

import (
	"errors"
	"fmt"
	"testing"
	"testing/synctest"
)

type bug struct{ at string }

func TestTheSpecBuildDiesOnAnUnscriptedPanic(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := &record{}
		s := newSet(r, nil)
		value := bug{at: "index"}
		func() {
			defer func() {
				if v := recover(); v != value {
					t.Errorf("Try let %v through, want the unscripted value", v)
				}
			}()
			_ = s.try("call.x", func() error { panic(value) })
			t.Error("Try returned on an unscripted panic")
		}()
		go s.once("job.x", func() { panic(value) }, false)
		synctest.Wait()
		got := r.snapshot()
		if len(got.deaths) != 1 || got.deaths[0] != value {
			t.Fatalf("Go deaths = %v, want the unscripted value", got.deaths)
		}
		if len(got.lines) != 0 || s.breaks("call.x")+s.breaks("job.x") != 0 {
			t.Fatalf("an unscripted panic was reported: %q", got.lines)
		}
	})
}

func TestTheSpecBuildRecoversAScriptedPanic(t *testing.T) {
	for _, value := range []any{Script("clone"), fmt.Errorf("world fault: %w", Script("clone"))} {
		r := &record{}
		s := newSet(r, nil)
		var b *Break
		if err := s.try("call.x", func() error { panic(value) }); !errors.As(err, &b) {
			t.Fatalf("Try of a scripted %T = %v, want a break", value, err)
		}
	}
}
