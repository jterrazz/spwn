package wiki

import (
	"maps"
	"slices"
	"strconv"
	"strings"
	"sync"
)

type Schema struct {
	// value: the file the schema was read from, which a finding about it names.
	File string
	// invariant: five, in order; a floor's number is its index.
	Floors []Floor
	// value: the floor of a page by the folder it sits in, the longest path
	// that holds it winning.
	FloorOf []FloorOf
	Seats   []Seat
	// value: the corpus's own kinds, by id.
	Kinds  map[string]Kind
	Icons  []Icon
	Repos  []Repo
	Mounts []Mount
	Budget Budget
}

type Floor struct {
	ID       string
	Name     string
	Question string
}

type FloorOf struct {
	Path  string
	Floor int
}

type Seat struct {
	ID    string
	Name  string
	Paths []string
	Reads []Intent
	Home  string
	// value: the blueprint: a reader's seat, not an owning seat of the doctrine.
	Plan bool
	Line int
}

type Icon struct {
	Name string
	Rows []string
	Line int
}

type Repo struct {
	Name string
	Path string
}

type Mount struct {
	At   string
	Repo string
	Path string
}

type Budget string

const (
	Budget30s   Budget = "30s"
	Budget2min  Budget = "2min"
	Budget10min Budget = "10min"
)

type Kind struct {
	ID     string
	Title  map[string]string
	Icon   string
	Weighs int
	// value: the field `state("...")` reads on a block of this kind; empty
	// when the kind has none.
	State  string
	Fields []Field
	Faces  FaceTemplates
	// value: an engine kind that holds no fact and is never counted in the cap.
	Structural bool
	Line       int
}

type Field struct {
	Name     string
	Type     string
	Required bool
	Max      int
	Values   []string
	To       string
}

func (k Kind) field(name string) (Field, bool) {
	for _, f := range k.Fields {
		if f.Name == name {
			return f, true
		}
	}
	return Field{}, false
}

func intents() []Intent {
	return []Intent{IntentTell, IntentDefine, IntentDecide, IntentLocate, IntentGuard, IntentProve}
}

// invariant: an author's text becomes an enum's value only by naming one of
// its constants, never by conversion, so each enum stays closed on the wire.
func named[T ~string](values []T, s string) (T, bool) {
	for _, v := range values {
		if string(v) == s {
			return v, true
		}
	}
	var none T
	return none, false
}

func isNamed[T ~string](values []T, s string) bool {
	_, ok := named(values, s)
	return ok
}

// value: a group's intent as served: the one it names, else tell, the
// intent of prose, as an intent that does not exist is refused (H17).
func intentOf(s string) Intent {
	if in, ok := named(intents(), s); ok {
		return in
	}
	return IntentTell
}

type FaceTemplates struct {
	// value: a template, `{field}` standing for a field's text and `{id}`
	// for the block's id.
	Word string
	Line string
	Icon string
}

// value: the attributes every kind takes besides its own fields.
func commonFields() []string { return []string{"id", "floor", "rests_on", "weight", "icon"} }

// value: a field written `name:type`, `!` after the type when it is
// required, `=a,b` for an enum's values, `@n` for a sentence's cap.
func fields(specs ...string) []Field {
	out := make([]Field, 0, len(specs))
	for _, s := range specs {
		name, typ, _ := strings.Cut(s, ":")
		typ, limit, _ := strings.Cut(typ, "@")
		typ, values, _ := strings.Cut(typ, "=")
		f := Field{Name: name, Type: strings.TrimSuffix(typ, "!"), Required: strings.HasSuffix(typ, "!")}
		if values != "" {
			f.Values = strings.Split(values, ",")
		}
		if limit != "" {
			f.Max, _ = strconv.Atoi(limit)
		}
		out = append(out, f)
	}
	return out
}

