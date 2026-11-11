package allmachines

import (
	"github.com/jterrazz/studio-station/src/catalog"

	"github.com/jterrazz/studio-station/src/images"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/pages"
	pageimages "github.com/jterrazz/studio-station/src/pages/images"
	pagemachines "github.com/jterrazz/studio-station/src/pages/machines"
	pageports "github.com/jterrazz/studio-station/src/pages/ports"
	"github.com/jterrazz/studio-station/src/ports"
)

const (
	KeyBuild  = "b"
	KeyForget = "f"
	KeyHealth = "p"
	KeyUnlock = "o"
)

type Reading struct {
	Machines *model.AllMachines
	Images   *[]images.Report
	// value: the images the catalog knows, one no Mac holds shown as such.
	Known   []catalog.Image
	Ports   *ports.AllMachines
	Tailnet *ports.Status
	Failed  map[string]page.Error
}

func Tab(r Reading, thisMac page.Context) page.Page {
	p := page.Page{ID: "all-machines", Context: thisMac}
	p.Sections = append(p.Sections, part(r, "all-machines", "All machines", r.Machines != nil, func() []page.Section {
		s := pagemachines.Section(*r.Machines, links(r.Ports))
		s.Fold = page.Open
		return []page.Section{s}
	})...)
	p.Sections = append(p.Sections, part(r, "images", "Images", r.Images != nil, func() []page.Section {
		s := pageimages.Section(*r.Images, r.Known, thisMac)
		s.Role, s.Fold = page.Plain, page.Open
		return []page.Section{s}
	})...)
	p.Sections = append(p.Sections, part(r, "tailnet", catalog.TailnetProduct, r.Tailnet != nil, func() []page.Section {
		parts := pageports.TailnetSections(*r.Tailnet)
		head := parts[0]
		head.Role = page.Plain
		return []page.Section{pages.Nest(head, parts[1:])}
	})...)
	p.Actions = Actions(r)
	return page.Seal(p)
}

func part(r Reading, id, title string, ready bool, build func() []page.Section) []page.Section {
	if e, failed := r.Failed[id]; failed {
		return []page.Section{{ID: id, Title: title, Columns: []page.Column{{Key: "name", Kind: page.KindText, Keep: 100}}, Items: []page.Item{{Error: &e}}}}
	}
	if !ready {
		return []page.Section{{ID: id, Title: title, Columns: []page.Column{{Key: "name", Kind: page.KindText, Keep: 100}},
			Items: []page.Item{{Row: &page.Row{ID: id, Label: "reading", Mark: page.Awaiting}}}}}
	}
	return build()
}

func links(f *ports.AllMachines) func(mac, vm string) []page.Link {
	if f == nil {
		return nil
	}
	return func(mac, vm string) []page.Link {
		var out []page.Link
		for _, h := range f.Macs {
			if h.Name != mac {
				continue
			}
			for _, s := range h.Shared {
				if s.VM == vm {
					out = append(out, pageports.Link(s))
				}
			}
		}
		return out
	}
}

func Actions(r Reading) []page.Action {
	var vms, remote, macs []string
	if r.Machines != nil {
		for _, h := range r.Machines.Macs {
			macs = append(macs, h.Name)
			if !h.ThisMac {
				remote = append(remote, h.Name)
			}
			for _, vm := range h.VMs {
				vms = append(vms, vm.Name)
				remote = append(remote, vm.Name)
			}
		}
	}
	var out []page.Action
	add := func(a page.Action, rows []string) {
		if a.On == page.OnPage || len(rows) > 0 {
			a.Rows = rows
			out = append(out, a)
		}
	}
	add(page.Action{Key: pagemachines.KeyDelete, Label: "delete", On: page.OnRow, Priority: 1, Mutates: true, Confirm: &page.Confirm{Yes: "delete", Danger: true}}, vms)
	add(page.Action{Key: pagemachines.KeyRestart, Label: "restart", On: page.OnRow, Priority: 2, Mutates: true, Confirm: &page.Confirm{Yes: "restart", Danger: true}}, append(append([]string{}, macs...), vms...))
	add(page.Action{Key: pagemachines.KeyEnter, Label: "enter", On: page.OnRow, Priority: 3, Mutates: true}, remote)
	add(page.Action{Key: pagemachines.KeyCreate, Label: "create", On: page.OnPage, Priority: 4, Mutates: true}, nil)
	add(page.Action{Key: pagemachines.KeyStart, Label: "start/stop", On: page.OnRow, Priority: 5, Mutates: true}, vms)
	add(page.Action{Key: KeyForget, Label: "forget", On: page.OnRow, Priority: 6, Mutates: true, Confirm: &page.Confirm{Yes: "forget"}}, without(remote, vms))
	add(page.Action{Key: KeyBuild, Label: "build images", On: page.OnPage, Priority: 7, Mutates: true}, nil)
	add(page.Action{Key: KeyHealth, Label: "health", On: page.OnRow, Priority: 8}, remote)
	add(page.Action{Key: KeyUnlock, Label: "unlock", On: page.OnRow, Priority: 9, Mutates: true}, without(remote, vms))
	return out
}

func without(list, drop []string) []string {
	var out []string
	for _, x := range list {
		keep := true
		for _, d := range drop {
			keep = keep && x != d
		}
		if keep {
			out = append(out, x)
		}
	}
	return out
}
