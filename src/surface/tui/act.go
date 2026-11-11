package tui

import (
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/jterrazz/studio-station/src/core/section"
	"github.com/jterrazz/studio-station/src/page"
	accountsscreen "github.com/jterrazz/studio-station/src/surface/tui/screens/accounts"
	appsscreen "github.com/jterrazz/studio-station/src/surface/tui/screens/apps"
	backgroundscreen "github.com/jterrazz/studio-station/src/surface/tui/screens/background"
	diskscreen "github.com/jterrazz/studio-station/src/surface/tui/screens/disk"
	"github.com/jterrazz/studio-station/src/surface/tui/screens/gesture"
	imagesscreen "github.com/jterrazz/studio-station/src/surface/tui/screens/images"
	matrixscreen "github.com/jterrazz/studio-station/src/surface/tui/screens/matrix"
	overviewscreen "github.com/jterrazz/studio-station/src/surface/tui/screens/overview"
	"github.com/jterrazz/studio-station/src/surface/tui/screens/screen"
	settingsscreen "github.com/jterrazz/studio-station/src/surface/tui/screens/settings"
	sshscreen "github.com/jterrazz/studio-station/src/surface/tui/screens/ssh"
	ui "github.com/jterrazz/studio-station/src/ui/render/tui"
)

// value: how long a success stays in the Progress strip (SPEC §9.1).
const toast = 4 * time.Second

type taskState int

const (
	running taskState = iota
	succeeded
	failedTask
)

// value: one command of this session, from its start to its toast or its
// failure, and the reads it falsifies.
type task struct {
	id    int
	step  gesture.Step
	state taskState
	err   *page.Error
	// value: the page keys the task's machine shows, read again once it ends.
	stale []string
	// value: the commands left of the gesture once this one is done.
	then     []gesture.Step
	n, total int
	// value: the call prints what its toast copies (a VM's ssh command).
	copy bool
	// value: when the call started, for the seconds the strip says it ran.
	began time.Time
}

// value: a question put, the one modal of the shell (SPEC §9.8): yes or no,
// the name typed, or an option picked.
type modal struct {
	question string
	lines    []string
	name     string
	typed    string
	options  []string
	at       int
	pick     func(string) gesture.Gesture
	task     task
	// value: the answers of the form the task's command asked first, held
	// while its question is put, then dropped with the modal.
	values map[string]string
	// value: the question q asks while a call of this session still runs.
	quit bool
}

// value: what the actions of this session hold: its tasks, its modal, the
// ticked rows and what each section's detail pane shows in its list's place.
type acts struct {
	tasks  []task
	next   int
	modal  *modal
	ticked map[string]map[string]bool
	shown  map[string]shown
	// value: the spinner's frame, turned once a second while a call runs.
	frame    int
	spinning bool
}

// value: the strip shows this many calls; the rest are counted on one line
// until tab opens them all (SPEC §9.1).
const stripLines = 3

type shown struct {
	cursor string
	read   string
	title  string
	note   []string
}

type stepped struct {
	t      task
	values map[string]string
	said   []string
	err    error
}

type toasted struct{ id int }

type spun struct{}

func (s shell) pageOf(key string) (page.Page, bool) {
	r, ok := s.reads[key]
	return r.page, ok && r.done && r.err == nil
}

// value: the action the key means on the focused row, from the shown
// page's Actions: a key the page does not name means nothing here.
func (s shell) action(key string) (page.Action, []page.Row, bool) {
	k := s.key(s.current(), s.section())
	p, ok := s.pageOf(k)
	if !ok {
		return page.Action{}, nil, false
	}
	l := s.list(k)
	cur, onRow := s.focused(k)
	for _, a := range p.Actions {
		if a.Key != key {
			continue
		}
		if a.On == page.OnPage {
			return a, nil, true
		}
		var rows []page.Row
		if key != "enter" {
			rows = s.tickedRows(k, l)
		}
		if len(rows) == 0 && onRow {
			rows = []page.Row{cur}
		}
		rows = slices.DeleteFunc(rows, func(r page.Row) bool {
			return strings.HasPrefix(r.ID, "summary/") || len(a.Rows) > 0 && !slices.Contains(a.Rows, r.ID)
		})
		if len(rows) > 0 {
			return a, rows, true
		}
	}
	return page.Action{}, nil, false
}

