package machinefile

import (
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func RightsOf(trees []Tree, name string) (rights *catalog.PortRights, by string, err error) {
	// invariant: the ports a Machinefile names are read from its own files
	// alone: itself, what it extends (depth first, a file read once), and the
	// owner's overlay after each, the last that says ports winning whole, as
	// the resolution merges them. A file elsewhere that the decoder refuses
	// changes nothing here; one of these that it refuses is the error. rights
	// is nil when no source holds a Machinefile of that name, or its files say
	// no ports. by is the file that says them (`base`, `web.local`), else the
	// Machinefile of that name, which leaves them to the default; empty when
	// there is none.
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
			rights, by = mf.Ports, name
		}
		overlay, ok, overlayErr := decodeNamed(trees, name, true)
		if overlayErr == nil && ok && overlay.Ports != nil {
			rights, by = overlay.Ports, strings.TrimSuffix(name+catalog.OverlaySuffix, ".yaml")
		}
		return overlayErr
	}
	err = walk(name, nil)
	if rights == nil {
		by = name
		if len(seen) == 0 {
			by = ""
		}
	}
	return rights, by, err
}

// value: the Machinefile of that name in the strongest tree that holds
// one, or its overlay, the owner's alone.
func decodeNamed(trees []Tree, name string, overlay bool) (catalog.Machinefile, bool, error) {
	file := machinefilesDir + "/" + name + catalog.MachinefileSuffix
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
