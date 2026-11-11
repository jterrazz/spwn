package tui

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/core/section"
	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/surface/tui/screens/palette"
	"github.com/jterrazz/studio-station/src/surface/tui/screens/screen"
	ui "github.com/jterrazz/studio-station/src/ui/render/tui"
)

type Failed struct{ Message, Hint string }

func (f *Failed) Error() string { return f.Message }

type answered struct {
	key  string
	page page.Page
	err  error
}

type read struct {
	page page.Page
	err  *page.Error
	done bool
}

type shell struct {
	ctx      context.Context
	cancel   context.CancelFunc
	deps     Deps
	width    int
	height   int
	dark     bool
	machines *page.Page
	nodes    []node
	at       string
	sec      int
	focus    ui.Pane
	side     int
	reads    map[string]read
	lists    map[string]ui.List
	asked    map[string]bool
	// value: the CPU percent of each machine at each read of Activity across
	// every machine this session made, the newest last.
	samples   map[string][]int
	filtering bool
	sheet     bool
	acts      acts
	// value: the views a key turned on, by section key (Disk by name).
	views map[string]map[string]bool
	// value: the matrix's cursor, by section key: its row and column.
	cells map[string]ui.Matrix
	// value: the row a section's cursor goes to once its page answers.
	want map[string]string
	// value: each screen's own state, by its read's key.
	states map[string]screen.State
	dialog *dialog
	gens   map[string]int
	held   *held
	sw     *switcher
	seen   visited
	pal    *actionsState
	recent []palette.Item
	last   clicked
}

const machinesKey = "\x00machines"

func newShell(ctx context.Context, d Deps) shell {
	ctx, cancel := context.WithCancel(ctx)
	s := shell{ctx: ctx, cancel: cancel, deps: d, width: 110, height: 30, dark: true, at: d.ThisMachine.Machine,
		focus: ui.PaneMaster, reads: map[string]read{}, lists: map[string]ui.List{}, asked: map[string]bool{}, samples: map[string][]int{},
		states: map[string]screen.State{}, gens: map[string]int{}, held: &held{}}
	s.nodes = tree(d.ThisMachine, nil)
	if d.Opens != "" && d.Opens != d.ThisMachine.Machine {
		s.at = d.Opens
	}
	s.side = s.sectionEntry()
	s.asked[s.key(s.current(), s.section())] = true
	return s
}

func (s shell) Init() tea.Cmd {
	// invariant: the shown section is read first; All machines and the other
	// Macs' Needs you follow once it has answered (SPEC §9.10).
	s.hold()
	return tea.Batch(tea.RequestBackgroundColor, s.fetch(s.key(s.current(), s.section()), s.words(s.current(), s.section())))
}

func (s shell) key(n node, id section.ID) string {
	if n.all() && id == section.Overview {
		return machinesKey
	}
	return n.name + "\x00" + string(id)
}

func (s shell) current() node {
	if n, ok := find(s.nodes, s.at); ok {
		return n
	}
	return s.nodes[0]
}

func (s shell) section() section.ID {
	secs := s.sections(s.current())
	return secs[min(s.sec, len(secs)-1)]
}

func (s shell) ask(key string, w []string) (shell, tea.Cmd) {
	if s.asked[key] {
		return s, nil
	}
	s.asked = with(s.asked, key, true)
	return s, s.fetch(key, w)
}

func (s shell) fetch(key string, w []string) tea.Cmd {
	ctx, pages := s.ctx, s.deps.Pages
	return func() tea.Msg {
		p, err := pages(ctx, w)
		return answered{key: key, page: p, err: err}
	}
}

func (s shell) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m, cmd := s.update(msg)
	next, ok := m.(shell)
	if !ok {
		return m, cmd
	}
	next.hold()
	next, spin := next.spin()
	return next, tea.Batch(cmd, spin)
}