func (s shell) tickedRows(key string, l ui.List) []page.Row {
	var out []page.Row
	for _, id := range l.Filtered("").Keys() {
		if !s.acts.ticked[key][id] {
			continue
		}
		if r, ok := l.Row(id); ok {
			out = append(out, r)
		}
	}
	return out
}

// invariant: a section's actions are its screen's: each maps the page's
// Action to what it runs, shows, copies or opens.
func (s shell) gesture(a page.Action, rows []page.Row) (gesture.Gesture, bool) {
	p, _ := s.pageOf(s.key(s.current(), s.section()))
	switch s.section() {
	case section.Overview:
		return overviewscreen.Gesture(p, a, rows)
	case section.Apps:
		return appsscreen.Gesture(p, a, rows)
	case section.Settings:
		if s.matrix() {
			_, machine, _ := s.cell(s.key(s.current(), s.section())).Current()
			return matrixscreen.Gesture(a, rows, machine)
		}
		return settingsscreen.Gesture(s.deps.Settings, a, rows)
	case section.Disk:
		return diskscreen.Gesture(p, a, rows)
	case section.Accounts:
		return accountsscreen.Gesture(p, a, rows)
	case section.Background:
		return backgroundscreen.Gesture(p, a, rows)
	case section.Images:
		return imagesscreen.Gesture(s.deps.ThisMachine, a, rows)
	case section.SSH:
		return sshscreen.Gesture(p, a, rows)
	case section.Activity, section.Ports, section.Awake:
	}
	return gesture.Gesture{}, false
}

func (s shell) act(key string) (shell, tea.Cmd, bool) {
	a, rows, ok := s.action(key)
	if !ok {
		return s, nil, false
	}
	g, ok := s.gesture(a, rows)
	if !ok {
		return s, nil, false
	}
	s.acts.ticked = with(s.acts.ticked, s.key(s.current(), s.section()), nil)
	s, cmd := s.perform(g)
	return s, cmd, true
}

func (s shell) perform(g gesture.Gesture) (shell, tea.Cmd) {
	k := s.key(s.current(), s.section())
	cur, _ := s.focused(k)
	switch {
	case g.View != "":
		on := map[string]bool{}
		for v, set := range s.views[k] {
			on[v] = set
		}
		on[g.View] = !on[g.View]
		s.views = with(s.views, k, on)
		return s, nil
	case g.Visit != nil:
		return s.visit(*g.Visit)
	case g.Values != nil:
		f := &screen.Form{Title: g.Values.Title}
		for _, x := range g.Values.Fields {
			f.Fields = append(f.Fields, screen.Field{Label: x.Label, Note: x.Note, Secret: x.Secret})
		}
		s.dialog = &dialog{call: screen.Call{Form: f}, answers: make([]string, len(f.Fields)), values: g.Values}
		return s, nil
	case g.Pick != nil:
		s.acts.modal = &modal{question: g.Pick.Title, options: g.Pick.Options, pick: g.Pick.Then}
		return s, nil
	case g.Read != nil:
		rk := readKey(g.Read)
		s.acts.shown = with(s.acts.shown, k, shown{cursor: cur.ID, read: rk, title: cur.Label})
		s.asked = with(s.asked, rk, false)
		s.focus = ui.PaneDetail
		return s.ask(rk, g.Read)
	case g.Note != nil:
		title := g.Title
		if title == "" {
			title = "Fix"
		}
		s.acts.shown = with(s.acts.shown, k, shown{cursor: cur.ID, title: title, note: g.Note})
		s.focus = ui.PaneDetail
		return s, nil
	case g.Copy != "":
		s = s.toast("Copied " + g.Copy)
		return s, tea.Batch(tea.SetClipboard(g.Copy), s.fade(s.acts.next))
	case g.Open != "":
		return s.outside("Opened "+g.Open, func() error { return s.deps.Open(s.ctx, g.Open) })
	case g.Terminal != "":
		return s.outside("A new terminal runs "+g.Terminal, func() error { return s.deps.Terminal(s.ctx, g.Terminal) })
	}
	if len(g.Steps) == 0 {
		return s, nil
	}
	return s.start(task{step: g.Steps[0], then: g.Steps[1:], n: 1, total: len(g.Steps), stale: s.stale()}, Answer{})
}

