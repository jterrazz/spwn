package machinefile

import (
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func RightsOf(trees []Tree, name, machine string) (rights *catalog.PortRights, by string, err error) {
	// invariant: the ports a Machinefile names are read from its own files
	// alone: itself, what it extends (depth first, a file read once), and the
	// owner's overlay after each, then the machine's own changes when it
	// names none of them, the last file that says ports winning whole and an
	// overlay changing them key by key, as the resolution merges them. A file
	// elsewhere that the decoder refuses changes nothing here; one of these
	// that it refuses is the error. rights is nil when no source holds a
	// Machinefile of that name, or its files say no ports. by is the file
	// that says them (`base`, `web.local`), else the Machinefile of that
	// name, which leaves them to the default; empty when there is none.
	var seen []string
	over := func(of, overlay string) error {
		mf, ok, overlayErr := decodeNamed(trees, overlay, true)
		if overlayErr != nil || !ok || mf.Overrides != of || mf.Ports == nil {
			return overlayErr
		}
		changed := *mf.Ports
		if rights != nil {
			changed = rights.Over(changed)
		}
		file := overlay + catalog.OverlaySuffix
		if problem := changed.Problem(); problem != "" {
			return refusedFile("machinefile", &catalog.FileError{File: file, Problem: "its ports with those of " + of + "'s chain: " + problem})
		}
		rights, by = &changed, strings.TrimSuffix(file, ".yaml")
		return nil
	}
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
			whole := *mf.Ports
			rights, by = &whole, name
		}
		return over(name, name)
	}
	err = walk(name, nil)
	if err == nil && len(seen) > 0 && machine != "" && !slices.Contains(seen, machine) {
		err = over(name, machine)
	}
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
		if overlay && t.Kind != User {
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

func Installs(lib Library, name, machine string) ([]string, error) {
	// invariant: what a VM's Machinefile installs is its plan's packages,
	// resolved for that machine as its own apply resolves it: those it
	// lists, bakes, and brings in by what they require, by the first method
	// its system matches, no lock read, since a lock picks among the methods
	// the file allows. None when the library holds no Machinefile of that
	// name; a plan that cannot be resolved is the error.
	if _, ok := lib.Machinefiles[name]; !ok {
		return nil, nil
	}
	r, err := Resolve(lib, name, Machine{Name: machine}, Lock{})
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(r.Steps))
	for _, s := range r.Steps {
		out = append(out, s.Package)
	}
	return out, nil
}
