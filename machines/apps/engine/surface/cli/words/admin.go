package words

import "github.com/jterrazz/studio-station/apps/engine/catalog"

func AdminApply(setting, mac string, thisMac bool) string {
	// value: the command that applies an admin setting with admin rights, on
	// its Mac's own terminal: station never runs sudo itself, from any
	// surface.
	command := catalog.StationAsRoot("settings apply " + setting)
	if thisMac {
		return command
	}
	return catalog.OnMac(mac, command)
}