func readKey(words []string) string { return "\x00read\x00" + strings.Join(words, " ") }

// value: every page a command on the shown machine may change: its own
// sections, the All machines reads and its Overview's Needs you.
func (s shell) stale() []string {
	n := s.current()
	keys := make([]string, 0, len(s.sections(n))+len(section.AllMachines()))
	for _, id := range s.sections(n) {
		keys = append(keys, s.key(n, id))
	}
	for _, id := range section.AllMachines() {
		keys = append(keys, s.key(s.nodes[0], id))
	}
	return keys
}

func (s shell) toast(text string) shell {
	s.acts.next++
	s.acts.tasks = append(slices.Clone(s.acts.tasks), task{id: s.acts.next, step: gesture.Step{Done: text}, state: succeeded})
	return s
}

func (s shell) fade(id int) tea.Cmd {
	return tea.Tick(toast, func(time.Time) tea.Msg { return toasted{id: id} })
}

func (s shell) outside(done string, run func() error) (shell, tea.Cmd) {
	s.acts.next++
	t := task{id: s.acts.next, step: gesture.Step{Doing: done, Done: done}, n: 1, total: 1, began: time.Now()}
	s.acts.tasks = append(slices.Clone(s.acts.tasks), t)
	return s, func() tea.Msg { return stepped{t: t, err: run()} }
}

func (s shell) start(t task, a Answer) (shell, tea.Cmd) {
	if t.id == 0 {
		s.acts.next++
		t.id = s.acts.next
	}
	t.state, t.began = running, time.Now()
	s.acts.tasks = append(slices.DeleteFunc(slices.Clone(s.acts.tasks), func(x task) bool { return x.id == t.id }), t)
	ctx, do, words := s.ctx, s.deps.Do, t.step.Words
	return s, func() tea.Msg {
		said, err := do(ctx, words, a)
		msg := stepped{t: t, said: said, err: err}
		if _, asked := errors.AsType[*Asked](err); asked {
			msg.values = a.Values
		}
		return msg
	}
}

// value: the form's answers by their input's name, handed to its command;
// the dialog that held them is gone.
func (s shell) handed(v gesture.Values, answers []string) (shell, tea.Cmd) {
	values := make(map[string]string, len(v.Fields))
	for i, f := range v.Fields {
		values[f.Name] = answers[i]
	}
	return s.start(task{step: v.Step, n: 1, total: 1, stale: s.stale()}, Answer{Values: values})
}

func (s shell) stepDone(msg stepped) (tea.Model, tea.Cmd) {
	t := msg.t
	var asked *Asked
	if errors.As(msg.err, &asked) {
		s.acts.tasks = slices.DeleteFunc(slices.Clone(s.acts.tasks), func(x task) bool { return x.id == t.id })
		m := &modal{question: question(t.step), name: asked.Name, task: t, values: msg.values}
		if asked.Name != "" {
			m.lines = []string{asked.Message}
		}
		s.acts.modal = m
		return s, nil
	}
	var cmds []tea.Cmd
	if msg.err != nil {
		t.state, t.err = failedTask, worded(msg.err)
	} else {
		t.state = succeeded
		if len(msg.said) > 0 {
			t.step.Done += ui.Sep + strings.Join(msg.said, ui.Sep)
		}
		cmds = append(cmds, s.fade(t.id))
	}
	s.acts.tasks = slices.Clone(s.acts.tasks)
	for i := range s.acts.tasks {
		if s.acts.tasks[i].id == t.id {
			s.acts.tasks[i] = t
		}
	}
	if len(t.then) > 0 {
		var cmd tea.Cmd
		s, cmd = s.start(task{step: t.then[0], then: t.then[1:], n: t.n + 1, total: t.total, stale: t.stale}, Answer{})
		cmds = append(cmds, cmd)
	}
	if len(t.stale) > 0 && len(t.then) == 0 {
		var cmd tea.Cmd
		s, cmd = s.refresh(t.stale)
		cmds = append(cmds, cmd)
	}
	return s, tea.Batch(cmds...)
}

