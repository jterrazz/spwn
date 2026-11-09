package menubar

import (
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/accounts"
	"github.com/jterrazz/studio-station/src/awake"
	"github.com/jterrazz/studio-station/src/core/machine"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/page"
)

const (
	// value: a menu's header, the first entry of its list: its words only.
	ActionTitle = "title:"
	// value: a row's own page, opened from its menu: the row's Nav.
	ActionOpenRow = "open:"
	ActionCopyOf  = "copy-of:"
	ActionShareOf = "share-of:"

	// value: an action reached through a menu alone, never drawn as a
	// row's or a page's button.
	OnMenu page.Scope = "menu"

	MenuSeparator = "-"

	appLogin     = "app:login"
	appSparkline = "app:sparkline"
	appQuit      = "app:quit"
)

// value: a menu's list being built: its entries in order, a separator
// between groups (never first, last or doubled), and the actions the
// entries name that the page does not carry yet.
type menu struct {
	ids   []string
	added []page.Action
	group bool
}

func (m *menu) title(id, words string) {
	m.added = append(m.added, page.Action{ID: ActionTitle + id, Label: words, On: OnMenu})
	m.ids = append(m.ids, ActionTitle+id)
}

func (m *menu) item(id string) {
	if id == "" {
		return
	}
	if m.group {
		m.ids = append(m.ids, MenuSeparator)
		m.group = false
	}
	m.ids = append(m.ids, id)
}

func (m *menu) add(a page.Action) {
	m.added = append(m.added, a)
	m.item(a.ID)
}

// invariant: a separator stands only between two groups that each hold
// an item.
func (m *menu) next() {
	if len(m.ids) > 0 && !strings.HasPrefix(m.ids[len(m.ids)-1], ActionTitle) && m.ids[len(m.ids)-1] != MenuSeparator {
		m.group = true
	}
}

// value: a copy of an action for a menu: its own id on the page, its
// menu's words and symbol; reached through the menu alone.
// value: a gesture the page already carries, its alert graded by its
// facts, moved into the menu alone under the menu's shorter words.
func (m *menu) own(p *page.Page, id, label, icon string, destructive bool) {
	for i := range p.Actions {
		if a := &p.Actions[i]; a.ID == id {
			a.On, a.Label, a.Icon = OnMenu, label, icon
			a.Destructive = destructive && a.Confirm != nil
			m.item(id)
			return
		}
	}
}

func menuCopy(a page.Action, id, label, icon string) page.Action {
	a.ID, a.On, a.Rows = id, OnMenu, nil
	if label != "" {
		a.Label = label
	}
	if icon != "" {
		a.Icon = icon
	}
	return a
}

func pageAction(p page.Page, id string) (page.Action, bool) {
	for _, a := range p.Actions {
		if a.ID == id {
			return a, true
		}
	}
	return page.Action{}, false
}

func actionsWith(p page.Page, prefix string) []page.Action {
	var out []page.Action
	for _, a := range p.Actions {
		if strings.HasPrefix(a.ID, prefix) {
			out = append(out, a)
		}
	}
	return out
}

func menuID(id, row string) string { return id + "@" + row }

// value: the words a link's copy is said by in the Copy submenu: "Link
// to Vite" for "Copy link to Vite".
func underCopy(label string) string {
	rest := strings.TrimPrefix(label, "Copy ")
	if rest == "" {
		return label
	}
	return strings.ToUpper(rest[:1]) + rest[1:]
}

// value: the name a link's copy names: "Vite" for "Copy link to Vite".
func linkName(label string) string { return strings.TrimPrefix(label, "Copy link to ") }

