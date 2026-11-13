package menubar

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageports "github.com/jterrazz/studio-station/apps/engine/pages/ports"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

const (
	SectionExposed   = "exposed"
	SectionPublic    = "public"
	SectionPrivate   = "private"
	SectionListening = "listening"
	SectionLocal     = "local"
	SectionHTTPS     = "https"

	ActionOpenPort    = "open-port:"
	ActionCopyPort    = "copy-port:"
	ActionQR          = "qr:"
	ActionSharePortOf = "share:"
	ActionShowProcess = "show-process:"
	ActionHTTPS       = "https:"

	CallSetHTTPS = "ports.set-https"

	TitlePublic    = "Public"
	TitlePrivate   = "Private"
	TitleListening = "Listening, not shared"
	TitleLocal     = "This Mac only"
	TitlePorts     = "Ports"
)

// value: whose ports a body lists: every Mac's, one Mac's and its VMs',
// or one VM's; the Mac the panel runs on, which a call needs no target
// for; the hour a share is fresh against.
type portsScope struct {
	Mac  string
	VM   string
	Self string
	Now  time.Time
}

type portsBody struct {
	Sections []page.Section
	Actions  []page.Action
	Public   int
	Private  int
	Exposed  int
	Local    int
	// value: the shares and the listeners offered with Share, in the order
	// listed: a workspace's Overview card draws its shares and its first
	// listener.
	Shares    []page.Item
	Listeners []page.Item
	// value: each share's audience in the class words, by its row.
	Audience map[string]page.Value
}

func (b *portsBody) act(a page.Action) {
	for _, have := range b.Actions {
		if have.ID == a.ID {
			return
		}
	}
	b.Actions = append(b.Actions, a)
}

func (sc portsScope) holds(h ports.MacPorts, machine string) bool {
	switch {
	case sc.VM != "":
		return machine == sc.VM
	case sc.Mac != "":
		return h.Name == sc.Mac
	}
	return true
}

func macWord(h ports.MacPorts, self string) string {
	if h.ThisMac {
		return page.ThisMac
	}
	return page.MacName(h.Name, self)
}

