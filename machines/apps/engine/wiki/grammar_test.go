package wiki_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/wiki"
)

func onlyBlock(t *testing.T, p wiki.Page) *wiki.Block {
	t.Helper()
	if len(p.Findings) > 0 {
		t.Fatalf("refused: %q", said(p))
	}
	items := p.Sections[0].Items
	if len(items) != 1 {
		t.Fatalf("one item, got %d", len(items))
	}
	b, ok := items[0].(*wiki.Block)
	if !ok {
		t.Fatalf("a block, got %T", items[0])
	}
	return b
}

func TestEachValueOfTheGrammarCompilesToItsRecord(t *testing.T) {
	cases := []struct{ name, attrs, props string }{
		{"a string, no escape", `means="a \n b"`, `{"means":"a \\n b"}`},
		{"a bare attribute is true", `means="m" retired`, `{"means":"m","retired":true}`},
		{"spaces around =", "means =\n  \"m\"", `{"means":"m"}`},
		{"numbers", `means={-1.5} not={0} code={42}`, `{"means":-1.5,"not":0,"code":42}`},
		{"true, false, null", `means={true} not={false} code={null}`, `{"means":true,"not":false,"code":null}`},
		{"an array with a trailing comma", `means={[ "a" , 1, [] ,]}`, `{"means":["a",1,[]]}`},
		{"an object of bare and quoted keys", `means={{ a_1: 1, "b c": { d: null }, }}`, `{"means":{"a_1":1,"b c":{"d":null}}}`},
		{"a ref", `replaced_by={ ref( "product/stack#wiki" ) }`, `{"replaced_by":{"ref":"product/stack#wiki"}}`},
		{"a ref in an array over lines", "means={[\nref(\"a\"),\nref(\"b#c\")\n]}", `{"means":[{"ref":"a"},{"ref":"b#c"}]}`},
		{"a query of a state", `means={query({ carried_in: { has: state("shipped") } })}`, `{"means":{"query":{"field":"carried_in","op":"has","operand":{"state":"shipped"}}}}`},
		{"a query of a literal", `means={query({state:{is:"making"}})}`, `{"means":{"query":{"field":"state","op":"is","operand":{"literal":"making"}}}}`},
		{"a query of a ref", `means={query({ by: { is: ref("product/stack#machines") } })}`, `{"means":{"query":{"field":"by","op":"is","operand":{"ref":"product/stack#machines"}}}}`},
		{"a brace inside a string", `means={"a } b"}`, `{"means":"a } b"}`},
	}
	for _, c := range cases {
		b := onlyBlock(t, wiki.Compile("p", "<Term id=\"x\" "+c.attrs+" />\n", spwnKinds()))
		props := b.Props[1:]
		got, err := json.Marshal(props)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != c.props {
			t.Errorf("%s:\n got %s\nwant %s", c.name, got, c.props)
		}
	}
}

func TestATagIsReadOnlyWhereALineOpensIt(t *testing.T) {
	cases := []struct{ name, source string }{
		{"after a line of prose", "Some prose.\n<Term id=\"a\" means=\"m\" />\n"},
		{"indented", "\n  <Term id=\"a\" means=\"m\" />\n"},
		{"inside a paragraph", "\nA <Term id=\"a\" means=\"m\" /> inline.\n"},
		{"in a backtick fence", "```md\n<Term id=\"a\" means=\"m\" />\n```\n"},
		{"in a tilde fence of four", "~~~~\n<Term id=\"a\" means=\"m\" />\n~~~\n<Term id=\"b\" means=\"m\" />\n~~~~\n"},
		{"a lower case tag", "\n<term id=\"a\" means=\"m\" />\n"},
	}
	for _, c := range cases {
		p := wiki.Compile("p", c.source, spwnKinds())
		for _, it := range p.Sections[0].Items {
			if _, ok := it.(*wiki.Prose); !ok {
				t.Errorf("%s: a %T, want prose only", c.name, it)
			}
		}
		if len(p.Findings) > 0 {
			t.Errorf("%s: %q", c.name, said(p))
		}
	}
}

