package calls_test

import (
	"bytes"
	"context"
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"
)

// value: the longest one answer may take to decode and be viewed; past it
// the read is no longer total.
const answerBudget = 10 * time.Second

type viewed struct {
	out     machinescalls.ListOut
	refused string
	views   []machines.MacView
}

// invariant: what a far Mac's answer to machines.list becomes here: decoded
// as the Router decodes it, then each Mac it names viewed as This Mac's.
func viewAnswer(raw []byte) viewed {
	var v viewed
	if err := call.Decode(raw, &v.out); err != nil {
		v.refused = err.Error()
		return v
	}
	for _, mac := range v.out.All.Macs {
		v.views = append(v.views, machines.ViewOf("mac-mini", machines.MacRecord{}, machines.Node{}, mac, true))
	}
	return v
}

func FuzzDecodeFarListing(f *testing.F) {
	answer := machinescalls.ListOut{All: machines.AllMachines{Macs: []machines.MacView{{
		Name: "mac-mini", Reachable: true, LAN: "192.168.1.20",
		VMs: []machines.VM{{Name: "linux-1", State: machines.Running}, {Name: "api", State: machines.Stopped}},
	}}}, Roomiest: "mac-mini"}
	seed, err := json.Marshal(answer)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(seed)
	for _, raw := range []string{
		`{"All":{"Macs":[{"Name":"mac-mini","VMs":[{"Name":"linux-1","State":""},{"State":"running"},{"Name":"x","State":"   "}]}]}}`,
		`{"All":{"Macs":[{"Name":"mac-mini","VMs":[{"Name":"linux-1","State":7}]}]}}`,
		`{"All":{"Macs":[{"Name":"mac-mini","VMs":[{"Name":"linux-1"}]}]}}`,
		`{"All":{"Macs":[null,{}],"Refused":[{}]},"Rooms":[{}]}`,
		`{"All":{"Macs":[{"Name":"mac-mini","Answered":"not a time"}]}}`,
		`{"All":{"Macs":{"Name":"mac-mini"}}}`,
		`[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]`,
		`null`,
		``,
	} {
		f.Add([]byte(raw))
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		var first, second viewed
		ctx, cancel := context.WithTimeout(t.Context(), answerBudget)
		defer cancel()
		done := make(chan struct{})
		go func() {
			defer close(done)
			first, second = viewAnswer(raw), viewAnswer(raw)
		}()
		select {
		case <-done:
		case <-ctx.Done():
			t.Fatalf("viewing %q took more than %v", raw, answerBudget)
		}
		once, err := json.Marshal([]any{first.out, first.refused, first.views})
		if err != nil {
			t.Fatal(err)
		}
		twice, err := json.Marshal([]any{second.out, second.refused, second.views})
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(once, twice) {
			t.Fatalf("viewing %q twice answered\n%s\nthen\n%s", raw, once, twice)
		}
		if first.refused != "" {
			return
		}
		for _, mac := range first.out.All.Macs {
			for _, vm := range mac.VMs {
				if !slices.Contains(vm.State.Enum(), string(vm.State)) {
					t.Errorf("viewing %q left %s in the state %q, off the enum", raw, vm.Name, vm.State)
				}
			}
		}
	})
}
