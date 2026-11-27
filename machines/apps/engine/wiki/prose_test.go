package wiki_test

import (
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/wiki"
)

// value: where a page link of blueprint/home lands, over a corpus of three
// pages.
var homeLands = wiki.LandsOn("blueprint/home", func(id string) bool {
	return slices.Contains([]string{"blueprint/home", "blueprint/machinefiles", "product/stack"}, id)
})

func TestALinkIsServedOnlyWhereTheWindowMayFollowIt(t *testing.T) {
	for _, c := range []struct {
		target string
		want   []wiki.Run
	}{
		{"#right", []wiki.Run{{Text: "a "}, {Text: "b", Link: "#right", Kind: wiki.LinkAnchor}, {Text: " c"}}},
		{"../product/stack.md#the-stack", []wiki.Run{{Text: "a "}, {Text: "b", Link: "product/stack", Anchor: "the-stack", Kind: wiki.LinkPage}, {Text: " c"}}},
		{"machinefiles.md", []wiki.Run{{Text: "a "}, {Text: "b", Link: "blueprint/machinefiles", Kind: wiki.LinkPage}, {Text: " c"}}},
		{"./machinefiles", []wiki.Run{{Text: "a "}, {Text: "b", Link: "blueprint/machinefiles", Kind: wiki.LinkPage}, {Text: " c"}}},
		{"../blueprint/../product/./stack.md", []wiki.Run{{Text: "a "}, {Text: "b", Link: "product/stack", Kind: wiki.LinkPage}, {Text: " c"}}},
		{"https://spwn.sh/a?b=c", []wiki.Run{{Text: "a "}, {Text: "b", Link: "https://spwn.sh/a?b=c", Kind: wiki.LinkExternal}, {Text: " c"}}},
		{"HTTP://spwn.sh", []wiki.Run{{Text: "a "}, {Text: "b", Link: "HTTP://spwn.sh", Kind: wiki.LinkExternal}, {Text: " c"}}},
		{"nowhere.md", []wiki.Run{{Text: "a b c"}}},
		{"../../product/stack.md", []wiki.Run{{Text: "a b c"}}},
		{"../../../etc/passwd", []wiki.Run{{Text: "a b c"}}},
		{"/product/stack.md", []wiki.Run{{Text: "a b c"}}},
		{"stack.md?x=1", []wiki.Run{{Text: "a b c"}}},
		{"..", []wiki.Run{{Text: "a b c"}}},
		{"javascript%3Aalert", []wiki.Run{{Text: "a b c"}}},
		{"javascript:alert(1", []wiki.Run{{Text: "a b c"}}},
		{" JavaScript:x", []wiki.Run{{Text: "a b c"}}},
		{"java\tscript:x", []wiki.Run{{Text: "a b c"}}},
		{"file:///etc/passwd", []wiki.Run{{Text: "a b c"}}},
		{"mailto:a@b.c", []wiki.Run{{Text: "a b c"}}},
		{"data:text/html,x", []wiki.Run{{Text: "a b c"}}},
		{"//evil.example/x", []wiki.Run{{Text: "a b c"}}},
		{`\\evil.example\x`, []wiki.Run{{Text: "a b c"}}},
		{"", []wiki.Run{{Text: "a b c"}}},
	} {
		got := wiki.Paragraphs("a [b]("+c.target+") c", homeLands)
		if len(got) != 1 || !slices.Equal(got[0].Runs, c.want) {
			t.Errorf("a link to %q is served as %+v; want %+v", c.target, got, c.want)
		}
	}
}