func (s shell) refresh(keys []string) (shell, tea.Cmd) {
	shownKey := s.key(s.current(), s.section())
	var cmds []tea.Cmd
	for _, k := range keys {
		if _, read := s.reads[k]; !read && k != shownKey {
			continue
		}
		s.asked = with(s.asked, k, false)
	}
	s, cmd := s.load()
	cmds = append(cmds, cmd)
	for _, k := range keys {
		if k == shownKey {
			continue
		}
		if _, read := s.reads[k]; !read {
			continue
		}
		n, id, ok := s.parse(k)
		if !ok {
			continue
		}
		var c tea.Cmd
		s, c = s.ask(k, s.words(n, id))
		cmds = append(cmds, c)
	}
	return s, tea.Batch(cmds...)
}

func (s shell) parse(key string) (node, section.ID, bool) {
	if key == machinesKey {
		return s.nodes[0], section.Overview, true
	}
	name, id, ok := strings.Cut(key, "\x00")
	if !ok {
		return node{}, "", false
	}
	if name == "" {
		return s.nodes[0], section.ID(id), true
	}
	n, found := find(s.nodes, name)
	return n, section.ID(id), found
}

func worded(err error) *page.Error {
	var f *Failed
	if errors.As(err, &f) {
		return &page.Error{Message: f.Message, Hint: f.Hint}
	}
	return &page.Error{Message: err.Error()}
}

func question(st gesture.Step) string {
	if st.Ask != "" {
		return st.Ask
	}
	return "station " + strings.Join(st.Words, " ") + "?"
}

func (s shell) answering(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	m := *s.acts.modal
	key := k.String()
	if key == "esc" {
		s.acts.modal = nil
		return s, nil
	}
	switch {
	case m.pick != nil:
		switch key {
		case "up", "k":
			m.at = max(m.at-1, 0)
		case "down", "j":
			m.at = min(m.at+1, len(m.options)-1)
		case "enter":
			s.acts.modal = nil
			if len(m.options) == 0 {
				return s, nil
			}
			return s.perform(m.pick(m.options[m.at]))
		}
	case m.name != "":
		switch key {
		case "enter":
			if m.typed != m.name {
				return s, nil
			}
			s.acts.modal = nil
			return s.start(m.task, Answer{Typed: m.typed, Values: m.values})
		case "backspace":
			if r := []rune(m.typed); len(r) > 0 {
				m.typed = string(r[:len(r)-1])
			}
		default:
			m.typed += k.Key().Text
		}
	default:
		switch key {
		case "y":
			s.acts.modal = nil
			if m.quit {
				return s.leave()
			}
			return s.start(m.task, Answer{Yes: true, Values: m.values})
		case "n":
			s.acts.modal = nil
			return s, nil
		}
		return s, nil
	}
	s.acts.modal = &m
	return s, nil
}

func (m modal) panel() ui.Panel {
	p := ui.Panel{Title: m.question}
	p.Lines = append(p.Lines, "")
	for _, l := range m.lines {
		p.Lines = append(p.Lines, " "+l)
	}
	switch {
	case m.pick != nil:
		for i, o := range m.options {
			lead := "   "
			if i == m.at {
				lead = " " + ui.Pointer + " "
			}
			p.Lines = append(p.Lines, lead+o)
		}
		if len(m.options) == 0 {
			p.Lines = append(p.Lines, " Nothing to pick.")
		}
	case m.name != "":
		p.Lines = append(p.Lines, " Type "+m.name+" to go on:", " "+ui.Typed+" "+m.typed)
	case m.quit:
		p.Lines = append(p.Lines, " It stops with station, what it finished staying done.", "", " y quit   n keep working")
	default:
		p.Lines = append(p.Lines, " "+m.question)
	}
	return p
}

