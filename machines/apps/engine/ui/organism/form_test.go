package organism_test

import (
	"maps"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/ui/organism"
)

func TestATerminalAsksEveryFieldADisclosureFolds(t *testing.T) {
	form := page.Form{Fields: []page.Field{
		{Key: "kind", Kind: page.FieldSegmented, Options: []page.Option{{Value: "workspace"}, {Value: "server"}}},
		{Key: "more", Kind: page.FieldDisclosure, Fields: []page.Field{{Key: "network", Kind: page.FieldSelect, Options: []page.Option{{Value: "nat"}}}, {Key: "memory", Kind: page.FieldSlider, Default: "8"}}},
	}}
	fields, read := organism.Fields(form)
	answers := read()
	if keys := slices.Sorted(maps.Keys(answers)); len(fields) != 3 || !slices.Equal(keys, []string{"kind", "memory", "network"}) {
		t.Errorf("a terminal asks %d fields, answering %v; want the segmented kind and the folded network and memory", len(fields), keys)
	}
	if answers["kind"] != "workspace" || answers["network"] != "nat" || answers["memory"] != "8" {
		t.Errorf("the answers before any key %v; want each field's first choice or default", answers)
	}
}