func (s shell) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if msg.Width > 0 && msg.Height > 0 {
			s.width, s.height = msg.Width, msg.Height
		}
	case tea.BackgroundColorMsg:
		s.dark = msg.IsDark()
	case answered:
		return s.answer(msg)
	case stepped:
		return s.stepDone(msg)
	case toasted:
		return s.faded(msg.id), nil
	case spun:
		return s.turned()
	case called:
		return s.finished(msg)
	case ticked:
		return s.relive(msg)
	case tea.KeyPressMsg:
		return s.press(msg)
	case tea.MouseClickMsg:
		return s.click(msg.Mouse())
	case tea.MouseWheelMsg:
		return s.wheel(msg.Mouse())
	}
	return s, nil
}

func (s shell) answer(msg answered) (tea.Model, tea.Cmd) {
	r := read{page: msg.page, done: true}
	if msg.err != nil {
		var f *Failed
		if errors.As(msg.err, &f) {
			r.err = &page.Error{Message: f.Message, Hint: f.Hint}
		} else {
			r.err = &page.Error{Message: msg.err.Error()}
		}
	}
	s.reads = with(s.reads, msg.key, r)
	var cmds []tea.Cmd
	if msg.key == machinesKey && r.err == nil {
		p := msg.page
		before := s.entries()[min(s.side, len(s.entries())-1)]
		s.machines = &p
		s.nodes = tree(s.deps.ThisMachine, s.machines)
		s.side = s.sectionEntry()
		if !before.isSection {
			for i, e := range s.entries() {
				if !e.isSection && !e.Heading && e.machine == before.machine && e.Label != "" {
					s.side = i
				}
			}
		}
		for _, n := range s.nodes {
			if n.kind == machine.Computer && !n.thisMac {
				var cmd tea.Cmd
				s, cmd = s.ask(s.key(n, section.Overview), s.words(n, section.Overview))
				cmds = append(cmds, cmd)
			}
		}
		var cmd tea.Cmd
		s, cmd = s.load()
		cmds = append(cmds, cmd)
	}
	if msg.key == s.key(s.nodes[0], section.Activity) && r.err == nil {
		s.samples = sampled(s.samples, msg.page)
	}
	if l, ok := s.lists[msg.key]; ok || r.err == nil {
		l.Skip = s.skips(msg.key)
		s.lists = with(s.lists, msg.key, l.Show(r.page))
	}
	s = s.reveal(msg.key)
	var cmd tea.Cmd
	s, cmd = s.ask(machinesKey, []string{"machines"})
	cmds = append(cmds, cmd)
	s, cmd = s.live(msg.key)
	return s, tea.Batch(append(cmds, cmd)...)
}

func (s shell) press(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := k.String()
	if t := k.Key().Text; len(t) == 1 && t != " " && strings.HasPrefix(key, "shift+") {
		key = t
	}
	if key == "ctrl+c" {
		return s.quit()
	}
	if s.sheet {
		s.sheet = false
		return s, nil
	}
	if s.sw != nil {
		return s.switching(k)
	}
	if s.pal != nil {
		return s.choosing(k)
	}
	if s.dialog != nil {
		return s.asking(k)
	}
	if s.acts.modal != nil {
		return s.answering(k)
	}
	if s.filtering {
		return s.typing(k)
	}
	if s.focus == ui.PaneProgress && key != "tab" && key != "shift+tab" && key != "q" && key != "?" {
		return s.progress(key)
	}
	switch key {
	case "q":
		return s.quit()
	case "?":
		s.sheet = true
	case "tab":
		s.focus = s.cycle(1)
	case "shift+tab":
		s.focus = s.cycle(-1)
	case "[":
		return s.show(s.sec - 1)
	case "]":
		return s.show(s.sec + 1)
	case "ctrl+r":
		return s.again()
	case "esc":
		return s.back()
	case "n":
		return s.create()
	case "m":
		s.sw = &switcher{}
	case "ctrl+p":
		return s.actions()
	case "-":
		return s.previous()
	default:
		if s.focus == ui.PaneSidebar {
			return s.sidebar(key)
		}
		if n, ok := s.unanswered(); ok {
			if pressed, cmd, ok := s.unansweredPress(n, key); ok {
				return pressed, cmd
			}
		}
		if sc := s.screen(); sc != nil && s.read() {
			if e, ok := sc.Press(key, s.view()); ok {
				return s.effect(e)
			}
		}
		return s.master(key)
	}
	return s, nil
}

