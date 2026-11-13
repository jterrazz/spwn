package ports

import (
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/pages"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

type TailnetJSON struct {
	State        string `json:"state"`
	Connected    bool   `json:"connected"`
	BackendState string `json:"backend_state"`
	Name         string `json:"name"`
	DNSName      string `json:"dns_name"`
	IP           string `json:"ip"`
	Variant      string `json:"variant"`
	CLI          string `json:"cli"`
}

func TailnetObject(s ports.Status) TailnetJSON {
	return TailnetJSON{
		State: string(s.State), Connected: s.Connected, BackendState: s.BackendState,
		Name: s.Name, DNSName: s.DNSName, IP: s.IP, Variant: s.Variant, CLI: s.CLI,
	}
}

func tailnetColumns() []page.Column {
	return []page.Column{{Key: "name", Kind: page.KindText, Keep: 100}, {Key: "value", Kind: page.KindList, Keep: 50}}
}

func TailnetSections(s ports.Status) []page.Section {
	rows := tailnetRows(s)
	switch {
	case s.State == catalog.StatePending:
		rows = append(rows, page.Row{ID: "node", Label: "node", Mark: page.Awaiting}, page.Row{ID: "cli", Label: "cli", Mark: page.Awaiting})
	case s.CLI != "":
		rows = append(rows, page.Row{ID: "cli", Label: "cli", Mark: page.OK, Cells: []page.Value{page.Path(s.CLI)}, Detail: page.Text(s.Variant)})
	}
	head := page.Section{ID: "tailnet", Title: catalog.TailnetProduct, Role: page.Banner, Columns: tailnetColumns(), Items: tailnetItems(rows)}
	if s.State == catalog.StateAbsent {
		return []page.Section{head}
	}
	peers := make([]page.Row, 0, len(s.Peers))
	for _, p := range s.Peers {
		state := page.Text("online")
		if !p.Online {
			state = page.Text("offline")
		}
		peers = append(peers, page.Row{ID: p.Name, Label: p.Name, Cells: []page.Value{page.Identity(p.IP), state, page.Texts(p.Tags...)}})
	}
	serve := make([]page.Row, 0, len(s.Serve))
	for _, e := range s.Serve {
		owner := "not station's"
		if e.Station {
			owner = "station"
		}
		label := "https " + strconv.Itoa(e.Port) + e.Path
		serve = append(serve, page.Row{ID: label, Label: label, Mark: page.OK, Cells: []page.Value{page.URL(e.Target)}, Detail: page.Text(owner)})
	}
	return []page.Section{head,
		{ID: "peers", Title: "Peers", Columns: []page.Column{
			{Key: "name", Kind: page.KindListing, Keep: 100}, {Key: "ip", Kind: page.KindIdentity, Keep: 60},
			{Key: "state", Kind: page.KindText, Keep: 70, Show: page.EyeOnly}, {Key: "tags", Kind: page.KindList, Keep: 20, Whole: true, Show: page.EyeOnly},
		}, Items: tailnetList(s, "peers", peers)},
		{ID: "serve", Title: "Serve", Columns: tailnetColumns(), Items: tailnetList(s, "serves", serve)},
	}
}

func tailnetItems(rows []page.Row) []page.Item {
	items := make([]page.Item, 0, len(rows))
	for i := range rows {
		items = append(items, page.Item{Row: &rows[i]})
	}
	return items
}

func tailnetList(s ports.Status, noun string, rows []page.Row) []page.Item {
	switch {
	case s.State == catalog.StatePending:
		return []page.Item{{Row: &page.Row{ID: noun, Label: "reading", Mark: page.Awaiting}}}
	case len(rows) == 0:
		return []page.Item{{Empty: &page.Empty{Noun: noun}}}
	}
	return tailnetItems(rows)
}

func tailnetRows(s ports.Status) []page.Row {
	if s.State == catalog.StatePending {
		return []page.Row{{ID: "tailnet", Label: catalog.TailnetProduct, Mark: page.Awaiting}}
	}
	head := page.Row{ID: "tailnet", Label: catalog.TailnetProduct, Mark: pages.MarkOf(s.State), Cells: []page.Value{stateValue(s)}, Hint: repair(s)}
	switch head.Mark {
	case page.Attention:
		head.Reason = page.Degraded
	case page.Broken:
		head.Reason = page.Down
	case page.Absent:
		head.Reason = page.Missing
	case page.OK, page.Awaiting, page.OffHere, page.NotForThisMachine, page.NotKnown, page.Unjudged:
	}
	rows := []page.Row{head}
	if s.Name != "" {
		rows = append(rows, page.Row{ID: "node", Label: "node", Mark: page.OK, Cells: []page.Value{page.Text(s.Name)}, Detail: page.Text(s.IP)})
	}
	return rows
}

func stateValue(s ports.Status) page.Value {
	switch {
	case s.State == catalog.StateAbsent:
		return page.Text("not installed")
	case s.BackendState == "":
		return page.Text("the " + catalog.TailnetProduct + " app does not answer")
	}
	return page.Texts(s.BackendState, s.DNSName)
}

func TailnetSummary(s ports.Status) page.Row {
	row := tailnetRows(s)[0]
	if s.State != catalog.StateOK {
		return row
	}
	row.Cells = []page.Value{page.Texts(s.BackendState, s.DNSName, s.IP, s.Variant, s.CLI)}
	return row
}

func repair(s ports.Status) *page.Hint {
	switch s.State {
	case catalog.StateOK, catalog.StatePending:
		return nil
	case catalog.StateAbsent:
		return page.Cmd("station apps install " + strings.ToLower(catalog.TailnetProduct))
	case catalog.StateDegraded, catalog.StateBroken, catalog.StateUnknown:
	}
	return &page.Hint{Command: "open " + catalog.TailnetProduct + " and sign in", External: true}
}