func portsOf(f ports.AllMachines, sc portsScope) portsBody {
	// invariant: SPEC A4 Ports: what is reachable outside Tailscale and not
	// declared first, then Public (from outside), Private (on your tailnet:
	// every share, and what a VM with its own tailnet device listens on),
	// then Listening, not shared, each listener offered with Share; a Mac's
	// page ends on its HTTPS links switch. A VM's page shows Public only
	// when it has a public port, and its listeners in its Private card. A
	// port follows its VM: unreachable while it is down, never a verb then.
	var b portsBody
	var exposed, public, private, local, listening []page.Item
	for _, h := range f.Macs {
		mark := len(private)
		heard := map[string]ports.Port{}
		for _, p := range h.Ports {
			heard[p.Key()] = p
		}
		shared := map[string]bool{}
		for _, s := range h.Shared {
			if !sc.holds(h, s.VM) {
				continue
			}
			key := s.VM + ":" + strconv.Itoa(s.GuestPort)
			shared[key] = true
			p, ok := heard[key]
			if ok && p.Declared != nil {
				continue
			}
			switch {
			case s.State == ports.PhaseNotAllowed:
				listening = append(listening, b.notAllowedRow(s, p.Answer, h, sc))
			case catalog.Audience(s.To) == catalog.AudiencePublic:
				public = append(public, b.sharedRow(s, p.Answer, h, sc))
			case catalog.Audience(s.To) == catalog.AudienceLocal:
				local = append(local, b.sharedRow(s, p.Answer, h, sc))
			default:
				private = append(private, b.sharedRow(s, p.Answer, h, sc))
			}
		}
		for _, n := range h.NotShared {
			if sc.holds(h, n.VM) {
				private = append(private, page.Item{Row: &page.Row{ID: n.VM + ":not-shared", Mark: page.Attention, Reason: page.Waiting,
					Label: catalog.NotSharedWords(n.Count), Detail: page.Text(n.Detail)}})
			}
		}
		for _, q := range h.Quiet {
			if sc.holds(h, q.VM) {
				private = append(private, page.Item{Row: &page.Row{ID: q.VM + ":quiet", Mark: page.Attention, Reason: page.Waiting,
					Label: catalog.QuietWords(q.Count), Detail: page.Text(catalog.QuietHow(q.VM))}})
			}
		}
		for _, p := range h.Ports {
			if !sc.holds(h, p.Machine.Name) {
				continue
			}
			switch {
			case p.Red():
				exposed = append(exposed, b.exposedRow(p, h, sc.Self))
			case p.Declared != nil:
				public = append(public, b.publicRow(p, sc))
			case shared[p.Key()], p.Reach == ports.ThisMachine:
			case p.Device || p.Machine.On == "":
				private = append(private, b.listenerRow(p, sc, false))
			default:
				listening = append(listening, b.listenerRow(p, sc, true))
			}
		}
		slices.SortStableFunc(private[mark:], func(a, b page.Item) int { return cmp.Compare(machineOf(a), machineOf(b)) })
	}
	b.Exposed, b.Public, b.Private = len(exposed), len(public), len(private)
	b.Local = len(local)
	heardCount := len(listening)
	wholeLocal, localCapped := page.Tallied(local), len(local) > catalog.PortRowsShown
	local = capped(local)
	wholeExposed, wholePublic, wholePrivate := page.Tallied(exposed), page.Tallied(public), page.Tallied(private)
	if sc.VM != "" {
		wholePrivate = page.Tallied(slices.Concat(private, listening))
	}
	wholeListening := page.Tallied(listening)
	shown := func(all []page.Item) bool { return len(all) > catalog.PortRowsShown }
	tallied := shown(private) || sc.VM != "" && shown(listening)
	exposedCapped, publicCapped, listeningCapped := shown(exposed), shown(public), shown(listening)
	exposed, public, private, listening = capped(exposed), capped(public), capped(private), capped(listening)
	if len(exposed) > 0 {
		b.act(declarePublic(ports.Listing{AllMachines: f}))
		b.Sections = append(b.Sections, page.Section{ID: SectionExposed, Title: "Exposed", Card: page.CardExposed,
			Caption: []page.Value{signaled(page.Text(catalog.ReachableOutside), page.SignalFailing)}, Columns: columns(), Items: exposed,
			Uncounted: exposedCapped, Counts: whole(exposedCapped, wholeExposed)})
	}
	if len(public) > 0 || sc.VM == "" {
		if len(public) == 0 {
			public = []page.Item{{Empty: &page.Empty{Text: "Nothing public."}}}
		}
		b.Sections = append(b.Sections, page.Section{ID: SectionPublic, Title: TitlePublic, Glance: page.Texts(TitlePublic, "from outside").Items, Caption: []page.Value{count(b.Public)}, Columns: columns(), Items: public,
			Uncounted: publicCapped, Counts: whole(publicCapped, wholePublic)})
	}
	priv := page.Section{ID: SectionPrivate, Title: TitlePrivate, Glance: page.Texts(TitlePrivate, "on your tailnet").Items, Caption: []page.Value{count(b.Private)}, Columns: columns(), Items: private,
		About: &page.About{Label: catalog.AboutPrivateLinks, Text: catalog.PrivateLinksAre}, Uncounted: tallied, Counts: whole(tallied, wholePrivate)}
	switch {
	case sc.VM != "" && len(private) > 0 && len(listening) > 0:
		priv.Items = append(priv.Items, page.Item{Group: &page.Group{Head: page.Row{ID: SectionListening, Label: TitleListening}, Items: listening}})
	case sc.VM != "":
		priv.Items = append(priv.Items, listening...)
	}
	if len(priv.Items) == 0 {
		priv.Items = []page.Item{{Empty: &page.Empty{Text: "Nothing shared yet"}}}
	}
	b.Sections = append(b.Sections, priv)
	if len(local) > 0 {
		b.Sections = append(b.Sections, page.Section{ID: SectionLocal, Title: TitleLocal, Caption: []page.Value{count(b.Local)}, Columns: columns(), Items: local,
			Uncounted: localCapped, Counts: whole(localCapped, wholeLocal)})
	}
	if sc.VM == "" && len(listening) > 0 {
		b.Sections = append(b.Sections, page.Section{ID: SectionListening, Title: TitleListening, Caption: []page.Value{count(heardCount)}, Columns: columns(), Items: listening,
			Uncounted: listeningCapped, Counts: whole(listeningCapped, wholeListening)})
	}
	return b
}