// invariant: SPEC A5's menu of a VM's row: its header (the VM and its
// Mac), Open, Open in Terminal and its first link; Copy (its ssh command
// and each link) and Share (the QR of its first link, a port, an account);
// its lifecycle as its own page offers it; Delete last. Every verb is the
// one its page carries, so it asks as the page asks.
func vmMenu(row page.Row, header string, own page.Page) menu {
	var m menu
	key := row.ID
	m.title(key, header)
	if row.Nav != "" {
		m.add(page.Action{ID: ActionOpenRow + key, Label: "Open", Icon: "arrow.up.forward.square", Key: "enter", On: OnMenu})
	}
	if a, ok := pageAction(own, ActionAttach+key); ok {
		m.add(terminalItem(a, key))
	} else if a, ok := pageAction(own, ActionOpenStation); ok && a.Disabled == "" && own.Footer != nil && slices.Contains(own.Footer.Buttons, ActionOpenStation) {
		m.add(terminalItem(a, key))
	}
	links := vmLinks(own)
	if len(links) > 0 {
		first := links[0]
		m.add(page.Action{ID: menuID("open-link", key), Label: "Open " + linkName(first.Label), Icon: "safari", On: OnMenu, Run: &page.Run{Open: first.Run.Copy}})
	}
	m.next()
	copies := menu{}
	if ssh := sshOf(own); ssh != "" {
		copies.add(page.Action{ID: menuID(ActionCopySSH, key), Label: "SSH command", Icon: "terminal", On: OnMenu, Run: &page.Run{Copy: ssh}, Hud: hud("Copied " + ssh)})
	}
	for _, l := range links {
		copies.add(menuCopy(l, menuID(l.ID, key), underCopy(l.Label), "link"))
	}
	if len(copies.ids) > 0 {
		m.added = append(m.added, copies.added...)
		m.add(page.Action{ID: ActionCopyOf + key, Label: "Copy", Icon: "doc.on.doc", On: OnMenu, Submenu: copies.ids})
	}
	shares := menu{}
	if qr := actionsWith(own, ActionQR); len(qr) > 0 {
		shares.add(menuCopy(qr[0], menuID(qr[0].ID, key), "Show QR", "qrcode"))
	}
	if a, ok := pageAction(own, ActionShare); ok {
		shares.add(menuCopy(a, menuID(a.ID, key), "A port", "point.3.connected.trianglepath.dotted"))
	}
	if a, ok := pageAction(own, ActionShareAccount); ok {
		shares.add(menuCopy(a, menuID(a.ID, key), "An account", "key"))
	}
	if len(shares.ids) > 0 {
		m.added = append(m.added, shares.added...)
		m.add(page.Action{ID: ActionShareOf + key, Label: "Share", Icon: "square.and.arrow.up", On: OnMenu, Submenu: shares.ids})
	}
	m.next()
	if a, ok := pageAction(own, ActionRestartUp); ok {
		m.add(menuCopy(a, menuID(ActionRestart, key), "Restart", "arrow.clockwise"))
	} else if a, ok := pageAction(own, ActionRestart); ok {
		m.add(menuCopy(a, menuID(a.ID, key), "", "arrow.clockwise"))
	}
	if a, ok := pageAction(own, ActionStop); ok {
		m.add(menuCopy(a, menuID(a.ID, key), "", "stop"))
	}
	if a, ok := pageAction(own, ActionStartVM); ok {
		m.add(menuCopy(a, menuID(a.ID, key), "Start", "play"))
	}
	m.next()
	if a, ok := pageAction(own, ActionDelete); ok {
		a = menuCopy(a, menuID(a.ID, key), "", "trash")
		a.Destructive = a.Confirm != nil
		m.add(a)
	}
	return m
}

// value: a VM's links, each copied as its page copies it: a workspace's
// shared ports, a server's served sites.
func vmLinks(own page.Page) []page.Action {
	var out []page.Action
	for _, a := range own.Actions {
		if (strings.HasPrefix(a.ID, "copy-port:") || strings.HasPrefix(a.ID, "copy:")) && a.Run != nil && a.Run.Copy != "" {
			out = append(out, a)
		}
	}
	return out
}

func sshOf(own page.Page) string {
	if own.Footer != nil && own.Footer.Copy != nil {
		return own.Footer.Copy.Text
	}
	return ""
}