func (m modal) keys() []ui.Binding {
	switch {
	case m.pick != nil:
		return []ui.Binding{bind("up/down", "move"), bind("enter", "pick"), bind("esc", "cancel")}
	case m.name != "":
		return []ui.Binding{bind("type", "the name"), bind("enter", "go on"), bind("esc", "cancel")}
	}
	if m.quit {
		return []ui.Binding{bind("y", "quit"), bind("n", "keep working"), bind("esc", "keep working")}
	}
	return []ui.Binding{bind("y", "yes"), bind("n", "no"), bind("esc", "no")}
}

func (s shell) running() int {
	n := 0
	for _, t := range s.acts.tasks {
		if t.state == running {
			n++
		}
	}
	return n
}

// invariant: the spinner turns once a second while a call runs, and stops
// with the last one, so an idle screen draws nothing.
func (s shell) spin() (shell, tea.Cmd) {
	if s.acts.spinning || s.running() == 0 {
		return s, nil
	}
	s.acts.spinning = true
	return s, tea.Tick(time.Second, func(time.Time) tea.Msg { return spun{} })
}

func (s shell) turned() (shell, tea.Cmd) {
	s.acts.frame++
	s.acts.spinning = false
	return s.spin()
}

// value: the strip opened to every call once it holds more than it shows.
func (s shell) overflows() bool { return len(s.acts.tasks) > stripLines }

// value: the failure waiting first, the one the strip's keys answer.
func (s shell) waiting() (task, bool) {
	for _, t := range s.acts.tasks {
		if t.state == failedTask {
			return t, true
		}
	}
	return task{}, false
}

// value: the failure's fix as a key: its hint's station command, keyed by
// its verb's first letter (`i install mas`).
func fixOf(t task) (key string, words []string, ok bool) {
	if t.err == nil {
		return "", nil, false
	}
	rest, ours := strings.CutPrefix(t.err.Hint, "station ")
	w := strings.Fields(rest)
	if !ours || len(w) < 2 {
		return "", nil, false
	}
	key = w[1][:1]
	if key == "x" || key == "l" {
		key = "enter"
	}
	return key, w, true
}

func logOf(t task) string {
	if t.err == nil {
		return ""
	}
	for _, w := range strings.Fields(t.err.Message) {
		if strings.Contains(w, "/logs/") {
			return strings.TrimRight(w, ".,;")
		}
	}
	return ""
}

func (t task) keys() []ui.Binding {
	var out []ui.Binding
	if key, w, ok := fixOf(t); ok {
		out = append(out, bind(key, strings.Join(w[1:], " ")))
	}
	if logOf(t) != "" {
		out = append(out, bind("l", "log"))
	}
	return append(out, bind("x", "dismiss"))
}

func (s shell) progress(key string) (tea.Model, tea.Cmd) {
	t, ok := s.waiting()
	if !ok {
		s.focus = ui.PaneMaster
		return s, nil
	}
	fk, w, fixed := fixOf(t)
	switch {
	case key == "x":
		s.acts.tasks = slices.DeleteFunc(slices.Clone(s.acts.tasks), func(x task) bool { return x.id == t.id })
		if _, more := s.waiting(); !more {
			s.focus = ui.PaneMaster
		}
	case key == "l" && logOf(t) != "":
		return s.outside("Opened "+logOf(t), func() error { return s.deps.Open(s.ctx, logOf(t)) })
	case fixed && key == fk:
		s.acts.tasks = slices.DeleteFunc(slices.Clone(s.acts.tasks), func(x task) bool { return x.id == t.id })
		s.focus = ui.PaneMaster
		return s.start(task{step: gesture.Step{Words: w, Doing: doing(w), Done: done(w)}, n: 1, total: 1, stale: t.stale}, Answer{})
	case key == "up" || key == "k" || key == "left" || key == "h":
		s.focus = ui.PaneMaster
	}
	return s, nil
}

func doing(w []string) string { return "Running station " + strings.Join(w, " ") }
func done(w []string) string  { return "Ran station " + strings.Join(w, " ") }

