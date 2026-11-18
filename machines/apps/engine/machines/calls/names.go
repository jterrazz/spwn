package calls

import (
	"maps"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
)

type claim int

const (
	unclaimed claim = iota
	claimMac
	claimVM
)

// value: every name a new VM may not take: each Mac's and VM's of the fleet,
// with the Mac a VM is on.
type claims struct {
	held map[string]claim
	on   map[string]string
}

// invariant: a name is held by a VM record on any host of the fleet or by a
// host; no tailnet node holds a VM's name since VMs left the tailnet.
func (c claims) free(name string) bool { return c.held[name] == unclaimed }

func (c claims) taken(name string) *model.NameTakenError {
	return &model.NameTakenError{Name: name, VM: c.held[name] == claimVM, Mac: c.held[name] == claimMac, On: c.on[name]}
}

// value: the claims with each VM This Mac records, on mac.
func (c claims) with(mac string, records []model.Record) claims {
	out := claims{held: maps.Clone(c.held), on: maps.Clone(c.on)}
	if out.held == nil {
		out.held, out.on = map[string]claim{}, map[string]string{}
	}
	for _, r := range records {
		out.held[r.Name], out.on[r.Name] = claimVM, mac
	}
	return out
}

func allocateName(image string, c claims) string { return model.FirstFree(image, c.free) }