// value: the page a row's menu sits on, with that menu: the actions its
// entries name added, its entries as the row's menu.
func withRowMenu(p *page.Page, rowID string, m menu) {
	if len(m.ids) == 0 {
		return
	}
	p.Actions = append(p.Actions, m.added...)
	for i := range p.Sections {
		for j := range p.Sections[i].Items {
			if r := p.Sections[i].Items[j].Row; r != nil && r.ID == rowID {
				r.Menu = m.ids
				return
			}
		}
	}
}

// value: the … menu of a page: its header first, then the entries; the
// actions its entries name added to the page.
func withPageMenu(p *page.Page, m menu) {
	for i := range p.Actions {
		if p.Actions[i].ID == ActionMenu {
			p.Actions[i].Submenu = m.ids
			p.Actions = append(p.Actions, m.added...)
			return
		}
	}
}

// invariant: SPEC A5: a Mac's … menu: Open in Terminal; Restart… (never
// on This Mac, whose Apple menu owns it), Stop everything…, Forget… (never
// This Mac); an asleep Mac's only Wake and Forget…. Its VMs' rows carry
// their own menu.
func macMenus(m Machine, h model.MacView, p page.Page) page.Page {
	if m.Asks == nil {
		return p
	}
	label := model.Label(*m.All, h)
	var pm menu
	pm.title(h.Name, label)
	if model.Reach(h) == model.ReachAsleep {
		if a, ok := pageAction(p, ActionWake); ok {
			pm.add(menuCopy(a, menuID(a.ID, "menu"), "Wake "+label, "alarm"))
		}
	} else {
		if a, ok := pageAction(p, ActionOpenStation); ok {
			pm.add(menuCopy(a, menuID(a.ID, "menu"), "", "terminal"))
		}
		if a, ok := pageAction(p, ActionUnlock); ok {
			pm.add(menuCopy(a, menuID(a.ID, "menu"), "", "lock.open"))
		}
		pm.next()
		pm.own(&p, ActionRestartMac, "Restart", "arrow.clockwise", false)
		pm.own(&p, ActionStopAll, "Stop everything", "stop", true)
	}
	pm.next()
	pm.own(&p, ActionForgetMac, "Forget", "trash", true)
	withPageMenu(&p, pm)
	for _, vm := range h.VMs {
		own := vmPage(m, h, vm)
		withRowMenu(&p, vm.Name, vmMenu(rowOf(p, vm.Name), vm.Name+" on "+label, own))
	}
	return p
}

func rowOf(p page.Page, id string) page.Row {
	for _, s := range p.Sections {
		for _, it := range s.Items {
			if it.Row != nil && it.Row.ID == id {
				return *it.Row
			}
		}
	}
	return page.Row{ID: id}
}

// invariant: SPEC A5: a VM's … menu: its header, then the verbs of its
// page in the menu's order, each group apart; Delete last.
func vmPageMenu(m Machine, h model.MacView, vm model.VM, p page.Page) page.Page {
	if _, ok := pageAction(p, ActionMenu); !ok || m.Asks == nil {
		return p
	}
	key := "menu"
	var pm menu
	pm.title(vm.Name, vm.Name+" on "+model.Label(*m.All, h))
	if a, ok := pageAction(p, ActionNewAgent); ok {
		pm.item(a.ID)
	}
	if ssh := sshOf(p); ssh != "" && vm.Lifetime.Kind() != machine.Server {
		pm.add(page.Action{ID: menuID(ActionCopySSH, key), Label: "Copy ssh command", Icon: "doc.on.doc", On: OnMenu, Run: &page.Run{Copy: ssh}, Hud: hud("Copied " + ssh)})
	}
	if a, ok := pageAction(p, ActionShare); ok && vm.State == model.Running {
		pm.add(menuCopy(a, menuID(a.ID, key), "Share a port", "point.3.connected.trianglepath.dotted"))
	}
	if a, ok := pageAction(p, ActionShareAccount); ok {
		pm.add(menuCopy(a, menuID(a.ID, key), "", "key"))
	}
	pm.next()
	for _, v := range []struct{ id, label, icon string }{{ActionRestart, "", "arrow.clockwise"}, {ActionStartVM, "Start", "play"}, {ActionStop, "", "stop"}} {
		a, ok := pageAction(p, v.id)
		if !ok {
			continue
		}
		a = menuCopy(a, menuID(a.ID, key), v.label, v.icon)
		a.Destructive = a.Confirm != nil && a.Confirm.Danger
		pm.add(a)
	}
	pm.next()
	if a, ok := pageAction(p, ActionDelete); ok {
		a = menuCopy(a, menuID(a.ID, key), "", "trash")
		a.Destructive = a.Confirm != nil
		pm.add(a)
	}
	withPageMenu(&p, pm)
	return page.Seal(p)
}