func (s shell) read() bool {
	r := s.reads[s.key(s.current(), s.section())]
	return r.done && r.err == nil
}

// invariant: SPEC §9.3: q asks before it ends a call this session still
// runs; ctrl+c over that question quits.
func (s shell) quit() (tea.Model, tea.Cmd) {
	if n := s.running(); n > 0 && (s.acts.modal == nil || !s.acts.modal.quit) {
		calls := "1 call still runs"
		if n > 1 {
			calls = strconv.Itoa(n) + " calls still run"
		}
		s.dialog = nil
		s.acts.modal = &modal{question: calls + " here: quit anyway?", quit: true}
		return s, nil
	}
	return s.leave()
}

func (s shell) leave() (tea.Model, tea.Cmd) {
	s.cancel()
	return s, tea.Quit
}

func (s shell) typing(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := s.key(s.current(), s.section())
	l := s.list(key)
	switch k.String() {
	case "esc":
		s.filtering = false
		l = l.Filtered("")
	case "enter":
		s.filtering = false
	case "backspace":
		if r := []rune(l.Filter); len(r) > 0 {
			l = l.Filtered(string(r[:len(r)-1]))
		}
	default:
		if t := k.Key().Text; t != "" {
			l = l.Filtered(l.Filter + t)
		}
	}
	s.lists = with(s.lists, key, l)
	return s, nil
}

func (s shell) back() (tea.Model, tea.Cmd) {
	key := s.key(s.current(), s.section())
	if s.focus == ui.PaneDetail {
		s.focus = ui.PaneMaster
		s.acts.shown = with(s.acts.shown, key, shown{})
		return s, nil
	}
	if l := s.list(key); l.Filter != "" {
		s.lists = with(s.lists, key, l.Filtered(""))
		return s, nil
	}
	n := s.current()
	switch {
	case n.all():
		return s, nil
	case n.kind == machine.Computer:
		return s.open("")
	default:
		return s.open(n.mac)
	}
}

func (s shell) open(name string) (tea.Model, tea.Cmd) {
	s.seen = s.seen.left(s.current().name, name)
	s.at, s.sec = name, 0
	s.side = s.sectionEntry()
	if s.focus == ui.PaneDetail {
		s.focus = ui.PaneMaster
	}
	return s.load()
}

func (s shell) show(i int) (tea.Model, tea.Cmd) {
	n := len(s.sections(s.current()))
	s.sec = (i%n + n) % n
	s.side = s.sectionEntry()
	if s.focus == ui.PaneDetail && !s.detailed() {
		s.focus = ui.PaneMaster
	}
	return s.load()
}

func (s shell) sidebar(key string) (tea.Model, tea.Cmd) {
	entries := s.entries()
	switch key {
	case "up", "k", "down", "j":
		step := 1
		if key == "up" || key == "k" {
			step = -1
		}
		for i := s.side + step; i >= 0 && i < len(entries); i += step {
			if entries[i].selectable() {
				s.side = i
				break
			}
		}
		if e := entries[s.side]; e.isSection {
			return s.show(e.sec)
		}
	case "right", "l":
		s.focus = ui.PaneMaster
	case "enter":
		e := entries[s.side]
		if e.isSection {
			s.focus = ui.PaneMaster
			return s, nil
		}
		return s.open(e.machine)
	}
	return s, nil
}

