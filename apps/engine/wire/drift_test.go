package wire

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	machinefilecalls "github.com/jterrazz/studio-station/apps/engine/machinefile/calls"
)

func TestAVMReadsUnboundOnlyWhenNoFileIsNamedAfterItAndNoneIsBound(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"the VM's own name, named by nothing", fail.New(fail.NotFound, machinefilecalls.NotFound, "linux-1", map[string]string{"by": ""}), true},
		{"an extends no source holds", fail.New(fail.NotFound, machinefilecalls.NotFound, "no-such-file", map[string]string{"by": "linux-dev"}), false},
		{"its own name, as another file's extends", fail.New(fail.NotFound, machinefilecalls.NotFound, "linux-1", map[string]string{"by": "linux-dev"}), false},
		{"a bound file gone", fail.New(fail.NotFound, machinefilecalls.NotFound, "linux-dev", map[string]string{"by": ""}), false},
		{"a path that does not read", fail.New(fail.NotFound, machinefilecalls.NotFound, "linux-1", map[string]string{"path": "yes"}), false},
		{"another refusal", fail.New(fail.Refused, machinefilecalls.Invalid, "linux-1", nil), false},
	}
	for _, c := range cases {
		if got := unbound(c.err, "linux-1"); got != c.want {
			t.Errorf("%s: unbound %v, want %v", c.name, got, c.want)
		}
	}
}