// invariant: SPEC A5: Home's menus: the station menu (New VM…, Add a
// Mac…, Open station in Terminal; the app's own Open at login, CPU in the
// menu bar and Quit); each server's row as its page offers its verbs;
// each Mac's (Open, New VM on it, Open in Terminal; asleep, Wake); the
// Awake tile's terms.
func homeMenus(r Reading, p page.Page) page.Page {
	var sm menu
	if _, ok := pageAction(p, ActionNewVM); ok {
		sm.item(ActionNewVM)
	}
	if _, ok := pageAction(p, ActionAddMac); ok {
		sm.item(ActionAddMac)
	} else {
		sm.add(page.Action{ID: ActionAddMac, Label: "Add a Mac", On: OnMenu, Sheet: page.SheetHint + RowAddMac})
	}
	if a, ok := pageAction(p, ActionOpenStation); ok {
		sm.add(menuCopy(a, menuID(a.ID, "menu"), "Open station in Terminal", ""))
	}
	sm.next()
	sm.add(page.Action{ID: appLogin, Label: "Open at login", On: OnMenu})
	sm.add(page.Action{ID: appSparkline, Label: "CPU in the menu bar", On: OnMenu})
	sm.next()
	sm.add(page.Action{ID: appQuit, Label: "Quit station", Key: "cmd+q", On: OnMenu})
	p.Actions = append(p.Actions, sm.added...)
	p.Actions = append(p.Actions, page.Action{ID: ActionMenu, Label: "More options for station", Icon: "ellipsis", On: OnMenu, Submenu: sm.ids})
	if r.All == nil {
		return p
	}
	m := Machine{All: r.All, Ports: r.Ports, Shares: r.Accounts, Zone: r.Zone, Now: r.now(), Asks: r.Asks}
	for _, h := range r.All.Macs {
		label := model.Label(*r.All, h)
		for _, vm := range h.VMs {
			if vm.Lifetime != model.AlwaysOn || r.Asks == nil {
				continue
			}
			m.Name = vm.Name
			withRowMenu(&p, vm.Name, vmMenu(rowOf(p, vm.Name), vm.Name+" on "+label, vmPage(m, h, vm)))
		}
		withRowMenu(&p, h.Name, macRowMenu(p, h, label))
	}
	if r.Awake != nil && slices.ContainsFunc(p.Sections, func(s page.Section) bool { return s.ID == SectionTiles }) {
		am := awakeMenu(*r.Awake, r.now(), r.Zone)
		p.Actions = append(p.Actions, am.added...)
		p.Actions = append(p.Actions, page.Action{ID: ActionMenu + ":" + TileAwake, Label: "More options for Awake", On: OnMenu, Submenu: am.ids})
	}
	return p
}

