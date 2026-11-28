package calls

import (
	"errors"
	"testing"
)

func TestASkillNotWrittenAlwaysSaysWhy(t *testing.T) {
	for _, c := range []struct{ err, want string }{
		{"open ~/.agents/skills/spwn/SKILL.md: permission denied\nmore", "open ~/.agents/skills/spwn/SKILL.md: permission denied"},
		{"\n  \n the folder is full\n", "the folder is full"},
		{"", unsaidWhy},
		{" \n\t\n", unsaidWhy},
	} {
		if got := why(errors.New(c.err)); got != c.want {
			t.Errorf("why(%q) = %q, want %q", c.err, got, c.want)
		}
	}
}
