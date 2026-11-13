package process

import (
	"maps"

	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

func (d *Engine) KeepSettings(vm string, read settings.LastRead) {
	// invariant: the last settings read of each workspace, held in this
	// engine's memory alone, for the matrix to show while it is stopped.
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.settings == nil {
		d.settings = engine.SettingsKept{}
	}
	d.settings[vm] = read
}

func (d *Engine) LastSettings(vm string) (settings.LastRead, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	read, ok := d.settings[vm]
	return read, ok
}

func (d *Engine) SettingsKept() engine.SettingsKept {
	d.mu.Lock()
	defer d.mu.Unlock()
	return maps.Clone(d.settings)
}

func (d *Engine) RememberSettings(k engine.SettingsKept) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.settings = maps.Clone(k)
}