func macRowMenu(p page.Page, h model.MacView, label string) menu {
	var m menu
	m.title(h.Name, label)
	m.add(page.Action{ID: ActionOpenRow + h.Name, Label: "Open", Icon: "arrow.up.forward.square", Key: "enter", On: OnMenu})
	if model.Reach(h) == model.ReachAsleep {
		w := wake(h, label)[0]
		m.add(menuCopy(w, menuID(w.ID, h.Name), "Wake "+label, "alarm"))
		return m
	}
	if a, ok := pageAction(p, ActionNewVMOn+h.Name); ok {
		a = menuCopy(a, menuID(a.ID, "menu"), "New VM on "+label, "plus")
		a.Key = "cmd+n"
		m.add(a)
	}
	open := openIn()
	if !h.ThisMac {
		open = openIn("--on", label)
	}
	m.add(menuCopy(open, menuID(open.ID, h.Name), "", "terminal"))
	return m
}

// value: the Awake tile's menu: This Mac kept awake for each term, until
// the hour, let sleep; then Awake of all Macs.
func awakeMenu(r awake.Report, now time.Time, zone *time.Location) menu {
	var m menu
	m.title(TileAwake, "Keep "+page.ThisMac+" awake")
	for _, a := range keeps(r, now, zone) {
		switch {
		case strings.HasPrefix(a.ID, ActionKeep):
			m.add(menuCopy(a, menuID(a.ID, TileAwake), "For "+a.Label, ""))
		case a.ID == ActionUntil:
			a = menuCopy(a, menuID(a.ID, TileAwake), "", "")
			until := a.Form.Fields[0].Default
			if a.Picked {
				until = moment(r.Session.Until)
			}
			a.Run.In = maps.Clone(a.Run.In)
			a.Run.In["Until"], a.Form = until, nil
			m.add(a)
		}
	}
	m.next()
	for _, a := range keeps(r, now, zone) {
		if a.ID == ActionLetGo {
			m.add(menuCopy(a, menuID(a.ID, TileAwake), "", "moon"))
		}
	}
	m.add(page.Action{ID: ActionOpenRow + TileAwake, Label: "Awake of all Macs", Icon: "sun.max", Key: "cmd+e", On: OnMenu})
	return m
}

// invariant: SPEC A5: a copy lent to a VM: its header (the account lent
// to the VM), Extend by its term, Take back last.
func loanMenu(c accounts.SharedCopy, extend page.Action, levels Levels, self string) menu {
	var m menu
	key := c.Mac + "/" + c.VM + "/" + c.Provider
	label := accounts.Label(c.Provider)
	m.title(key, label+" lent to "+c.VM)
	m.add(menuCopy(extend, menuID(extend.ID, "menu"), VerbExtend+" "+hoursLeft(extendBy), "clock"))
	if levels == nil {
		return m
	}
	m.next()
	ask := ask(levels, "take-back", machine.Workspace, label, "Take back "+label+" from "+c.VM+"?", "take back", "Its copy on "+c.VM+" is deleted.")
	back := page.Action{ID: menuID(ActionTakeBack+"/"+c.Provider, key), Label: "Take back", Icon: "stop", On: OnMenu, Mutates: true, Confirm: ask, Destructive: ask != nil,
		Run: &page.Run{Call: CallTakeBack, In: answered(ask, map[string]string{"Account": c.Provider, "From": c.VM})}}
	if c.Mac != self {
		back.Run.Target = &page.Target{Name: c.Mac}
	}
	m.add(back)
	return m
}

// value: the Ports page's … menu: Share a port… and Declare a port
// public…, with no header.
func portsMenu(p page.Page) page.Page {
	var pm menu
	if a, ok := pageAction(p, ActionShare); ok {
		pm.add(menuCopy(a, menuID(a.ID, "menu"), "", "point.3.connected.trianglepath.dotted"))
	}
	if a, ok := pageAction(p, ActionDeclarePublic); ok {
		pm.add(menuCopy(a, menuID(a.ID, "menu"), "Declare a port public", "globe"))
	}
	withPageMenu(&p, pm)
	return p
}

func terminalItem(a page.Action, key string) page.Action {
	a = menuCopy(a, menuID(a.ID, key), "Open in Terminal", "terminal")
	a.Key = "cmd+enter"
	return a
}
