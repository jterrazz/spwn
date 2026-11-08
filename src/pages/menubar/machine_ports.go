package menubar

import (
	"github.com/jterrazz/studio-station/src/catalog"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/page"
	"strconv"
	"strings"
)

// value: the workspace's shared ports, each a link: private, on Tailscale;
// one with no profile is TCP, or Web when it has a web link.
func portsSection(m Machine, _ model.MacView, vm model.VM) page.Section {
	s := page.Section{ID: SectionPorts, Title: "Private", Caption: []page.Value{page.Text("shared on " + catalog.TailnetProduct)}, Uncounted: true}
	if m.Ports == nil {
		s.Items = []page.Item{{Row: &page.Row{ID: "reading", Label: "reading", Mark: page.Awaiting}}}
		return s
	}
	for _, h := range m.Ports.Macs {
		for _, sv := range h.Shared {
			if sv.VM != vm.Name {
				continue
			}
			value := page.Text(h.PublicName + ":" + strconv.Itoa(sv.Reserved))
			if sv.URLs.Public != "" {
				value = page.URL(sv.URLs.Public)
			}
			label := cmpWord(titled(sv.Profile), "TCP")
			if sv.Profile == "" && strings.HasPrefix(sv.URLs.Public, "http") {
				label = "Web"
			}
			s.Items = append(s.Items, page.Item{Link: &page.Link{Label: label, Lead: page.List(page.Text(label), page.Port(int64(sv.GuestPort))), Value: value, Row: strconv.Itoa(sv.GuestPort)}})
		}
	}
	if len(s.Items) == 0 {
		s.Items = []page.Item{{Empty: &page.Empty{Text: "No port shared yet"}}}
	}
	return s
}

// value: Stop sharing on each of the workspace's shared ports, the row
// its link is.
func stopSharings(m Machine, vm model.VM) []page.Action {
	if m.Ports == nil {
		return nil
	}
	var out []page.Action
	for _, h := range m.Ports.Macs {
		for _, sv := range h.Shared {
			if sv.VM != vm.Name {
				continue
			}
			a := stopSharing(sv, h.Name)
			row := strconv.Itoa(sv.GuestPort)
			a.ID, a.Rows = ActionStopSharing+vm.Name+"/"+row, []string{row}
			out = append(out, a)
		}
	}
	return out
}

// value: a Mac's Ports segment: the ports its VMs share, each a link.
func macPorts(m Machine, h model.MacView) page.Section {
	s := page.Section{ID: SectionPorts, Title: "Ports", Segment: SegmentPorts, Uncounted: true}
	if m.Ports != nil {
		for _, mp := range m.Ports.Macs {
			if mp.Name != h.Name {
				continue
			}
			for _, sv := range mp.Shared {
				value := page.Text(mp.PublicName + ":" + strconv.Itoa(sv.Reserved))
				if sv.URLs.Public != "" {
					value = page.URL(sv.URLs.Public)
				}
				s.Items = append(s.Items, page.Item{Link: &page.Link{Label: sv.VM, Lead: page.List(page.Text(sv.VM), page.Port(int64(sv.GuestPort))), Value: value, Row: sv.VM + "/" + strconv.Itoa(sv.GuestPort)}})
			}
		}
	}
	if len(s.Items) == 0 {
		s.Items = []page.Item{{Empty: &page.Empty{Text: "No port shared yet"}}}
	}
	return s
}
