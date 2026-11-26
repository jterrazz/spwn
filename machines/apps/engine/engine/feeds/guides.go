package feeds

import (
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/core/guide"
	"github.com/jterrazz/studio-station/apps/engine/engine/store"
	"github.com/jterrazz/studio-station/apps/engine/machines"
)

// value: every act the engines of the fleet guide, each as its Guide: This
// Mac's board and the copy of every other Mac's, the call engine.guides answers.
const ViewGuides = "engine.guides"

// value: the act families a Guide is projected for; the calls that name one
// and these are one list (rule guides).
func GuidedFamilies() []string { return []string{machines.GuideCreate} }

// value: the projection of an act family, by the family a call's Guide
// names; false for a family no projection knows.
func Guided(family string) (func(a guide.Act, vm *machines.VM) guide.Guide, bool) {
	switch family {
	case machines.GuideCreate:
		return machines.CreateGuide, true
	}
	return nil, false
}

// invariant: a Mac whose board never held an act has no slice: the list
// reads fresh and empty, as the board says it all when it holds nothing.
func (v view) guides(all *machines.AllMachines) store.View {
	keys := []store.Key{}
	for k := range v.s.Slices {
		if k.Family() == store.FamilyActs {
			keys = append(keys, k)
		}
	}
	slices.Sort(keys)
	out := guide.Guides{Guides: []guide.Guide{}}
	for _, k := range keys {
		held, _, ok := value[guide.Acts](v.s, k)
		if !ok {
			continue
		}
		for _, a := range held.Acts {
			if project, known := Guided(a.Family); known {
				out.Guides = append(out.Guides, project(a, vmOn(all, k.Machine(), a.Subject)))
			}
		}
	}
	own, had := v.s.Slices[store.Of(store.FamilyActs, v.this)]
	if !had {
		return store.View{Value: out, State: store.Fresh}
	}
	return viewOf(own, out)
}

func vmOn(all *machines.AllMachines, mac, name string) *machines.VM {
	for _, h := range all.Macs {
		if h.Name != mac {
			continue
		}
		for i := range h.VMs {
			if h.VMs[i].Name == name {
				vm := h.VMs[i]
				return &vm
			}
		}
	}
	return nil
}