func TestATagFollowsABlankLineAHeadingAnotherTagOrTheStart(t *testing.T) {
	source := "<Term id=\"a\" means=\"m\" />\n<Term id=\"b\" means=\"m\" />\n\n<Term id=\"c\" means=\"m\" />\n## H\n<Term id=\"d\" means=\"m\" />\n"
	p := wiki.Compile("p", source, spwnKinds())
	var ids []string
	for _, s := range p.Sections {
		for _, it := range s.Items {
			if b, ok := it.(*wiki.Block); ok {
				ids = append(ids, b.ID)
			}
		}
	}
	if strings.Join(ids, " ") != "a b c d" || len(p.Findings) > 0 {
		t.Errorf("blocks %v, findings %q", ids, said(p))
	}
}

func TestABlockKnowsItsAddressItsPlaceAndItsFaces(t *testing.T) {
	b := onlyBlock(t, wiki.Compile("product/stack", "<Rule\nid=\"one-writer\"\nsays=\"A store has one writer.\"\nforce=\"hard\"\n/>\n", spwnKinds()))
	if b.Address != "product/stack#one-writer" || b.At != (wiki.At{Line: 1, Col: 1}) || b.Span != (wiki.Span{From: 1, To: 5}) {
		t.Errorf("address %q at %+v span %+v", b.Address, b.At, b.Span)
	}
	if b.Faces != (wiki.BlockFaces{Word: "one-writer", Line: "A store has one writer.", Icon: "hard"}) {
		t.Errorf("faces %+v", b.Faces)
	}
	if v, _ := b.Props.Get("says"); v.At != (wiki.At{Line: 3, Col: 6}) {
		t.Errorf("the value of says is at %+v, want 3:6", v.At)
	}
}

func TestAColumnCountsCharactersNotBytes(t *testing.T) {
	p := wiki.Compile("p", "<Term id=\"é\" means=\"déjà vu\" @ />\n", spwnKinds())
	if got := said(p); len(got) != 1 || !strings.HasPrefix(got[0], "H38 1:30 ") {
		t.Errorf("got %q, want the @ at column 30", got)
	}
}

func FuzzCompile(f *testing.F) {
	seeds := []string{
		"# T\n\n<Table id=\"x\" of=\"conviction\" in={ref(\"p\")} where={query({ carried_in: { has: state(\"shipped\") } })} columns={[\"says\"]} />\n",
		"<Group id=\"g\" title=\"G\" intent=\"tell\" fold>\n\n<Term id=\"a\" means=\"m\" />\n\n</Group>\n",
		"<Term id=\"a\" means={process.exit(1)} />\n",
		"<Term id=\"a\" means={`${require('child_process').execSync('id')}`} />\n",
		"<Term id=\"a\" means={(() => { throw 1 })()} />\n",
		"<Term id=\"a\" means={[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[1]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]} />\n",
		"---\nkind: x\n---\n# T\n<Group>\n</Group>\n</Group>\n## H\n```\n<A\n",
		"<Figure id=\"f\" form=\"quad\" shows={ref(\"p#a\")} parts={[{ name: \"a\", at: \"xy\" }]} caption=\"c\" />",
		"<Page id=\"p/child\" />\n<\xff\n<A\xc3",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, source string) {
		first := wiki.Compile("p", source, spwnKinds())
		once, err := json.Marshal(first)
		if err != nil {
			t.Fatalf("the page does not encode: %v", err)
		}
		again, _ := json.Marshal(wiki.Compile("p", source, spwnKinds()))
		if !bytes.Equal(once, again) {
			t.Fatalf("two compiles of one source differ")
		}
		lines := strings.Count(strings.ReplaceAll(source, "\r\n", "\n"), "\n") + 1
		for _, finding := range first.Findings {
			if finding.Line < 1 || finding.Line > lines || finding.Col < 1 {
				t.Fatalf("a finding outside the page: %+v", finding)
			}
		}
	})
}
