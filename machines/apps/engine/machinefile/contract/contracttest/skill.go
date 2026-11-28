package contracttest

import (
	"context"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
)

// value: the `spwn` skill of a fresh machine.

type SkillSubject func(t *testing.T) contract.Skill

func Skill(t *testing.T, subject SkillSubject) {
	ctx := context.Background()
	t.Run("a skill never written reads as none", func(t *testing.T) {
		if got, ok, err := subject(t).Read(ctx); err != nil || ok || len(got) != 0 {
			t.Errorf("Read = %q, %v, %v; want none", got, ok, err)
		}
	})
	t.Run("a skill written is read back, and the same bytes again write nothing", func(t *testing.T) {
		s := subject(t)
		for i, step := range []struct {
			body    string
			changed bool
		}{{"# one\n", true}, {"# one\n", false}, {"# two\n", true}} {
			changed, err := s.Write(ctx, []byte(step.body))
			if err != nil || changed != step.changed {
				t.Fatalf("write %d = %v, %v; want changed %v", i, changed, err, step.changed)
			}
			if got, ok, err := s.Read(ctx); err != nil || !ok || string(got) != step.body {
				t.Fatalf("after write %d Read = %q, %v, %v", i, got, ok, err)
			}
		}
	})
}