// invariant: the engine's kinds, SPEC-v1 §5: eight content kinds and five
// structural ones, their enums r8's (SPEC-primitives, "The Kind schemas").
// The table is built once, as a serve looks a kind up for each block and
// each row it builds; no caller writes to it.
var builtinKinds = sync.OnceValue(func() map[string]Kind {
	return map[string]Kind{
		"term": {ID: "term", Fields: fields("means:sentence!", "not:sentence", "code:anchor", "retired:date", "replaced_by:ref"),
			Faces: FaceTemplates{Word: "{id}", Line: "{means}"}},
		"rule": {ID: "rule", Fields: fields("says:sentence!@160", "force:enum!=hard,soft,hardened", "guards:ref|glob!", "proof:anchor", "read_at:anchor", "precedents:list<object>", "waived:object", "waiver:sentence"),
			Faces: FaceTemplates{Word: "{id}", Line: "{says}", Icon: "{force}"}},
		"scenario": {ID: "scenario", Fields: fields("of:ref!", "given:sentence", "when:sentence", "then:sentence", "proof:anchor"),
			Faces: FaceTemplates{Word: "{id}", Line: "{when}: {then}"}},
		"component": {ID: "component", Fields: fields("level:enum!="+strings.Join(levels(), ","), "aims:sentence!", "source:anchor!", "made_of:list<ref>"),
			Faces: FaceTemplates{Word: "{id}", Line: "{aims}", Icon: "{level}"}},
		"claim": {ID: "claim", Fields: fields("says:text!", "carried_by:anchor!", "certainty:enum!=decided,trial", "because:ref"),
			Faces: FaceTemplates{Word: "{id}", Line: "{says}"}},
		"folder": {ID: "folder", Fields: fields("path:anchor!", "holds:sentence!", "writes:list<ref|anchor>", "written_by:ref"),
			Faces: FaceTemplates{Word: "{id}", Line: "{holds}"}},
		"link": {ID: "link", Fields: fields("from:ref|glob!", "to:ref|glob|text!", "right:enum!=allowed,refused", "because:ref|sentence!", "proof:anchor"),
			Faces: FaceTemplates{Word: "{id}", Line: "{from} {right} {to}", Icon: "{right}"}},
		"decision": {ID: "decision", Fields: fields("adr:anchor!", "decides:sentence!"),
			Faces: FaceTemplates{Word: "{id}", Line: "{decides}"}},
		"group": {ID: "group", Structural: true, Fields: fields("title:sentence!", "intent:enum!=tell,define,decide,locate,guard,prove", "fold:bool"),
			Faces: FaceTemplates{Word: "{title}"}},
		"table": {ID: "table", Structural: true, Fields: fields("of:text!", "in:ref!", "columns:list<text>!", "where:query", "sort:text"),
			Faces: FaceTemplates{Word: "{id}"}},
		"page": {ID: "page", Structural: true, Fields: fields("face:enum=line,card,open"),
			Faces: FaceTemplates{Word: "{id}"}},
		"figure": {ID: "figure", Structural: true, Fields: fields("form:enum!="+strings.Join(forms(), ","), "shows:ref!", "caption:text!", "parts:list<object>", "centre:text", "wall:text", "stop:text", "x:text", "y:text"),
			Faces: FaceTemplates{Word: "{id}", Line: "{caption}"}},
		"lens": {ID: "lens", Structural: true, Fields: fields("of:text", "from:text"),
			Faces: FaceTemplates{Word: "{id}"}},
	}
})

// value: a component's levels, lowest first: the folders of
// machines/apps/engine/ui/.
func levels() []string { return []string{"token", "atom", "molecule", "organism", "template"} }

func forms() []string { return []string{"orbit", "gate", "flow", "stack", "quad"} }

// value: the engine's content kinds, which count in the cap with the corpus's.
func contentKinds() []string {
	var out []string
	for id, k := range builtinKinds() {
		if !k.Structural {
			out = append(out, id)
		}
	}
	slices.Sort(out)
	return out
}

// invariant: an engine kind wins over a corpus kind of the same id; that a
// corpus redeclares one is wiki.yaml's to refuse, not the compiler's.
func (s Schema) kind(id string) (Kind, bool) {
	if k, ok := builtinKinds()[id]; ok {
		return k, true
	}
	k, ok := s.Kinds[id]
	if ok && k.ID == "" {
		k.ID = id
	}
	return k, ok
}

// value: every kind's tag name, `conviction` written `Conviction`, sorted.
func (s Schema) tagNames() []string {
	ids := slices.Collect(maps.Keys(builtinKinds()))
	for id := range s.Kinds {
		if !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, tagOf(id))
	}
	slices.Sort(out)
	return out
}

func tagOf(kind string) string {
	if kind == "" {
		return ""
	}
	return strings.ToUpper(kind[:1]) + kind[1:]
}

func kindOf(tag string) string {
	if tag == "" {
		return ""
	}
	return strings.ToLower(tag[:1]) + tag[1:]
}

func (k Kind) takes(attr string) bool {
	if slices.Contains(commonFields(), attr) {
		return true
	}
	return slices.ContainsFunc(k.Fields, func(f Field) bool { return f.Name == attr })
}

func (k Kind) fieldNames() []string {
	out := make([]string, 0, len(k.Fields))
	for _, f := range k.Fields {
		out = append(out, f.Name)
	}
	return out
}

// value: the floor a page path takes, by the longest folder of FloorOf that
// holds it, 0 when none does.
func (s Schema) floorOf(path string) int {
	best, floor := -1, 0
	for _, f := range s.FloorOf {
		if strings.HasPrefix(path, f.Path) && len(f.Path) > best {
			best, floor = len(f.Path), f.Floor
		}
	}
	return floor
}

// value: the seat whose folders hold a page path, empty when none does.
func (s Schema) seatOf(path string) string {
	for _, seat := range s.Seats {
		for _, p := range seat.Paths {
			if strings.HasPrefix(path, p) {
				return seat.ID
			}
		}
	}
	return ""
}

// value: the icons the engine ships by name: one per engine kind, per intent
// and per value a built-in icon Face renders (a rule's force, a component's
// level, a link's right). Their grids are the window's to draw; the set is
// built once, as each block naming an icon looks it up, and no caller
// writes to it.
var builtinIcons = sync.OnceValue(func() map[string]bool {
	out := map[string]bool{}
	for id := range builtinKinds() {
		out[id] = true
	}
	for _, i := range intents() {
		out[string(i)] = true
	}
	for _, name := range append([]string{"hard", "soft", "hardened", "allowed", "refused"}, levels()...) {
		out[name] = true
	}
	return out
})

// value: whether an icon is the engine's or the schema's: a set lookup, then
// at most MaxIcons names compared.
func (s Schema) iconExists(name string) bool {
	return builtinIcons()[name] || slices.ContainsFunc(s.Icons, func(i Icon) bool { return i.Name == name })
}

// value: the content kinds a corpus reads: the engine's and its own.
func (s Schema) contentKindIDs() []string {
	out := contentKinds()
	for id := range s.Kinds {
		if !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	slices.Sort(out)
	return out
}
