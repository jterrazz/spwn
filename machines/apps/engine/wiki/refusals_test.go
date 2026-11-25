package wiki_test

import (
	"fmt"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/wiki"
)

func said(p wiki.Page) []string {
	out := make([]string, 0, len(p.Findings))
	for _, f := range p.Findings {
		out = append(out, fmt.Sprintf("%s %d:%d %s", f.Rule, f.Line, f.Col, f.Says))
	}
	return out
}

func TestEachHardRefusalOfTheParseAndTheCompileSaysWhere(t *testing.T) {
	const expression = " is an expression: in braces, a value, ref(...) or query(...), nothing else"
	cases := []struct{ name, body, want string }{
		{"a tag the page ends inside", "<Table id=\"x\" of=\"a\"\n\nThe end.\n", `H38 3:1 <Table at line 3 is never closed: end it with />`},
		{"a group tag the page ends inside", "<Group id=\"g\" title=\"G\"", `H38 3:1 <Group at line 3 is never closed: end it with >`},
		{"a brace never closed", "<Term id=\"a\" means={[1, 2\n", `H38 3:1 <Term at line 3 is never closed: end it with />`},
		{"a string the line ends inside", "<Rule id=\"x />\n", `H38 3:10 <Rule at line 3: the string of "id" is never terminated: a string ends with " on the line it opens`},
		{"text after the tag", "<Term id=\"a\" means=\"b\" /> and more\n", `H38 3:26 <Term at line 3: text follows the tag on its last line: a tag ends its line`},
		{"a block that opens", "<Term id=\"a\" means=\"b\">\n", `H38 3:1 <Term at line 3 ends with >: a block ends with />, only a Group holds what follows it`},
		{"an attribute twice", "<Term id=\"a\" id=\"b\" />\n", `H38 3:14 <Term at line 3: "id" is written twice`},
		{"attributes with no space", "<Term id=\"a\"means=\"b\" />\n", `H38 3:13 <Term at line 3: an attribute is set apart from what precedes it by a space`},
		{"a bare word as a value", "<Term id=a />\n", `H38 3:10 <Term at line 3: the value of "id" is a string "..." or a braced value {...}`},
		{"a character that names nothing", "<Term id=\"a\" @x />\n", `H38 3:14 <Term at line 3: expected an attribute or the tag's end, found "@"`},
		{"a method call", "<Table id=\"t\" of=\"conviction\" in={ref(\"p\")} columns={convictions.filter(c => c.weight > 1)} />\n", `H39 3:53 {convictions.filter(c => c.weight > 1)}` + expression},
		{"state outside a condition", "<Term id=\"a\" means={state(\"shipped\")} />\n", `H39 3:20 {state("shipped")}` + expression},
		{"a template literal", "<Term id=\"a\" means={`hi ${name}`} />\n", "H39 3:20 {`hi ${name}`}" + expression},
		{"an operator", "<Term id=\"a\" means={1 + 2} />\n", `H39 3:20 {1 + 2}` + expression},
		{"a call of the page's", "<Term id=\"a\" means={process.exit(1)} />\n", `H39 3:20 {process.exit(1)}` + expression},
		{"an expression over lines", "<Term id=\"a\" means={[\n  1,\n  2 +\n]} />\n", `H39 3:20 {[ 1, 2 + ]}` + expression},
		{"a ref of a name", "<Term id=\"a\" means={ref(name)} />\n", `H39 3:20 {ref(name)}` + expression},
		{"a query of two conditions", "<Table id=\"t\" of=\"part\" in={ref(\"p\")} columns={[]} where={query({ state: { is: \"shipped\" }, space: { is: \"os\" } })} />\n", `H39 3:58 {query({ state: { is: "shipped" }, space: { is: "os" } })}` + expression},
		{"a query of another operator", "<Table id=\"t\" of=\"part\" in={ref(\"p\")} columns={[]} where={query({ state: { not: \"shipped\" } })} />\n", `H39 3:58 {query({ state: { not: "shipped" } })}` + expression},
		{"a group in a group", "<Group id=\"a\" title=\"A\" intent=\"tell\">\n\n<Group id=\"b\" title=\"B\" intent=\"tell\">\n\n</Group>\n\n</Group>\n", `H40 5:1 <Group "b"> opens inside <Group "a">: a group never holds a group`},
		{"a group over a heading", "<Group id=\"code\" title=\"C\" intent=\"locate\">\n\n## Where the code is\n", `H40 5:1 <Group "code"> is still open at "## Where the code is": close it before the heading`},
		{"a group never closed", "<Group id=\"g\" title=\"G\" intent=\"tell\">\n\nprose\n", `H40 3:1 <Group "g"> is never closed: end it with </Group>`},
		{"a close with no group", "</Group>\n", `H40 3:1 </Group> closes no group`},
		{"an attribute its kind lacks", "<Alternative id=\"a\" gives=\"g\" costs=\"c\" price=\"1\" />\n", `H41 3:41 "price" is not a field of alternative`},
		{"a table written row by row", "<Table id=\"t\" of=\"conviction\" in={ref(\"p\")} columns={[\"says\"]} rows={[]} />\n", `H22 3:64 a table is not written row by row: a row is a block of kind conviction; write the block, the table finds it`},
		{"a kind written wrong", "<Rul id=\"x\" />\n", `H1 3:1 unknown kind "Rul" (did you mean Rule?)`},
		{"a kind that is no kind", "<Zebra id=\"x\" />\n", `H1 3:1 unknown kind "Zebra"`},
	}
	for _, c := range cases {
		got := said(wiki.Compile("p", "# T\n\n"+c.body, spwnKinds()))
		if len(got) != 1 || got[0] != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
}

func TestReadingGoesOnAfterARefusedTag(t *testing.T) {
	p := wiki.Compile("p", "# T\n\n<Term id=\"a\" means={x} />\n\n<Term id=\"b\" means=\"read\" />\n", spwnKinds())
	if got := said(p); len(got) != 1 {
		t.Fatalf("one refusal, got %q", got)
	}
	items := p.Sections[0].Items
	if len(items) != 1 {
		t.Fatalf("the tag after the refused one compiles alone, got %d items", len(items))
	}
	if b, ok := items[0].(*wiki.Block); !ok || b.ID != "b" {
		t.Errorf("the block after the refused tag is b, got %#v", items[0])
	}
}
