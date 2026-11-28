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

// value: the names a new VM of This Mac may not take, each Mac's and each
// VM's This Mac holds, with the Mac a VM is on; and the names VMs of other
// Macs bear, which an asked name may take and an unasked one never does.
type claims struct {
	held      map[string]claim
	on        map[string]string
	elsewhere map[string]bool
}

// invariant: a VM's name is held on its own computer (ADR-109 §2): a name
// is taken by a host or by a VM record of This Mac, never by a VM of
// another Mac, whose full address `<vm>.<computer>` stays apart; no tailnet
// node holds a VM's name since VMs left the tailnet.
func (c claims) free(name string) bool { return c.held[name] == unclaimed }

// invariant: a name chosen for a create that asked none makes no bare name
// ambiguous: it is free here and borne by no VM of another Mac.
func (c claims) unused(name string) bool { return c.free(name) && !c.elsewhere[name] }

func (c claims) taken(name string) *model.NameTakenError {
	return &model.NameTakenError{Name: name, VM: c.held[name] == claimVM, Mac: c.held[name] == claimMac, On: c.on[name]}
}

// value: the claims with each VM This Mac records, on mac.
func (c claims) with(mac string, records []model.Record) claims {
	out := claims{held: maps.Clone(c.held), on: maps.Clone(c.on), elsewhere: c.elsewhere}
	if out.held == nil {
		out.held, out.on = map[string]claim{}, map[string]string{}
	}
	for _, r := range records {
		out.held[r.Name], out.on[r.Name] = claimVM, mac
	}
	return out
}

func allocateName(image string, c claims) string { return model.FirstFree(image, c.unused) }