func (s shell) master(key string) (tea.Model, tea.Cmd) {
	k := s.key(s.current(), s.section())
	l := s.list(k)
	if key == "space" {
		return s.tick(), nil
	}
	if _, folded := l.Unfold(); key != "enter" || s.matrix() || (!folded && !s.table()) {
		if acted, cmd, ok := s.act(key); ok {
			return acted, cmd
		}
	}
	if s.matrix() && s.focus == ui.PaneMaster {
		steps := map[string][2]int{"up": {-1, 0}, "k": {-1, 0}, "down": {1, 0}, "j": {1, 0}, "pgup": {-10, 0}, "pgdown": {10, 0},
			"left": {0, -1}, "h": {0, -1}, "right": {0, 1}, "l": {0, 1}}
		if step, ok := steps[key]; ok {
			moved, inside := s.moveCell(step[0], step[1])
			if !inside && s.sidebarShown() {
				moved.focus = ui.PaneSidebar
			}
			return moved, nil
		}
	}
	switch key {
	case "up", "k":
		l = l.Move(-1)
		s.acts.shown = with(s.acts.shown, k, shown{})
	case "down", "j":
		l = l.Move(1)
		s.acts.shown = with(s.acts.shown, k, shown{})
	case "pgup":
		l = l.Move(-10)
	case "pgdown":
		l = l.Move(10)
	case "left", "h":
		if s.focus == ui.PaneDetail {
			s.focus = ui.PaneMaster
		} else if s.sidebarShown() {
			s.focus = ui.PaneSidebar
		}
	case "right", "l":
		if s.detailed() {
			s.focus = ui.PaneDetail
		}
	case "enter":
		if name, ok := s.opened(); ok && s.table() {
			return s.open(name)
		}
		var opened bool
		if l, opened = l.Unfold(); !opened && s.detailed() {
			s.focus = ui.PaneDetail
		}
	case "/":
		if s.listed() {
			s.filtering = true
		}
	}
	s.lists = with(s.lists, k, l)
	return s, nil
}

func (s shell) sidebarShown() bool { return s.width >= ui.SidebarFrom }

func (s shell) listed() bool {
	if s.table() {
		return true
	}
	switch s.section() {
	case section.Activity:
		return s.current().all()
	case section.Overview, section.SSH:
		return false
	default:
		return true
	}
}

func (s shell) detailed() bool {
	return (s.listed() && !s.table() && !s.matrix() && s.section() != section.Awake && (s.section() != section.Activity || !s.current().all())) || s.detailShown() != nil
}

// value: what the section's detail pane shows in place of the row's own
// detail, while its row has the cursor.
func (s shell) detailShown() *shown {
	k := s.key(s.current(), s.section())
	o, ok := s.acts.shown[k]
	if !ok || (o.read == "" && o.note == nil) || o.cursor != s.cursorRow(k) {
		return nil
	}
	return &o
}

func (s shell) cursorRow(key string) string {
	r, _ := s.focused(key)
	return r.ID
}

func (s shell) masterWidth() int {
	width := s.width - 2
	if s.sidebarShown() {
		width -= ui.SidebarWidth
	}
	if ((s.detailed() && !s.wide()) || s.paned()) && s.width >= ui.ThreePanes {
		width -= ui.DetailWidth
	}
	return width
}

// value: the sections the screen a read is for keeps its cursor off.
func (s shell) skips(key string) func(page.Section) bool {
	for _, n := range s.nodes {
		for _, id := range s.sections(n) {
			if s.key(n, id) != key {
				continue
			}
			if sk, ok := screenFor(n, id).(screen.Skips); ok {
				m := s.machineOf(n)
				return func(p page.Section) bool { return sk.Skips(m, p) }
			}
			return nil
		}
	}
	return nil
}

// value: the shown screen draws one wide pane, its detail opened over it.
func (s shell) wide() bool {
	w, ok := s.screen().(screen.Wide)
	return ok && w.Wide(s.machineOf(s.current()))
}

func (s shell) page(n node, id section.ID) (page.Page, bool) {
	r, ok := s.reads[s.key(n, id)]
	return r.page, ok && r.done && r.err == nil
}

// invariant: a machine's sections are those its own Overview has a line
// for: the machine gates each read by its capabilities, and its Overview
// leaves out what it refused (a Linux workspace has no job manager, so no
// Background). Its kind's table stands in until that Overview answers, and
// while the machine answers no read (a stopped VM) (SPEC §3.2, round 13 E1).
func (s shell) sections(n node) []section.ID {
	if n.all() {
		return section.AllMachines()
	}
	kind := section.For(n.kind)
	p, ok := s.page(n, section.Overview)
	if !ok {
		return kind
	}
	var lines []section.ID
	for _, sec := range p.Sections {
		for _, it := range sec.Items {
			if it.Row == nil {
				continue
			}
			if id := section.ID(it.Row.ID); id != section.Overview && slices.Contains(kind, id) {
				lines = append(lines, id)
			}
		}
	}
	if len(lines) == 0 {
		return kind
	}
	return slices.DeleteFunc(kind, func(id section.ID) bool { return id != section.Overview && !slices.Contains(lines, id) })
}

