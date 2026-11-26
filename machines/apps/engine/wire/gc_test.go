package wire

import "testing"

func TestOnlyAShortRunTurnsTheCollectorOff(t *testing.T) {
	for _, c := range []struct {
		args  []string
		vars  map[string]string
		short bool
	}{
		{args: []string{"--version"}, short: true},
		{args: nil, short: true},
		{args: []string{"machines", "--json"}, short: true},
		{args: []string{"shell", "prompt"}, short: true},
		{args: []string{"engine", "stop"}, short: true},
		{args: []string{"engine", "run"}},
		{args: []string{"engine", "attach"}},
		{args: []string{"guest", "serve", "--stdio"}},
		{args: []string{"--version"}, vars: map[string]string{"GOGC": "100"}},
		{args: []string{"--version"}, vars: map[string]string{"GOMEMLIMIT": "1GiB"}},
	} {
		if got := shortLived(c.args, c.vars); got != c.short {
			t.Errorf("station %v with %v: short-lived %v, want %v", c.args, c.vars, got, c.short)
		}
	}
}
