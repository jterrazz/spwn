package calls

import "testing"

func TestAllocateNameSkipsEveryHeldName(t *testing.T) {
	for _, c := range []struct {
		name string
		held []string
		want string
	}{
		{name: "the smallest free integer fills a gap", held: []string{"linux-1", "linux-3"}, want: "linux-2"},
		{name: "a host named like a VM holds its name", held: []string{"x", "linux-1"}, want: "linux-2"},
	} {
		t.Run(c.name, func(t *testing.T) {
			held := claims{held: map[string]claim{}}
			for _, r := range c.held {
				held.held[r] = claimVM
			}
			if got := allocateName("linux", held); got != c.want {
				t.Errorf("allocateName = %q, want %q", got, c.want)
			}
		})
	}
}
