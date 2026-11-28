package machinefile

import (
	"errors"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func RightsOf(trees []Tree, name, machine string) (rights *catalog.PortRights, by string, err error) {
	// invariant: the ports a Machinefile names are read from its own files
	// alone, within the chain's bounds (walkChain), each decoded once:
	// itself, what it extends (depth first, a file read once), and the
	// owner's overlay after each, then the machine's own changes when it
	// names none of them, the last file that says ports winning whole and an
	// overlay changing them key by key, as the resolution merges them. A file
	// elsewhere that the decoder refuses changes nothing here; one of these
	// that it refuses is the error. rights is nil when no source holds a
	// Machinefile of that name, or its files say no ports. by is the file
	// that says them (`base`, `web.local`), else the Machinefile of that
	// name, which leaves them to the default; empty when there is none. A
	// chain the resolution refuses (two siblings giving one entry two ways,
	// judged for the chain's own system, merged.systemFor) is refused here
	// the same, so no VM opens a port by it.
	var seen []string
	var chain []Placed
	over := func(of, overlay string) error {
		p, ok, overlayErr := decodeNamed(trees, overlay, true)
		mf := p.File
		if ok && overlayErr == nil && mf.Overrides == of {
			chain = append(chain, p)
		}
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
	read := map[string]Placed{}
	seen, err = walkChain(name, func(name string) ([]string, bool, error) {
		p, ok, decodeErr := decodeNamed(trees, name, false)
		read[name] = p
		return p.File.Extends, ok, decodeErr
	})
	var none *NotFoundError
	if errors.As(err, &none) && none.By == "" {
		err = nil
	}
	for _, n := range seen {
		chain = append(chain, read[n])
		if p := read[n].File.Ports; p != nil {
			whole := *p
			rights, by = &whole, n
		}
		if overErr := over(n, n); overErr != nil {
			err = overErr
			break
		}
	}
	if err == nil && len(seen) > 0 && machine != "" && !slices.Contains(seen, machine) {
		err = over(name, machine)
	}
	if err == nil && len(seen) > 0 {
		err = judgedChain(trees, name, chain)
	}
	if rights == nil {
		by = name
		if len(seen) == 0 {
			by = ""
		}
	}
	return rights, by, err
}

// value: the refusal of the merge of these files, judged against the
// sources' catalog for the system the chain says, as the resolution
// judges it; nil when it stands.
func judgedChain(trees []Tree, name string, chain []Placed) error {
	cat, err := CatalogOf(trees)
	if err != nil {
		return err
	}
	m := merge(chain)
	return m.judged(cat, m.systemFor(name, Machine{}))
}

// value: the Machinefile of that name in the strongest tree that holds
// one, or its overlay, the owner's alone, placed as the library places it.
func decodeNamed(trees []Tree, name string, overlay bool) (Placed, bool, error) {
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
				return Placed{}, false, refusedFile("machinefile", err)
			}
			kind := t.Kind
			if overlay {
				kind = Overlay
			}
			return Placed{File: mf, Kind: kind, Path: t.display(f), Body: f.Body}, true, nil
		}
	}
	return Placed{}, false, nil
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
