package wiki_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/wiki"
)

func TestTheCorpusWikiYamlGivesItsSchema(t *testing.T) {
	s := spwnKinds()
	floors := make([]string, 0, len(s.Floors))
	for _, f := range s.Floors {
		floors = append(floors, f.ID)
	}
	if strings.Join(floors, " ") != "intention product domains architecture code" || s.Floors[1].Question != "What do we build, and for whom?" {
		t.Errorf("floors %v, the second asks %q", floors, s.Floors[1].Question)
	}
	if len(s.FloorOf) != 5 || s.FloorOf[4] != (wiki.FloorOf{Path: "machines/", Floor: 3}) {
		t.Errorf("floor_of %+v", s.FloorOf)
	}
	if len(s.Seats) != 4 {
		t.Fatalf("seats %+v", s.Seats)
	}
	tech := s.Seats[2]
	if tech.ID != "engineering" || tech.Name != "Technique" || !slices.Equal(tech.Paths, []string{"engineering/", "machines/"}) || tech.Home != "machines/reference/ports" || tech.Plan || tech.Line != 36 {
		t.Errorf("the engineering seat %+v", tech)
	}
	if !s.Seats[0].Plan || !slices.Equal(s.Seats[0].Reads, []wiki.Intent{wiki.IntentTell, wiki.IntentDefine, wiki.IntentDecide}) {
		t.Errorf("the blueprint seat %+v", s.Seats[0])
	}
	if s.Budget != wiki.Budget2min || len(s.Repos) != 2 || s.Mounts[0] != (wiki.Mount{At: "machines", Repo: "spwn", Path: "../work/repositories/spwn/machines/docs"}) {
		t.Errorf("budget %q repos %+v mounts %+v", s.Budget, s.Repos, s.Mounts)
	}
	conviction := s.Kinds["conviction"]
	names := make([]string, 0, len(conviction.Fields))
	for _, f := range conviction.Fields {
		names = append(names, f.Name)
	}
	if strings.Join(names, " ") != "says opposite carried_in broken_by adr" || conviction.Weighs != 1 || conviction.Icon != "conviction" || conviction.Line != 55 {
		t.Errorf("conviction %+v", conviction)
	}
	if f := conviction.Fields[2]; f.Name != "carried_in" || f.Type != "list<ref>" || f.To != "part" || !f.Required {
		t.Errorf("carried_in %+v", f)
	}
	if s.Kinds["part"].State != "state" || !slices.Equal(s.Kinds["part"].Fields[3].Values, []string{"shipped", "making"}) {
		t.Errorf("part %+v", s.Kinds["part"])
	}
	if len(s.Icons) != 5 || s.Icons[0].Name != "conviction" || len(s.Icons[0].Rows) != 5 {
		t.Errorf("icons %+v", s.Icons)
	}
}

// value: the least wiki.yaml the engine reads; each refusal case below
// replaces one line of it.
const minimalWiki = `wiki: spwn.wiki/v1
floors:
    - { id: intention, name: Intention, question: "Why?" }
    - { id: product, name: Product, question: "What?" }
    - { id: domains, name: Domains, question: "Which words?" }
    - { id: architecture, name: Architecture, question: "Where?" }
    - { id: code, name: Code, question: "Which lines?" }
floor_of: { blueprint/: 0 }
seats:
    - { id: blueprint, name: Vision, path: [blueprint/], reads: [tell], home: blueprint/a }
repos: { spwn: ../spwn }
mounts:
    - { at: machines, repo: spwn, path: ../spwn/docs }
budget: 2min
kinds:
    part:
        state: state
        fields:
            state: { type: enum, values: [shipped, making] }
        faces: { line: "{state}" }
`

func TestTheMinimalWikiYamlIsRead(t *testing.T) {
	if _, err := wiki.ReadSchema("wiki.yaml", []byte(minimalWiki)); err != nil {
		t.Fatal(err)
	}
}

