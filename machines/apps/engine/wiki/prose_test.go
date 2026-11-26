package wiki_test

import (
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/wiki"
)

func TestALinkIsServedOnlyWhereTheWindowMayFollowIt(t *testing.T) {
	for _, c := range []struct {
		target string
		want   []wiki.Run
	}{
		{"#right", []wiki.Run{{Text: "a "}, {Text: "b", Link: "#right", Kind: wiki.LinkAnchor}, {Text: " c"}}},
		{"../product/stack.md#the-stack", []wiki.Run{{Text: "a "}, {Text: "b", Link: "../product/stack.md#the-stack", Kind: wiki.LinkPage}, {Text: " c"}}},
		{"machinefiles.md", []wiki.Run{{Text: "a "}, {Text: "b", Link: "machinefiles.md", Kind: wiki.LinkPage}, {Text: " c"}}},
		{"https://spwn.sh/a?b=c", []wiki.Run{{Text: "a "}, {Text: "b", Link: "https://spwn.sh/a?b=c", Kind: wiki.LinkExternal}, {Text: " c"}}},
		{"HTTP://spwn.sh", []wiki.Run{{Text: "a "}, {Text: "b", Link: "HTTP://spwn.sh", Kind: wiki.LinkExternal}, {Text: " c"}}},
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
		got := wiki.Paragraphs("a [b](" + c.target + ") c")
		if len(got) != 1 || !slices.Equal(got[0].Runs, c.want) {
			t.Errorf("a link to %q is served as %+v; want %+v", c.target, got, c.want)
		}
	}
}
