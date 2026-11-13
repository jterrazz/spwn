package menubar

import (
	"slices"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

// value: the actions of New VM's sheet besides new-vm itself: a workspace
// on a Mac, a server on a Mac, and each one's Create and Open in Terminal.
const (
	ActionNewServerOn = "new-server:"
	ActionCreateOpen  = "open-new:"

	CallCreateServer = "machines.create-server"

	keyOn, keyKind, keyShare, keyShareFor = "On", "Kind", "Share", "ShareFor"
	keyOptions, keySharing                = "Options", "Sharing"
	soon                                  = "Ready in about a minute."
)

// value: the memory a new VM may ask, in GB: the presets.
func memoryStops() []int {
	out := make([]int, 0, len(model.Presets()))
	for _, p := range model.Presets() {
		out = append(out, int(p.MemoryBytes()>>30))
	}
	return out
}

// invariant: SPEC A5 New VM: the engine builds the sheet whole, one form
// per Mac and per kind, so every word it says (the memory a Mac has free,
// the accounts it can share, the name the VM takes) is the engine's; On and
// Kind swap between those forms, which the page carries side by side.
type vmSheet struct {
	all     model.AllMachines
	signed  accounts.Reading
	self    string
	primary string
	roomy   bool
	held    []string
}

// value: New VM's sheets over the fleet as read: new-vm on primary (the
// roomiest Mac when roomy, which the subtitle says), then a workspace and
// a server on every Mac awake, with their split items.
func newVMSheets(all *model.AllMachines, shares *accounts.AllMachines, self, primary string, roomy bool) []page.Action {
	if all == nil {
		return nil
	}
	s := vmSheet{all: *all, self: self, primary: primary, roomy: roomy, held: model.Held(*all)}
	if shares != nil {
		s.signed = shares.Read()
	}
	var out []page.Action
	for _, h := range s.macs() {
		if h.Name == primary {
			out = append(out, s.workspace(h, ActionNewVM)...)
		}
	}
	for _, h := range s.macs() {
		out = append(out, s.workspace(h, ActionNewVMOn+h.Name)...)
		out = append(out, s.server(h)...)
	}
	return out
}

func (s vmSheet) macs() []model.MacView {
	var out []model.MacView
	for _, h := range s.all.Macs {
		if !h.Asleep {
			out = append(out, h)
		}
	}
	return out
}

func (s vmSheet) workspaceID(mac string) string {
	if mac == s.primary {
		return ActionNewVM
	}
	return ActionNewVMOn + mac
}

func (s vmSheet) subtitle(h model.MacView, id string) string {
	name := model.Label(s.all, h)
	if id == ActionNewVM && s.roomy {
		return "On " + name + ", the Mac with the most free memory. " + soon
	}
	return "On " + name + ". " + soon
}

// value: On: every Mac of the fleet with its memory free, an asleep one
// there but not to pick; picking one opens its own form.
func (s vmSheet) on(h model.MacView, id func(string) string) page.Field {
	f := page.Field{Key: keyOn, Label: "On", Kind: page.FieldSelect, Swaps: true, Default: id(h.Name)}
	for _, m := range s.all.Macs {
		o := page.Option{Label: model.Label(s.all, m)}
		if !m.Asleep {
			o.Value = id(m.Name)
			o.Note, _, _ = free(m)
		} else {
			o.Disabled = "asleep"
		}
		f.Options = append(f.Options, o)
	}
	return f
}

func (s vmSheet) kind(h model.MacView, picked string) page.Field {
	return page.Field{Key: keyKind, Label: "Kind", Kind: page.FieldSegmented, Swaps: true, Default: picked, Options: []page.Option{
		{Value: s.workspaceID(h.Name), Label: "Workspace"}, {Value: ActionNewServerOn + h.Name, Label: "Server"},
	}}
}

func system(key string) page.Field {
	return page.Field{Key: key, Label: "System", Kind: page.FieldSegmented, Default: "station-linux", Options: []page.Option{{Value: "station-linux", Label: "Linux"}, {Value: "macos", Label: "macOS"}}}
}

// invariant: SPEC A9 1-8: every stop labeled, the part past the Mac's
// free memory hatched from where it truly falls, a stop past it refused
// with why, and the helper said in words.
func (s vmSheet) memory(h model.MacView, key string) page.Field {
	name := model.Label(s.all, h)
	_, room, known := free(h)
	gb := int(room >> 30)
	f := page.Field{Key: key, Label: "Memory", Kind: page.FieldSlider, Stops: memoryStops(), Default: gigabytes(model.PresetWorkspace.MemoryBytes() >> 30)}
	if known {
		f.Limit, f.LimitNote = gb, "Over "+strconv.Itoa(gb)+" GB would squeeze "+name
	}
	for _, n := range memoryStops() {
		o := page.Option{Value: gigabytes(int64(n)), Label: strconv.Itoa(n) + " GB"}
		if known && n > gb {
			o.Disabled = "only " + strconv.Itoa(gb) + " GB free on " + name
		}
		f.Options = append(f.Options, o)
	}
	return f
}

func (s vmSheet) name(key, note string) page.Field {
	held := func(n string) bool { return !slices.Contains(s.held, n) }
	return page.Field{Key: key, Label: "Name", Kind: page.FieldText, Default: model.FirstFree("linux", held), Note: note,
		Taken: s.held, TakenHud: answer(key) + " already exists"}
}

// invariant: SPEC A9 1-9: the accounts the Mac is signed in to, checked,
// the ones it is out of there but refused with why; Share for under
// them, only while one is checked.
func (s vmSheet) sharing(h model.MacView) page.Field {
	name := model.Label(s.all, h)
	chips := page.Field{Key: keyShare, Kind: page.FieldMulti}
	var checked []string
	for _, on := range s.signed.SignedIn {
		if on.Mac != h.Name {
			continue
		}
		for _, a := range on.Accounts {
			if _, ok := accounts.SecretOf(a.Provider); ok {
				chips.Options = append(chips.Options, page.Option{Value: a.Provider, Label: accounts.Label(a.Provider)})
				checked = append(checked, a.Provider)
			}
		}
	}
	for _, out := range s.signed.SignedOut {
		if _, ok := accounts.SecretOf(out.Provider); ok && out.Mac == h.Name {
			label := accounts.Label(out.Provider)
			chips.Options = append(chips.Options, page.Option{Value: out.Provider, Label: label, Disabled: label + ": sign in on " + name + " first"})
		}
	}
	chips.Default = strings.Join(checked, ",")
	shareFor := page.Field{Key: keyShareFor, Label: "Share for", Kind: page.FieldSegmented, Default: shareFor()[0].Value, Options: shareFor(),
		When: &page.When{Key: keyShare, Set: true}}
	return page.Field{Key: keySharing, Label: "Share", Kind: page.FieldDisclosure, Summary: page.Text("{" + keyShare + ":, |No account}[ for " + answer(keyShareFor) + "]"),
		Fields: []page.Field{chips, shareFor}}
}

func hudSaying(words string) *page.Value {
	v := page.Text(words)
	return &v
}

func create() []page.Primary { return []page.Primary{{Label: "Create"}} }

func answer(key string) string { return "{" + key + "}" }

// value: the glyphs a sheet's words hold, named for the kit to draw: its
// separator and its ellipsis.
const (
	answerSep      = "{sep}"
	answerEllipsis = "{ellipsis}"
)

func (s vmSheet) workspace(h model.MacView, id string) []page.Action {
	name := model.Label(s.all, h)
	const (
		lifetime, image, memory, vm = "Request.Lifetime", "Request.Image", "Request.Size.MemoryBytes", "Request.Name"
	)
	options := page.Field{Key: keyOptions, Label: "Options", Kind: page.FieldDisclosure,
		Summary: page.Texts(answer(lifetime), answer(image), answer(memory)),
		Note:    answer(vm) + answerSep + "shares {" + keyShare + ": and |no account}[ for " + answer(keyShareFor) + "]",
		Fields: []page.Field{
			s.kind(h, id),
			{Key: lifetime, Label: "Lifetime", Kind: page.FieldSegmented, Default: string(model.Ephemeral), Options: []page.Option{
				{Value: string(model.Ephemeral), Label: "Ephemeral"}, {Value: string(model.Persistent), Label: "Persistent"},
			}},
			system(image), s.memory(h, memory), s.sharing(h), s.name(vm, "Generated, editable"),
		}}
	form := &page.Form{Title: "New VM", Subtitle: s.subtitle(h, id), Split: []string{ActionCreateOpen + id}, Primary: create(),
		Fields: []page.Field{s.on(h, s.workspaceID), options}}
	run := &page.Run{Call: CallCreate, In: map[string]string{"Request.Network": "nat"}, Target: macTarget(h.Name, s.self)}
	a := page.Action{ID: id, Label: "New VM", Icon: "plus", On: page.OnPage, Priority: 1, Mutates: true, Sheet: page.SheetNewVM, Form: form, Run: run,
		Hud: hudSaying("Creating " + answer(vm) + " on " + name + answerEllipsis)}
	if id == ActionNewVM {
		a.Key = "cmd+n"
	}
	if id == ActionNewVM && s.roomy {
		a.Icon = ""
	}
	if id != ActionNewVM {
		a.On, a.Rows = page.OnRow, []string{h.Name}
	}
	return []page.Action{a, s.opening(a, vm)}
}

func (s vmSheet) server(h model.MacView) []page.Action {
	name := model.Label(s.all, h)
	const image, memory, vm = "Image", "Size.MemoryBytes", "Name"
	id := ActionNewServerOn + h.Name
	options := page.Field{Key: keyOptions, Label: "Options", Kind: page.FieldDisclosure,
		Summary: page.Texts(answer(keyKind), answer(image), answer(memory)),
		Fields:  []page.Field{s.kind(h, id), system(image), s.memory(h, memory)}}
	form := &page.Form{Title: "New VM", Subtitle: s.subtitle(h, id), Split: []string{ActionCreateOpen + id}, Primary: create(),
		Fields: []page.Field{s.on(h, func(mac string) string { return ActionNewServerOn + mac }), s.name(vm, "A server needs a name you will recognize"), options}}
	if h.Laptop {
		form.Warn = name + " may sleep: a server pauses with it."
	}
	run := &page.Run{Call: CallCreateServer, Target: macTarget(h.Name, s.self)}
	a := page.Action{ID: id, Label: "New VM", Icon: "plus", Priority: 1, Mutates: true, Sheet: page.SheetNewVM, Form: form, Run: run,
		Hud: hudSaying("Creating " + answer(vm) + " on " + name + answerEllipsis)}
	return []page.Action{a, s.opening(a, vm)}
}

// value: the split item: the same create, then its terminal on the VM it
// made, by the name asked.
func (s vmSheet) opening(a page.Action, name string) page.Action {
	run := *a.Run
	run.Then = &page.Run{Terminal: true, Argv: []string{"--on", answer(name)}, Target: run.Target}
	return page.Action{ID: ActionCreateOpen + a.ID, Label: "Create and Open in Terminal", Icon: "terminal", Key: "opt+enter", Priority: 4, Mutates: true,
		Run: &run, Form: a.Form, Hud: a.Hud}
}
