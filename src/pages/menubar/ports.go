package menubar

import (
	"slices"
	"strconv"
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/page"
	pageports "github.com/jterrazz/studio-station/src/pages/ports"
	"github.com/jterrazz/studio-station/src/ports"
)

const (
	PortsID page.ID = "menubar.ports"

	ActionReview        = "review"
	ActionFullList      = "full-list"
	ActionDeclarePublic = "declare-public"
	ActionStopSharing   = "stop-sharing:"

	CallDeclare     = "ports.declare"
	CallStopSharing = "ports.stop-sharing"
)

type PortsOf struct {
	// value: the machine whose ports the page lists, as the owner names it
	// (`mac-mini`, `linux-1`); empty is This Mac.
	Machine string
	// value: the hour the page is read at: a share of the last few
	// seconds is fresh against it.
	Now time.Time
}

func Ports(l ports.Listing, of PortsOf, thisMac page.Context) page.Page {
	// invariant: SPEC A4's Ports of all machines: the way back, Ports, its
	// facts (how many public, how many on the tailnet, red how many
	// exposed), then the ports body every page of ports shares; its footer
	// is Full list in Terminal and Share a port…. Scoped to a machine it
	// lists that machine's ports alone.
	sc := portsScope{Self: thisMac.Machine, Now: of.Now}
	where := "All machines"
	if of.Machine != "" {
		where = of.Machine
		sc.VM = of.Machine
		for _, h := range l.AllMachines.Macs {
			if h.Name == of.Machine {
				sc.VM, sc.Mac = "", h.Name
				where = macWord(h, sc.Self)
			}
		}
	}
	b := portsOf(l.AllMachines, sc)
	facts := []page.Value{page.Text(where), page.Text(strconv.Itoa(b.Public) + " public"), page.Text(strconv.Itoa(b.Private) + " on your tailnet")}
	if b.Exposed > 0 {
		facts = append(facts, signaled(page.Text(strconv.Itoa(b.Exposed)+" exposed"), page.SignalFailing))
	}
	p := page.Page{ID: PortsID, Command: "menubar", Context: thisMac,
		Header: &page.Header{Back: "All machines", Glyph: page.Glyph{Kind: "symbol:network"}, Buttons: []string{ActionMenu}},
		Footer: &page.Footer{Kind: page.FooterTerminals, Buttons: []string{ActionFullList, ActionShare}}}
	p.Sections = append([]page.Section{{ID: SectionSummary, Title: "Ports", Role: page.Summary, Caption: facts}}, b.Sections...)
	full := []string{"ports"}
	if of.Machine == "" {
		full = append(full, "--all-machines")
	} else if sc.Mac == "" || sc.Mac != thisMac.Machine {
		full = append(full, "--on", of.Machine)
	}
	for _, a := range []page.Action{declarePublic(l), shareAPort(l, sc),
		{ID: ActionFullList, Label: "Full list in Terminal", On: page.OnPage, Run: &page.Run{Terminal: true, Argv: full}},
		{ID: ActionMenu, Label: "More options for Ports", Icon: "ellipsis", On: page.OnPage, Submenu: []string{ActionShare, ActionDeclarePublic}}} {
		b.act(a)
	}
	p.Actions = b.Actions
	return page.Seal(portsMenu(p))
}

// value: Share a port…: the workspaces whose listeners are not shared, and
// the port to share; ports.share on the Mac of the workspace picked.
func shareAPort(l ports.Listing, sc portsScope) page.Action {
	var opts []page.Option
	for _, h := range l.AllMachines.Macs {
		for _, p := range h.Ports {
			if p.Machine.On == "" || p.Device || !sc.holds(h, p.Machine.Name) {
				continue
			}
			if !slices.ContainsFunc(opts, func(o page.Option) bool { return o.Value == p.Machine.Name }) {
				opts = append(opts, page.Option{Value: p.Machine.Name, Label: p.Machine.Name})
			}
		}
	}
	form := &page.Form{Title: "Share a port", Subtitle: "Private on your tailnet.", Fields: []page.Field{
		{Key: "VM", Label: "Of", Kind: page.FieldSelect, Options: opts, Required: true},
		{Key: "Port", Label: "Port", Kind: page.FieldText, Required: true, Note: "a port number, like 5173"},
	}}
	if len(opts) > 0 {
		form.Fields[0].Default = opts[0].Value
	}
	return page.Action{ID: ActionShare, Label: "Share a port", On: page.OnPage, Priority: 3, Mutates: true, Sheet: page.SheetSharePort, Form: form, Run: &page.Run{Call: CallSharePort}}
}

func review(machine string, rows []string) page.Action {
	argv := []string{"ports"}
	if machine != "" {
		argv = append(argv, "--on", machine)
	}
	return page.Action{ID: ActionReview, Label: "Review", On: page.OnRow, Rows: rows, Priority: 1, Run: &page.Run{Terminal: true, Argv: argv}}
}

// value: the routes a port is declared public by, in the owner's words.
func routes() []page.Option {
	out := make([]page.Option, 0, len(catalog.Routes))
	for _, r := range catalog.Routes {
		out = append(out, page.Option{Value: r.Name, Label: r.Label})
	}
	return out
}

// invariant: Declare public… offers the ports that listen there and are
// not declared yet, red ones first.
func declarePublic(l ports.Listing) page.Action {
	var opts []page.Option
	for _, red := range []bool{true, false} {
		for _, h := range l.AllMachines.Macs {
			for _, p := range h.Ports {
				if p.Declared != nil || p.Red() != red || p.Reach == ports.ThisMachine {
					continue
				}
				opts = append(opts, page.Option{Value: "Machine=" + strconv.Quote(p.Machine.Name) + " Port=" + strconv.Quote(strconv.Itoa(p.Port)), Label: p.Machine.Name + ":" + strconv.Itoa(p.Port)})
			}
		}
	}
	form := &page.Form{Title: "Declare a port public", Fields: []page.Field{
		{Key: "Port", Label: "Port", Kind: page.FieldSelect, Options: opts, Required: true},
		{Key: "Via", Label: "Via", Kind: page.FieldSelect, Options: routes(), Default: catalog.RouteCloudflareTunnel, Required: true},
	}}
	if len(opts) > 0 {
		form.Fields[0].Default = opts[0].Value
	}
	return page.Action{ID: ActionDeclarePublic, Label: "Declare public", On: page.OnPage, Priority: 2, Mutates: true, Form: form, Run: &page.Run{Call: CallDeclare}}
}

func stopSharing(s ports.SharedPort, mac string) page.Action {
	id := pageports.Row(s).ID
	port := strconv.Itoa(s.GuestPort)
	confirm := &page.Confirm{Question: "Stop sharing " + s.VM + "'s port " + port + "?",
		Lines: []page.Value{page.Text(released(s))},
		Yes:   "Stop sharing", No: "Cancel", Danger: true}
	return page.Action{ID: ActionStopSharing + id, Label: "Stop sharing", On: page.OnRow, Rows: []string{id}, Priority: 3, Mutates: true, Confirm: confirm,
		Run: &page.Run{Call: CallStopSharing, In: map[string]string{"VM": strconv.Quote(s.VM), "Port": strconv.Quote(port), "Mac": strconv.Quote(mac), "Confirm": "yes"}}}
}

func released(s ports.SharedPort) string {
	if s.Reserved == 0 {
		return "Its link stops working on " + catalog.TailnetProduct + "."
	}
	return "Its link stops working on " + catalog.TailnetProduct + ", and reserved port " + strconv.Itoa(s.Reserved) + " is let go."
}
