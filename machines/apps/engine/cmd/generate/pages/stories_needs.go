package pages

import (
	"github.com/jterrazz/studio-station/apps/engine/overview"
)

// value: the worlds of Home's sections and of the Needs you page: the home
// scene with Awake off; the fleet's updates installed, then the owner's
// package file refused.
func needsWorlds() []World {
	home := func(name string, edits ...func(*Scene)) World {
		return World{Name: name, Edit: func(f *Scene) {
			homeScene(f)
			for _, e := range edits {
				e(f)
			}
		}, Pages: []string{"menubar"}}
	}
	needs := func(name string, edit func(*Scene)) World {
		return World{Name: name, Edit: edit, Pages: []string{"needs"}}
	}
	return []World{
		home("machines.home.awake-off", awakeOff),
		needs("machines.needs.updated", updatesInstalled),
		needs("machines.needs.catalog-refused", catalogRefused),
	}
}

func awakeOff(f *Scene) {
	f.Awake.Session = nil
}

// value: the owner's package file refused: the run reads the built-in packages,
// and Needs you names the file and its line.
func catalogRefused(f *Scene) {
	f.Home.Needs = append(f.Home.Needs, overview.Need{Machine: f.Home.Head.Name, Section: "overview", Broken: true, Severity: "broken",
		Subject: overview.RefusedSubject, Text: "~/.config/station/catalog/apps/jq.yaml:3: unknown key `vesion`", Path: "~/.config/station/catalog/apps/jq.yaml"})
}

func updatesInstalled(f *Scene) {
	var kept []overview.Need
	for _, n := range f.Home.Needs {
		if n.Call != "apps.update" {
			kept = append(kept, n)
		}
	}
	f.Home.Needs = kept
}