// value: a port row's chip: its software's logo, else the quiet chip of
// a port.
func chip(n catalog.PortName) page.Glyph {
	if n.Logo != "" {
		return portGlyph(n)
	}
	return page.Glyph{Kind: "port:web"}
}

// value: the machine a port row is on, by its address: a Mac's private
// card is grouped by machine.
func machineOf(it page.Item) string {
	name, _, _ := strings.Cut(it.Row.ID, ":")
	return name
}

func signaled(v page.Value, s page.Signal) page.Value {
	v.Signal = s
	return v
}

// invariant: a section whose rows are capped keeps the tally of them all,
// so its counts never read the rows drawn alone.
func whole(capped bool, all []page.Tally) []page.Tally {
	if capped {
		return all
	}
	return nil
}

// invariant: a list shows catalog.PortRowsShown rows and one line for the
// rest, so a VM with thousands of listeners never draws thousands of rows.
func capped(items []page.Item) []page.Item {
	if len(items) <= catalog.PortRowsShown {
		return items
	}
	return append(slices.Clone(items[:catalog.PortRowsShown]), page.Item{Empty: &page.Empty{Text: catalog.MoreRows(len(items) - catalog.PortRowsShown)}})
}

func count(n int) page.Value { return page.Text(strconv.Itoa(n)) }

// value: where a port is reached, as its row's address says it, in mono:
// its machine and its port (`linux-1:5173`).
func address(machine string, port int) string { return machine + ":" + strconv.Itoa(port) }

// value: a port whose VM does not answer: greyed, said so in red after its
// name, with no verb.
func unanswered(row *page.Row, a ports.Answer) bool {
	if a == ports.Answers {
		return false
	}
	row.Signal, row.Mac = page.SignalFaint, ""
	row.Cells = []page.Value{signaled(page.Text(string(a)), page.SignalFailing)}
	return true
}

func (b *portsBody) sharedRow(s ports.SharedPort, a ports.Answer, h ports.MacPorts, sc portsScope) page.Item {
	named := s.Named()
	key := address(s.VM, s.GuestPort)
	if b.Audience == nil {
		b.Audience = map[string]page.Value{}
	}
	b.Audience[key] = pageports.Audience(s)
	mark, reason, state := pageports.State(s)
	row := page.Row{ID: key, Mark: mark, Reason: reason, Label: named.Name, Detail: page.Text(key), Glyph: chip(named)}
	if !sc.Now.IsZero() && !s.ReservedAt.IsZero() && sc.Now.Sub(s.ReservedAt) < model.FreshFor {
		row.Fresh = true
	}
	if unanswered(&row, a) {
		return page.Item{Row: &row}
	}
	stop := stopSharing(s, h.Name)
	stop.ID, stop.Rows = ActionStopSharing+key, []string{key}
	if mark != page.OK {
		flagShare(&row, mark, state)
		b.act(stop)
		row.Menu = []string{stop.ID}
		b.Shares = append(b.Shares, page.Item{Row: &row})
		return page.Item{Row: &row}
	}
	link := cmpWord(s.URLs.Public, key)
	verbs := b.linkVerbs(&row, named.Name, key, link, s.HTTP && s.URLs.Public != "")
	b.act(stop)
	row.Menu = slices.Concat(verbs, []string{"-", stop.ID})
	b.Shares = append(b.Shares, page.Item{Row: &row})
	return page.Item{Row: &row}
}

// value: a share that is not served: its state's first word flagged beside
// its name in the state's tint, its cause after its address, and no
// link, since nothing answers there.
func flagShare(row *page.Row, mark page.Mark, state page.Value) {
	words := []page.Value{state}
	if state.Kind == page.KindList {
		words = state.Items
	}
	signal := page.SignalWaits
	if mark == page.Absent {
		signal, row.Signal = page.SignalFaint, page.SignalFaint
	}
	row.Cells = []page.Value{signaled(words[0], signal)}
	if len(words) > 1 && !strings.HasPrefix(words[1].Text, "reserved ") {
		row.Detail = page.List(row.Detail, words[1])
	}
}

