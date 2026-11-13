package machinefile

import (
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func ImageDrift(baked []byte, now Template) []string {
	// invariant: an image is stale by what its build's lock (baked) says
	// against its Machinefile now (hard case 17): an item the Machinefile
	// bakes now that the image holds no lock line of, and an item the image
	// holds whose lock now asks another method or version. A lock either side
	// cannot read says nothing: drift is only ever what can be named.
	then, err := catalog.DecodeLock(machinefilesDir+"/"+now.Name+catalog.LockSuffix, baked)
	if err != nil || len(baked) == 0 {
		return nil
	}
	held := map[string]catalog.LockedApp{}
	for _, a := range then.Apps {
		held[a.Name] = a
	}
	var out []string
	for _, b := range now.Bake {
		if _, ok := held[b]; !ok {
			out = append(out, b+" is not baked into the image")
		}
	}
	if len(now.Lock) == 0 {
		return out
	}
	locked, err := catalog.DecodeLock(machinefilesDir+"/"+now.Name+catalog.LockSuffix, now.Lock)
	if err != nil {
		return out
	}
	for _, a := range locked.Apps {
		h, ok := held[a.Name]
		switch {
		case !ok, h == a:
		case h.Method != a.Method:
			out = append(out, a.Name+" is by "+h.Method+" in the image, "+a.Method+" locked")
		default:
			out = append(out, a.Name+" "+said(h.Version)+" in the image, "+said(a.Version)+" locked")
		}
	}
	return slices.Compact(out)
}

func said(version string) string {
	if version == "" {
		return "unpinned"
	}
	return version
}
