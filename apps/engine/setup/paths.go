package setup

import (
	"github.com/jterrazz/studio-station/apps/engine/machines"
)

type Roots struct{ Prefs, Data, State, Cache string }

type Paths struct {
	Prefs, Data, State, Cache, Bin string
	Roots                          Roots
}

type Mac = machines.MacRecord

type V1Preferences struct {
	// value: station's preferences as a v1 config.json converts to them, one
	// field per group.
	ThisMachine string
	Macs        map[string]Mac
	VMUser      string
	UserEmail   string
	UserName    string
}