// value: a listener outside its VM's rights: flagged `not allowed` beside
// its name, the rights it is outside in its detail line, and offered Share,
// the owner's own call, which shares it anyway; the sheet lists it too.
func (b *portsBody) notAllowedRow(s ports.SharedPort, a ports.Answer, h ports.MacPorts, sc portsScope) page.Item {
	named := s.Named()
	key := address(s.VM, s.GuestPort)
	row := page.Row{ID: key, Mark: page.Attention, Reason: page.Waiting, Label: named.Name, Detail: page.Texts(key, rightsOf(s.Detail)), Glyph: chip(named),
		Cells: []page.Value{signaled(page.Text(catalog.NotAllowedWord), page.SignalWaits)}}
	if unanswered(&row, a) {
		return page.Item{Row: &row}
	}
	act := page.Action{ID: ActionSharePortOf + key, Label: VerbShare, On: page.OnRow, Rows: []string{key}, Mutates: true,
		Run: &page.Run{Call: CallSharePort, In: map[string]string{"VM": s.VM, "Port": strconv.Itoa(s.GuestPort)}, Target: macTarget(h.Name, sc.Self)},
		Hud: hud("Shared " + named.Name + " of " + s.VM)}
	row.Verb = act.ID
	b.act(act)
	b.Listeners = append(b.Listeners, page.Item{Row: &row})
	return page.Item{Row: &row}
}

// value: the rights a listener is outside, the words in the parentheses of
// the engine's detail (`open: 3000-4999`), else the detail whole.
func rightsOf(detail string) string {
	if _, in, ok := strings.Cut(detail, "("); ok {
		return strings.TrimSuffix(in, ")")
	}
	return detail
}

// value: a link's verbs: the row opens a web page, or copies what is not
// one, and offers its QR beside; the QR is the link's on a phone. It
// returns them in the order a share's menu lists them.
func (b *portsBody) linkVerbs(row *page.Row, name, key, link string, web bool) []string {
	copyIt := page.Action{ID: ActionCopyPort + key, Label: "Copy link to " + name, On: page.OnRow, Rows: []string{row.ID}, Run: &page.Run{Copy: link}, Hud: hud("Copied " + link)}
	if !web {
		row.Verb = copyIt.ID
		b.act(copyIt)
		return []string{copyIt.ID}
	}
	open := page.Action{ID: ActionOpenPort + key, Label: "Open " + name, On: page.OnRow, Rows: []string{row.ID}, Run: &page.Run{Open: link}}
	qr := page.Action{ID: ActionQR + key, Label: "Show QR for " + name, Icon: "qrcode", On: page.OnRow, Rows: []string{row.ID}}
	row.Verb, row.Quick = open.ID, qr.ID
	b.act(open)
	b.act(copyIt)
	b.act(qr)
	return []string{open.ID, copyIt.ID, qr.ID}
}

// value: a public port's row: the link its declaration names, its host in
// the address pill, opened in a click with its QR beside; without one, its
// address, copied.
func (b *portsBody) publicRow(p ports.Port, sc portsScope) page.Item {
	named := p.Named()
	key := p.Key()
	route, _ := catalog.RouteNamed(p.Declared.Via)
	link, at, web := key, key, false
	if domain, isWeb := p.Declared.Domain(); domain != "" {
		link, at, web = p.Declared.URL, domain, isWeb
	}
	row := page.Row{ID: key, Mark: page.OK, Label: named.Name, Detail: page.Texts(at, route.Label), Glyph: chip(named)}
	if sc.VM == "" && p.Machine.On != "" {
		row.Mac = p.Machine.Name
	}
	if !unanswered(&row, p.Answer) {
		b.linkVerbs(&row, named.Name, key, link, web)
	}
	return page.Item{Row: &row}
}

func (b *portsBody) listenerRow(p ports.Port, sc portsScope, share bool) page.Item {
	named := p.Named()
	key := p.Key()
	row := page.Row{ID: key, Mark: page.OK, Label: named.Name, Detail: page.Text(key), Glyph: chip(named)}
	if unanswered(&row, p.Answer) {
		return page.Item{Row: &row}
	}
	if !share {
		b.linkVerbs(&row, named.Name, key, key, false)
		return page.Item{Row: &row}
	}
	a := page.Action{ID: ActionSharePortOf + key, Label: VerbShare, On: page.OnRow, Rows: []string{key}, Mutates: true,
		Run: &page.Run{Call: CallSharePort, In: map[string]string{"VM": p.Machine.Name, "Port": strconv.Itoa(p.Port)}, Target: macTarget(p.Machine.Mac(), sc.Self)},
		Hud: hud("Shared " + named.Name + " of " + p.Machine.Name)}
	row.Verb = a.ID
	b.act(a)
	b.Listeners = append(b.Listeners, page.Item{Row: &row})
	return page.Item{Row: &row}
}

