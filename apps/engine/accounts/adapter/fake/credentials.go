package fake

import (
	"encoding/json"

	"github.com/jterrazz/studio-station/apps/engine/world"
)

type Credentials struct {
	// invariant: the credential store inside a VM of the world is the VM's
	// own slice's secrets: what an import writes there, a delete removes.
	w  *world.World
	vm string
}

func CredentialsOf(w *world.World, vm string) *Credentials { return &Credentials{w: w, vm: vm} }

func (c *Credentials) Import(stdin []byte) (imported bool) {
	var values map[string]string
	if json.Unmarshal(stdin, &values) != nil {
		return false
	}
	c.w.Lock()
	defer c.w.Unlock()
	m := mac(c.w, c.vm)
	if m.Secrets == nil {
		m.Secrets = map[string]string{}
	}
	for name, v := range values {
		m.Secrets[name] = v
	}
	return true
}

func (c *Credentials) Delete(names []string) {
	c.w.Lock()
	defer c.w.Unlock()
	for _, n := range names {
		delete(mac(c.w, c.vm).Secrets, n)
	}
}