func (s shell) strip() []ui.Task {
	var out []ui.Task
	for _, t := range s.acts.tasks {
		switch t.state {
		case running:
			text := t.step.Doing
			if t.total > 1 {
				text += ui.Sep + strconv.Itoa(t.n) + "/" + strconv.Itoa(t.total)
			}
			if ran := int(time.Since(t.began).Seconds()); !t.began.IsZero() && ran > 0 {
				text += ui.Sep + strconv.Itoa(ran) + " s"
			}
			out = append(out, ui.Task{Mark: page.Awaiting, Text: text, Frame: s.acts.frame})
		case succeeded:
			out = append(out, ui.Task{Mark: page.OK, Text: t.step.Done})
		case failedTask:
			text := t.step.Doing + " failed"
			if t.err != nil && t.err.Message != "" {
				text += ": " + t.err.Message
			}
			out = append(out, ui.Task{Mark: page.Broken, Text: text, Keys: t.keys()})
		}
	}
	if len(out) > stripLines && s.focus != ui.PaneProgress {
		more := len(out) - stripLines
		return append(out[:stripLines], ui.Task{More: true, Text: "+" + strconv.Itoa(more) + " more" + ui.Sep + "tab to open"})
	}
	return out
}

func (s shell) faded(id int) shell {
	s.acts.tasks = slices.DeleteFunc(slices.Clone(s.acts.tasks), func(x task) bool { return x.id == id && x.state == succeeded })
	return s
}

func (s shell) tick() shell {
	k := s.key(s.current(), s.section())
	l := s.list(k)
	if _, _, ok := l.Current(); !ok {
		return s
	}
	marks := map[string]bool{}
	for id, v := range s.acts.ticked[k] {
		marks[id] = v
	}
	id := l.Cursor
	if marks[id] {
		delete(marks, id)
	} else {
		marks[id] = true
	}
	s.acts.ticked = with(s.acts.ticked, k, marks)
	return s
}

// value: the actions the focused row answers, for the footer and the sheet.
func (s shell) actionKeys() []ui.Binding {
	k := s.key(s.current(), s.section())
	p, ok := s.pageOf(k)
	if !ok {
		return nil
	}
	var out []ui.Binding
	own := s.screenKeys()
	for _, a := range p.Actions {
		if slices.ContainsFunc(own, func(b ui.Binding) bool { return b.Key == a.Key }) {
			continue
		}
		if _, rows, ok := s.action(a.Key); ok {
			if _, does := s.gesture(a, rows); does {
				out = append(out, ui.Binding{Key: a.Key, Label: a.Label, Mutates: a.Mutates})
			}
		}
	}
	return out
}

// value: a section ticks rows when one of its page's row actions runs one
// command per row (Apps' install, update, remove).
func (s shell) ticks() bool {
	p, ok := s.pageOf(s.key(s.current(), s.section()))
	if !ok || s.section() != section.Apps {
		return false
	}
	return slices.ContainsFunc(p.Actions, func(a page.Action) bool { return a.On == page.OnRow && a.Mutates })
}

func (s shell) openedPanel(f ui.Shell, o shown, room int) ui.Panel {
	pane := ui.Panel{Title: o.title}
	if o.note != nil {
		for _, l := range o.note {
			pane.Lines = append(pane.Lines, wrap(l, ui.DetailWidth-3)...)
		}
		return pane
	}
	r, ok := s.reads[o.read]
	switch {
	case !ok || !r.done:
		pane.Lines = []string{" " + ui.Glyph(page.Awaiting) + " reading"}
	case r.err != nil:
		pane.Lines = ui.ErrorBlock(*r.err, s.dark)
	default:
		pane.Lines = ui.List{}.Show(r.page).Lines(f, ui.DetailWidth-2, room, false)
	}
	return pane
}

// value: a line of the detail pane cut at its words, never inside one.
func wrap(line string, width int) []string {
	words := strings.Fields(line)
	if len(words) == 0 {
		return []string{""}
	}
	var out []string
	cur := ""
	for _, w := range words {
		if cur != "" && len([]rune(cur))+1+len([]rune(w)) > width {
			out = append(out, " "+cur)
			cur = w
			continue
		}
		if cur != "" {
			cur += " "
		}
		cur += w
	}
	return append(out, " "+cur)
}
