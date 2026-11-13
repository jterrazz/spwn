package fake

import "github.com/jterrazz/studio-station/apps/engine/world"

const thisMac = "this-mac"

func alone() *world.World {
	w := &world.World{ThisMac: thisMac}
	w.Mac(thisMac)
	return w
}