func sampled(samples map[string][]int, p page.Page) map[string][]int {
	out := make(map[string][]int, len(samples))
	for k, v := range samples {
		out[k] = v
	}
	for _, sec := range p.Sections {
		if sec.ID != "machines" {
			continue
		}
		for _, it := range sec.Items {
			if it.Row == nil {
				continue
			}
			pct, err := strconv.Atoi(strings.TrimSpace(strings.TrimSuffix(ui.Plain(column(sec, *it.Row, "cpu")), "%")))
			if err != nil {
				continue
			}
			out[it.Row.ID] = append(slices.Clone(out[it.Row.ID]), pct)[max(len(out[it.Row.ID])+1-sparkPoints, 0):]
		}
	}
	return out
}

func (s shell) cycle(step int) ui.Pane {
	var panes []ui.Pane
	if s.sidebarShown() {
		panes = append(panes, ui.PaneSidebar)
	}
	panes = append(panes, ui.PaneMaster)
	if s.detailed() {
		panes = append(panes, ui.PaneDetail)
	}
	if _, waits := s.waiting(); waits || s.overflows() {
		panes = append(panes, ui.PaneProgress)
	}
	i := max(slices.Index(panes, s.focus), 0)
	return panes[((i+step)%len(panes)+len(panes))%len(panes)]
}

type entry struct {
	ui.Entry
	machine   string
	isSection bool
	sec       int
}

func (e entry) selectable() bool { return !e.Heading && e.Label != "" }

func (s shell) entries() []entry {
	out := []entry{{Entry: ui.Entry{Label: "MACHINES", Heading: true}}}
	out = append(out, entry{Entry: ui.Entry{Label: allMachines, AllMachines: true}, machine: ""})
	var kind machine.Kind
	vms := s.nodes[1:]
	for i, n := range vms {
		if n.kind == machine.Computer {
			e := entry{Entry: ui.Entry{Label: n.label, Mark: n.mark}, machine: n.name}
			if c, ok := s.needs(n); ok && c > 0 {
				e.Badge, e.Tone = strconv.Itoa(c), page.Unjudged
			}
			out = append(out, e)
			kind = ""
			continue
		}
		if n.kind != kind {
			kind = n.kind
			out = append(out, entry{Entry: ui.Entry{Label: map[machine.Kind]string{machine.Server: "Servers", machine.Workspace: "Workspaces"}[n.kind], Heading: true, Depth: 1}})
		}
		last := i+1 >= len(vms) || vms[i+1].kind != n.kind || vms[i+1].mac != n.mac
		label := n.label
		if slices.Contains(n.facts, "Always on") {
			label += " always on"
		}
		out = append(out, entry{Entry: ui.Entry{Label: label, Mark: n.mark, Depth: 1, Branch: true, Last: last}, machine: n.name})
	}
	cur := s.current()
	out = append(out, entry{}, entry{Entry: ui.Entry{Label: cur.heading(), Heading: true}})
	for i, id := range s.sections(cur) {
		badge, tone := s.badge(cur, id)
		out = append(out, entry{Entry: ui.Entry{Label: id.Title(), Badge: badge, Tone: tone, Current: i == s.sec}, isSection: true, sec: i})
	}
	return out
}

func (s shell) sectionEntry() int {
	for i, e := range s.entries() {
		if e.isSection && e.sec == s.sec {
			return i
		}
	}
	return 0
}

func (s shell) needs(n node) (int, bool) {
	r, ok := s.reads[s.key(n, section.Overview)]
	if !ok || !r.done || r.err != nil {
		return 0, false
	}
	for _, sec := range r.page.Sections {
		if sec.ID == "needs-you" {
			c := 0
			for _, it := range sec.Items {
				if it.Row != nil {
					c++
				}
			}
			return c, true
		}
	}
	return 0, true
}