// invariant: an exposed port is its own card: its number and the app
// that listens, where it listens and that it is not declared, then
// Declare public… and Show process, the process its review lists.
func (b *portsBody) exposedRow(p ports.Port, h ports.MacPorts, self string) page.Item {
	key := p.Key()
	var what []page.Value
	if app := p.App(); app != "" {
		what = []page.Value{page.Text(app)}
	}
	where := p.Machine.Name
	if p.Machine.On == "" {
		where = macWord(h, self)
	}
	row := page.Row{ID: key, Mark: page.Broken, Reason: page.Exposed, Label: strconv.Itoa(p.Port), Cells: what, Detail: page.Texts(where, "on your network, not declared"),
		Glyph: symbol("exclamationmark.shield", page.SignalFailing)}
	machine := ""
	if !h.ThisMac {
		machine = h.Name
	}
	show := review(machine, []string{key})
	show.ID, show.Label = ActionShowProcess+key, "Show process"
	declare := declareHint(p, h.Name, self)
	row.Menu = []string{declare.ID, show.ID}
	b.act(declare)
	b.act(show)
	return page.Item{Row: &row}
}

// value: a Mac's HTTPS links switch: on, its shares are served with an
// https:// link through its tailnet's name; the switch is ports.set-https
// on that Mac.
func httpsSection(h ports.MacPorts, label, self string) (page.Section, page.Action) {
	on := ports.HTTPSLinks(h.HTTPS)
	v, to := page.Off(), "true"
	if on {
		v, to = page.On(), "false"
	}
	a := page.Action{ID: ActionHTTPS + h.Name, Label: "HTTPS links on " + label, On: page.OnRow, Rows: []string{SectionHTTPS}, Mutates: true,
		Run: &page.Run{Call: CallSetHTTPS, In: map[string]string{"On": to}, Target: macTarget(h.Name, self)}}
	row := page.Row{ID: SectionHTTPS, Mark: page.OK, Label: a.Label, Glyph: page.Glyph{Kind: "symbol:lock"}, Cells: []page.Value{v}, Verb: a.ID}
	return page.Section{ID: SectionHTTPS, Title: "HTTPS links", Card: page.CardFlush, Columns: columns(), Items: []page.Item{{Row: &row}}}, a
}

func segmented(b portsBody, segment string) portsBody {
	for i := range b.Sections {
		b.Sections[i].Segment = segment
	}
	return b
}

func (m Machine) self() string {
	if m.All == nil {
		return ""
	}
	for _, h := range m.All.Macs {
		if h.ThisMac {
			return h.Name
		}
	}
	return ""
}

// value: a Mac's Ports segment: its ports and its VMs', then its HTTPS
// links switch.
func macPorts(m Machine, h model.MacView) portsBody {
	if m.Ports == nil {
		row := page.Row{ID: "reading", Label: "reading", Mark: page.Awaiting}
		return portsBody{Sections: []page.Section{{ID: SectionPrivate, Title: TitlePrivate, Segment: SegmentPorts, Items: []page.Item{{Row: &row}}}}}
	}
	b := portsOf(*m.Ports, portsScope{Mac: h.Name, Self: m.self(), Now: m.Now})
	for _, mp := range m.Ports.Macs {
		if mp.Name == h.Name {
			s, a := httpsSection(mp, model.Label(*m.All, h), m.self())
			b.Sections = append(b.Sections, s)
			b.act(a)
		}
	}
	return segmented(b, SegmentPorts)
}

// value: a workspace's Ports segment: its shares and its listeners.
func workspacePorts(m Machine, vm model.VM) portsBody {
	if m.Ports == nil {
		row := page.Row{ID: "reading", Label: "reading", Mark: page.Awaiting}
		return portsBody{Sections: []page.Section{{ID: SectionPrivate, Title: TitlePrivate, Segment: SegmentPorts, Items: []page.Item{{Row: &row}}}}}
	}
	return segmented(portsOf(*m.Ports, portsScope{VM: vm.Name, Self: m.self(), Now: m.Now}), SegmentPorts)
}
