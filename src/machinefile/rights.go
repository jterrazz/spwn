package machinefile

import (
	"slices"

	"github.com/jterrazz/studio-station/src/catalog"
)

func RightsOf(trees []Tree, name string) (rights *catalog.PortRights, err error) {
	// invariant: the ports a Machinefile names are read from its own files
	// alone: itself, what it extends (depth first, a file read once), and the
	// owner's overlay after each, the last that says ports winning whole, as
	// the resolution merges them. A file elsewhere that the decoder refuses
	// changes nothing here; one of these that it refuses is the error. rights
	// is nil when no source holds a Machinefile of that name, or its files say
	// no ports.
	var seen []string
	var walk func(name string, stack []string) error
	walk = func(name string, stack []string) error {
		if i := slices.Index(stack, name); i >= 0 {
			return &CycleError{Path: append(slices.Clone(stack[i:]), name), Of: "extends"}
		}
		mf, ok, decodeErr := decodeNamed(trees, name, false)
		if decodeErr != nil || !ok {
			if decodeErr == nil && len(stack) > 0 {
				decodeErr = &NotFoundError{Name: name, By: stack[len(stack)-1]}
			}
			return decodeErr
		}
		stack = append(stack, name)
		for _, e := range mf.Extends {
			if walkErr := walk(e, stack); walkErr != nil {
				return walkErr
			}
		}
		if slices.Contains(seen, name) {
			return nil
		}
		seen = append(seen, name)
		if mf.Ports != nil {
			rights = mf.Ports
		}
		overlay, ok, overlayErr := decodeNamed(trees, name, true)
		if overlayErr == nil && ok && overlay.Ports != nil {
			rights = overlay.Ports
		}
		return overlayErr
	}
	err = walk(name, nil)
	return rights, err
}

// value: the Machinefile of that name in the strongest tree that holds
// one, or its overlay, the owner's alone.
func decodeNamed(trees []Tree, name string, overlay bool) (catalog.Machinefile, bool, error) {
	file := machinefilesDir + "/" + name + ".yaml"
	if overlay {
		file = machinefilesDir + "/" + name + catalog.OverlaySuffix
	}
	for _, t := range trees {
		if overlay && t.Layer != User {
			continue
		}
		for _, f := range t.Files {
			if f.Path != file {
				continue
			}
			mf, err := catalog.DecodeMachinefile(t.display(f), f.Body)
			if err != nil {
				return catalog.Machinefile{}, false, refusedFile("machinefile", err)
			}
			return mf, true, nil
		}
	}
	return catalog.Machinefile{}, false, nil
}