// invariant: a section's badge is its worst mark, or Overview's Needs-you
// count: read from the section's own page once read, else its Overview line.
func (s shell) badge(n node, id section.ID) (string, page.Mark) {
	if id == section.Overview && !n.all() {
		if c, ok := s.needs(n); ok {
			if c == 0 {
				return "", page.Unjudged
			}
			return strconv.Itoa(c), page.Unjudged
		}
		return ui.Glyph(page.Awaiting), page.Awaiting
	}
	if r, ok := s.reads[s.key(n, id)]; ok && r.done {
		if r.err != nil {
			return ui.Glyph(page.Broken), page.Broken
		}
		return worst(r.page)
	}
	if n.all() {
		return "", page.Unjudged
	}
	ov, ok := s.reads[s.key(n, section.Overview)]
	if !ok || !ov.done {
		return ui.Glyph(page.Awaiting), page.Awaiting
	}
	for _, sec := range ov.page.Sections {
		for _, it := range sec.Items {
			if it.Row == nil || it.Row.ID != string(id) {
				continue
			}
			m := page.Unjudged
			for _, c := range it.Row.Cells {
				m = graver(m, page.Mark(c.Label))
				for _, v := range c.Items {
					m = graver(m, page.Mark(v.Label))
				}
			}
			if id == section.Awake && m == page.Unjudged {
				return "", m
			}
			if m == page.Broken || m == page.Attention || id == section.Awake {
				return ui.Glyph(m), m
			}
			return "", page.Unjudged
		}
	}
	return "", page.Unjudged
}

func worst(p page.Page) (string, page.Mark) {
	m := page.Unjudged
	var walk func(items []page.Item)
	walk = func(items []page.Item) {
		for _, it := range items {
			if it.Row != nil {
				m = graver(m, it.Row.Mark)
			}
			if it.Group != nil {
				m = graver(m, it.Group.Head.Mark)
				walk(it.Group.Items)
			}
		}
	}
	for _, sec := range p.Sections {
		walk(sec.Items)
	}
	if m == page.Broken || m == page.Attention {
		return ui.Glyph(m), m
	}
	return "", page.Unjudged
}

func graver(a, b page.Mark) page.Mark {
	rank := func(m page.Mark) int {
		switch m {
		case page.Broken:
			return 3
		case page.Attention:
			return 2
		case page.OK:
			return 1
		default:
			return 0
		}
	}
	if rank(b) > rank(a) {
		return b
	}
	return a
}

func (s shell) crumbs() []string {
	n := s.current()
	out := []string{allMachines}
	if n.all() {
		if s.section() != section.Overview {
			out = append(out, s.section().Title())
		}
		return out
	}
	if n.mac != "" {
		if mac, ok := find(s.nodes, n.mac); ok {
			out = append(out, mac.label)
		}
	}
	return append(out, n.label, s.section().Title())
}

func (s shell) header() ui.Header {
	n := s.current()
	if n.all() {
		n = s.nodes[1]
		for _, x := range s.nodes {
			if x.thisMac {
				n = x
			}
		}
	}
	h := ui.Header{Mark: n.mark}
	if n.kind == machine.Computer {
		h.Parts = []string{n.label, n.name}
		if n.engine != "" {
			h.Parts = append(h.Parts, n.engine)
		}
		if awake := s.awake(n); awake != "" {
			h.Parts = append(h.Parts, "Awake "+awake)
		}
		h.Drop = []int{1, 3, 2}
		return h
	}
	h.Parts = append([]string{n.label}, n.facts...)
	return h
}

func (s shell) awake(n node) string {
	r, ok := s.reads[s.key(n, section.Overview)]
	if !ok || !r.done {
		return ""
	}
	for _, sec := range r.page.Sections {
		for _, it := range sec.Items {
			if it.Row == nil || it.Row.ID != string(section.Awake) || len(it.Row.Cells) == 0 {
				continue
			}
			c := it.Row.Cells[0]
			if len(c.Items) > 0 {
				c = c.Items[0]
			}
			if c.Text == "sleeps normally" {
				return "off"
			}
			if c.Text != "" {
				return c.Text
			}
			return ui.Plain(c)
		}
	}
	return ""
}

