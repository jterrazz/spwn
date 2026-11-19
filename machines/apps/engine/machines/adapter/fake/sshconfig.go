package fake

import (
	"context"
	"errors"
	"maps"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

type SSHConfig struct {
	// invariant: the ssh client config of one Mac of the world: its managed
	// blocks, and the hosts the owner defined outside them.
	w   *world.World
	mac string
}

func SSHConfigOf(w *world.World, mac string) *SSHConfig { return &SSHConfig{w: w, mac: mac} }

func NewSSHConfig() *SSHConfig {
	w := &world.World{ThisMac: "this-mac"}
	w.Mac("this-mac")
	return SSHConfigOf(w, "this-mac")
}

func (c *SSHConfig) blocks() *world.SSH {
	m := c.w.Mac(c.mac)
	if m.SSH == nil {
		m.SSH = &world.SSH{}
	}
	if m.SSH.Blocks == nil {
		m.SSH.Blocks = map[string]world.SSHBlock{}
	}
	return m.SSH
}

func (c *SSHConfig) Foreign(name string) {
	c.w.Lock()
	defer c.w.Unlock()
	cfg := c.blocks()
	cfg.Foreign = append(cfg.Foreign, name)
}

func (c *SSHConfig) Managed() []string {
	c.w.Lock()
	defer c.w.Unlock()
	return slices.Sorted(maps.Keys(c.blocks().Blocks))
}

func (c *SSHConfig) put(blocks []contract.MachineBlock) error {
	cfg := c.blocks()
	var refused error
	for _, b := range blocks {
		if slices.Contains(cfg.Foreign, b.Name) {
			if refused == nil {
				refused = &contract.SSHConfigError{Path: "~/.ssh/config", Cause: errors.New("a block of yours already names " + b.Name + " (outside any managed block)")}
			}
			continue
		}
		cfg.Blocks[b.Name] = world.SSHBlock{User: b.User, Key: b.Key, Behind: b.Behind, ThisMac: b.ThisMac}
	}
	return refused
}

func (c *SSHConfig) Put(_ context.Context, blocks []contract.MachineBlock) error {
	c.w.Lock()
	defer c.w.Unlock()
	return c.put(blocks)
}

func (c *SSHConfig) Replace(_ context.Context, blocks []contract.MachineBlock) error {
	c.w.Lock()
	defer c.w.Unlock()
	refused := c.put(blocks)
	maps.DeleteFunc(c.blocks().Blocks, func(name string, _ world.SSHBlock) bool {
		return !slices.ContainsFunc(blocks, func(b contract.MachineBlock) bool { return b.Name == name })
	})
	return refused
}

func (c *SSHConfig) Remove(_ context.Context, names ...string) error {
	c.w.Lock()
	defer c.w.Unlock()
	for _, name := range names {
		delete(c.blocks().Blocks, name)
	}
	return nil
}
