package wiki

import (
	"maps"
	"slices"
	"strings"
)

type Schema struct {
	// value: what a wiki.yaml declares that the compiler reads, the corpus's
	// own kinds by id.
	Kinds map[string]Kind
}

type Kind struct {
	ID     string
	Icon   string
	Weighs int
	// value: the field `state("...")` reads on a block of this kind; empty
	// when the kind has none.
	State  string
	Fields []Field
	Faces  FaceTemplates
	// value: an engine kind that holds no fact and is never counted in the cap.
	Structural bool
}

type Field struct {
	Name     string
	Type     string
	Required bool
	Max      int
	Values   []string
	To       string
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

func fields(specs ...string) []Field {
	out := make([]Field, 0, len(specs))
	for _, s := range specs {
		name, typ, _ := strings.Cut(s, ":")
		required := strings.HasSuffix(typ, "!")
		out = append(out, Field{Name: name, Type: strings.TrimSuffix(typ, "!"), Required: required})
	}
	return out
}

// invariant: the engine's kinds, SPEC-v1 §5: eight content kinds and five
// structural ones. A field written `name:type!` is required.
func builtinKinds() map[string]Kind {
	return map[string]Kind{
		"term": {ID: "term", Fields: fields("means:sentence!", "not:sentence", "code:anchor", "retired:date", "replaced_by:ref"),
			Faces: FaceTemplates{Word: "{id}", Line: "{means}"}},
		"rule": {ID: "rule", Fields: fields("says:sentence!", "force:enum!", "guards:ref|glob!", "proof:anchor", "read_at:anchor", "precedents:list<precedent>", "waived:object", "waiver:sentence"),
			Faces: FaceTemplates{Word: "{id}", Line: "{says}", Icon: "{force}"}},
		"scenario": {ID: "scenario", Fields: fields("of:ref!", "given:sentence", "when:sentence", "then:sentence", "proof:anchor"),
			Faces: FaceTemplates{Word: "{id}", Line: "{when}: {then}"}},
		"component": {ID: "component", Fields: fields("level:enum!", "aims:sentence!", "source:anchor!", "made_of:list<ref>"),
			Faces: FaceTemplates{Word: "{id}", Line: "{aims}", Icon: "{level}"}},
		"claim": {ID: "claim", Fields: fields("says:text!", "carried_by:anchor!", "certainty:enum!", "because:ref"),
			Faces: FaceTemplates{Word: "{id}", Line: "{says}"}},
		"folder": {ID: "folder", Fields: fields("path:anchor!", "holds:sentence!", "writes:list<ref|anchor>", "written_by:ref"),
			Faces: FaceTemplates{Word: "{id}", Line: "{holds}"}},
		"link": {ID: "link", Fields: fields("from:ref|glob!", "to:ref|glob|text!", "right:enum!", "because:ref|sentence!", "proof:anchor"),
			Faces: FaceTemplates{Word: "{id}", Line: "{from} {right} {to}", Icon: "{right}"}},
		"decision": {ID: "decision", Fields: fields("adr:anchor!", "decides:sentence!"),
			Faces: FaceTemplates{Word: "{id}", Line: "{decides}"}},
		"group": {ID: "group", Structural: true, Fields: fields("title:sentence!", "intent:enum!", "fold:bool"),
			Faces: FaceTemplates{Word: "{title}"}},
		"table": {ID: "table", Structural: true, Fields: fields("of:text!", "in:ref!", "columns:list<text>!", "where:query", "sort:text"),
			Faces: FaceTemplates{Word: "{id}"}},
		"page": {ID: "page", Structural: true, Fields: fields("face:enum"),
			Faces: FaceTemplates{Word: "{id}"}},
		"figure": {ID: "figure", Structural: true, Fields: fields("form:enum!", "shows:ref!", "caption:sentence!", "parts:list<object>", "centre:text", "wall:text", "stop:text", "x:text", "y:text"),
			Faces: FaceTemplates{Word: "{id}", Line: "{caption}"}},
		"lens": {ID: "lens", Structural: true, Fields: fields("of:text", "from:text"),
			Faces: FaceTemplates{Word: "{id}"}},
	}
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
