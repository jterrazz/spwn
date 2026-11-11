package wire

import (
	"github.com/jterrazz/studio-station/src/core/call"
	coremachine "github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/engine/process"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

// value: another Mac of the spec world as its own binary builds it there:
// the registry its engine serves, its profile, and its engine.
type farMac struct {
	registry *call.Registry
	profile  coremachine.Profile
	start    func() *process.Engine
	// value: its own Router, which reaches the VMs it holds.
	relay served
}

// invariant: a far Mac is built over its own slice of the world, with no
// terminal: what it asks, it asks of the call's input, as over ssh.
func (m machine) farMac(env kit.Env, w *worldMac) farMac {
	there := resolve(m.raw, m.door, m.vars, w, m.cat)
	fenv := kit.Env{Out: env.Out, On: &kit.On{ThisMachine: there.thisMac()}, Self: there.profile.Ref}
	r := there.router(fenv)
	fenv.Calls = r
	return farMac{registry: there.registry(fenv), profile: there.profile, start: func() *process.Engine { return there.engine(fenv) }, relay: served{r}}
}