func TestEachRefusalOfWikiYamlSaysWhere(t *testing.T) {
	floor := "    - { id: product, name: Product, question: \"What?\" }\n"
	seat := "    - { id: blueprint, name: Vision, path: [blueprint/], reads: [tell], home: blueprint/a }\n"
	state := "            state: { type: enum, values: [shipped, making] }\n"
	cases := []struct{ name, from, to, want string }{
		{"no schema line", "wiki: spwn.wiki/v1\n", "", `wiki.yaml:1: H42 names no schema: its first key is ` + "`wiki: spwn.wiki/v1`"},
		{"a newer schema", "wiki: spwn.wiki/v1\n", "wiki: spwn.wiki/v2\n", `wiki.yaml: H42 is spwn.wiki/v2, newer than this station reads (spwn.wiki/v1): update station`},
		{"an unknown key", "budget: 2min\n", "budget: 2min\ntheme: dark\n", `wiki.yaml:15: H42 "theme" is no key wiki.yaml reads here`},
		{"an unknown key of a field", state, "            state: { type: enum, values: [shipped, making], shade: red }\n", `wiki.yaml:19: H42 "shade" is no key wiki.yaml reads here`},
		{"four floors", floor, "", `wiki.yaml:2: H42 4 floors, a corpus has 5: intention, product, domains, architecture, code`},
		{"a floor twice", floor, "    - { id: intention, name: Product, question: \"What?\" }\n", `wiki.yaml:4: H42 floor "intention" is declared twice`},
		{"a floor with no question", floor, "    - { id: product, name: Product }\n", `wiki.yaml:4: H42 floor "product" names no name or no question: a floor is the question it answers`},
		{"floor_of out of the floors", "floor_of: { blueprint/: 0 }\n", "floor_of: { blueprint/: 5 }\n", `wiki.yaml:8: H42 floor_of blueprint/ is 5: a floor is 0 to 4`},
		{"floor_of of no folder", "floor_of: { blueprint/: 0 }\n", "floor_of: { blueprint: 0 }\n", `wiki.yaml:8: H42 floor_of "blueprint" is no folder: a folder of the corpus, ending in /`},
		{"seat paths that overlap", seat, seat + "    - { id: deep, name: Deep, path: [blueprint/x/], reads: [tell], home: blueprint/x/a }\n", `wiki.yaml:11: H42 seat paths blueprint/ and blueprint/x/ overlap`},
		{"a seat twice", seat, seat + "    - { id: blueprint, name: V, path: [other/], reads: [tell], home: other/a }\n", `wiki.yaml:11: H42 seat "blueprint" is declared twice`},
		{"a seat of no intent", seat, "    - { id: blueprint, name: Vision, path: [blueprint/], reads: [explain], home: blueprint/a }\n", `wiki.yaml:10: H42 seat "blueprint" reads "explain", which is no intent: tell, define, decide, locate, guard or prove`},
		{"a seat with no home", seat, "    - { id: blueprint, name: Vision, path: [blueprint/], reads: [tell] }\n", `wiki.yaml:10: H42 seat "blueprint" names no home: the page it opens on`},
		{"a mount of no repository", "    - { at: machines, repo: spwn, path: ../spwn/docs }\n", "    - { at: machines, repo: web, path: ../spwn/docs }\n", `wiki.yaml:13: H42 mount machines reads the repository "web", which repos does not name`},
		{"a budget that does not exist", "budget: 2min\n", "budget: 5min\n", `wiki.yaml:14: H42 budget "5min" does not exist: 30s, 2min or 10min`},
		{"an engine kind declared", "    part:\n", "    rule:\n", `wiki.yaml:16: H42 kind rule is the engine's: a corpus declares kinds of its own`},
		{"a kind that weighs 3", "        state: state\n", "        state: state\n        weighs: 3\n", `wiki.yaml:16: H42 kind part weighs 3: 0, 1 or 2`},
		{"a field type that does not exist", state, state + "            price: { type: money }\n", `wiki.yaml:20: H42 kind part: field price's type "money" is none of sentence, text, enum, ref, anchor, date, bool, int, list<T> or a union a|b`},
		{"an enum with no values", state, "            state: { type: enum }\n", `wiki.yaml:19: H42 kind part: field state: an enum names its values, and only an enum does`},
		{"a max on no sentence", state, state + "            count: { type: int, max: 3 }\n", `wiki.yaml:20: H42 kind part: field count: max caps a sentence, at 1 character or more`},
		{"a ref to no kind", state, state + "            of: { type: ref, to: planet }\n", `wiki.yaml:20: H42 kind part: field of points to "planet", which is no kind`},
		{"a common field declared", state, state + "            weight: { type: text }\n", `wiki.yaml:20: H42 kind part: weight is a field every kind takes, never declared`},
		{"a state that is no enum", "        state: state\n", "        state: shade\n", `wiki.yaml:16: H42 kind part: its state is "shade", which is no enum field of it`},
		{"a face of no field", "        faces: { line: \"{state}\" }\n", "        faces: { line: \"{price}\" }\n", `wiki.yaml:16: H42 kind part: the face "{price}" names {price}, which is no field of it`},
		{"a kind id with a dash", "    part:\n", "    the-part:\n", `wiki.yaml:16: H42 kind "the-part" is no id: lower-case letters and digits, its tag the id capitalised`},
		{"a question cut by its comma", floor, "    - { id: product, name: Product, question: What, and for whom }\n", `wiki.yaml:4: H42 "and for whom" is no key wiki.yaml reads here`},
	}
	for _, c := range cases {
		if !strings.Contains(minimalWiki, c.from) {
			t.Fatalf("%s: the minimal wiki.yaml has no %q", c.name, c.from)
		}
		_, err := wiki.ReadSchema("wiki.yaml", []byte(strings.Replace(minimalWiki, c.from, c.to, 1)))
		if err == nil || err.Error() != c.want {
			t.Errorf("%s:\n got %v\nwant %s", c.name, err, c.want)
		}
	}
}

func TestEveryContentKindPastSixteenIsRefused(t *testing.T) {
	var b strings.Builder
	for _, id := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"} {
		b.WriteString("    k" + id + ":\n        fields: { x: { type: text } }\n")
	}
	body := strings.Replace(minimalWiki, "kinds:\n", "kinds:\n"+b.String(), 1)
	_, err := wiki.ReadSchema("wiki.yaml", []byte(body))
	if want := `wiki.yaml:15: H42 10 kinds of the corpus and 8 of the engine make 18 content kinds, 16 at most`; err == nil || err.Error() != want {
		t.Errorf("got %v\nwant %s", err, want)
	}
}
