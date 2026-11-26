package calls

import (
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/journal"
	"github.com/jterrazz/studio-station/apps/engine/images"
)

func TestABuildIsKeptAsOneLineNamingTheImagesItMade(t *testing.T) {
	made := []images.Record{{Name: "base-ubuntu"}, {Name: "studio-linux"}}
	for _, c := range []struct {
		name string
		in   BuildIn
		out  []images.Record
		err  error
		text string
		kind bool
	}{
		{"one image", BuildIn{Names: []string{"base-ubuntu"}}, made[:1], nil, "base-ubuntu built", true},
		{"every missing image", BuildIn{}, made, nil, "base-ubuntu, studio-linux built", false},
		{"a failure names what was asked", BuildIn{Names: []string{"studio-linux"}}, nil, errors.New("no"), "studio-linux not built", true},
		{"a failure of a build that named none names what it made", BuildIn{}, made[:1], errors.New("no"), "base-ubuntu not built", true},
	} {
		lines := keptBuild(c.in, c.out, c.err)
		if len(lines) != 1 || lines[0].Text != c.text || (lines[0].Subject != nil) != c.kind {
			t.Errorf("%s: kept %+v; want one line %q", c.name, lines, c.text)
		}
		if c.kind && lines[0].Subject.Kind != journal.Image {
			t.Errorf("%s: subject %+v; want an image", c.name, lines[0].Subject)
		}
	}
	if lines := keptBuild(BuildIn{}, nil, nil); lines != nil {
		t.Errorf("a build that made nothing kept %+v", lines)
	}
}