func (s shell) View() tea.View {
	v := tea.NewView(s.frame().Render())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	v.WindowTitle = "station"
	return v
}

func (s shell) frame() ui.Shell {
	f := ui.Shell{Width: s.width, Height: s.height, Dark: s.dark, NoColor: s.deps.NoColor, Header: s.header(), Crumbs: s.crumbs(), Focus: s.focus}
	f.ShowSidebar = s.sidebarShown()
	entries := s.entries()
	for _, e := range entries {
		f.Sidebar = append(f.Sidebar, e.Entry)
	}
	f.Cursor = s.side
	key := s.key(s.current(), s.section())
	l, r := s.list(key), s.reads[key]
	title := l.Title(s.section().Title()) + ui.Sep + s.current().label
	if words := s.viewTitle(key); words != "" {
		title += ui.Sep + words
	}
	l.Bound = s.bound()
	sc := s.screen()
	if sc != nil && r.done && r.err == nil {
		title = sc.Title(s.view())
		if l.Filter != "" {
			title += " (/" + l.Filter + ")"
		}
	}
	room := max(s.height-5, 1)
	width := s.masterWidth()
	detail := s.detailed() && ((s.width >= ui.ThreePanes && !s.wide()) || s.focus == ui.PaneDetail)
	switch {
	case !r.done:
		f.Master = ui.Panel{Title: title, Lines: []string{" " + ui.Glyph(page.Awaiting) + " " + s.section().Title() + ui.Sep + "reading " + s.current().label}}
	case r.err != nil:
		f.Master = ui.Panel{Title: title, Lines: ui.ErrorBlock(*r.err, s.dark)}
		if n, ok := s.unanswered(); ok {
			f.Master.Lines = s.unansweredLines(n)
		}
	case s.matrix():
		m := s.cell(key)
		f.Master = ui.Panel{Title: m.Title(s.section().Title(), s.current().label), Lines: m.Lines(f, width, room, s.focus == ui.PaneMaster || s.focus == ui.PaneDetail)}
	default:
		l.Ticked = s.acts.ticked[key]
		f.Master = ui.Panel{Title: title, Lines: l.Lines(f, width, room, s.focus == ui.PaneMaster || s.focus == ui.PaneDetail)}
		if sc != nil {
			if lines, ok := sc.Lines(s.view()); ok {
				f.Master.Lines = lines
			}
		}
		// invariant: a narrow terminal draws the fixed panes under the
		// master pane's lines, never drops one.
		if panes := s.panes(); s.width >= ui.ThreePanes {
			f.Beside = panes
		} else {
			for _, p := range panes {
				f.Master.Lines = append(append(f.Master.Lines, "", " "+p.Title), p.Lines...)
			}
		}
	}
	if detail {
		pane := ui.Panel{Title: "Detail"}
		if r.done && r.err == nil {
			if _, row, ok := l.Current(); ok {
				pane.Title = row.Label
			}
			pane.Lines = l.Detail(s.dark, ui.DetailWidth-2)
		}
		if o := s.detailShown(); o != nil {
			pane = s.openedPanel(f, *o, room)
		}
		f.Detail, f.Overlay = &pane, s.width < ui.ThreePanes || s.wide()
	}
	f.Progress = s.strip()
	if s.acts.modal != nil {
		p := s.acts.modal.panel()
		f.Modal = &p
	}
	f.Footer = s.footer()
	if s.sheet {
		f.Sheet = s.keySheet()
	}
	f.Dialog = s.dialogPanel()
	if p := s.switcherPanel(); p != nil {
		f.Dialog = p
	}
	if p := s.actionsPanel(); p != nil {
		f.Dialog = p
	}
	return f
}

func with[K comparable, V any](m map[K]V, k K, v V) map[K]V {
	out := make(map[K]V, len(m)+1)
	for key, val := range m {
		out[key] = val
	}
	out[k] = v
	return out
}
