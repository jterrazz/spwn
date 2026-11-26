package project_test

import (
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/project"
)

func file(path, data string) project.File {
	return project.File{Path: path, Data: []byte(data), Mode: 0o644}
}

func TestChangesOf(t *testing.T) {
	a, b := file("CLAUDE.md", "a"), file("CLAUDE.md", "b")
	p := project.RunFolder("neo") + "/CLAUDE.md"
	one := map[string][]project.File{"neo": {a}}
	for _, c := range []struct {
		name         string
		rendered     map[string][]project.File
		record, disk project.Record
		want         []project.Change
	}{
		{"a file the disk lacks is added", one, nil, nil, []project.Change{{Agent: "neo", Path: p, Kind: project.Add}}},
		{"a file the disk holds as rendered is no change", one, nil, project.Record{p: project.Digest(a)}, nil},
		{"a file apply left, rendered anew, is overwritten", map[string][]project.File{"neo": {b}}, project.Record{p: project.Digest(a)}, project.Record{p: project.Digest(a)},
			[]project.Change{{Agent: "neo", Path: p, Kind: project.Overwrite}}},
		{"a file edited since apply is overwritten by hand", one, project.Record{p: project.Digest(a)}, project.Record{p: project.Digest(b)},
			[]project.Change{{Agent: "neo", Path: p, Kind: project.Overwrite, ByHand: true}}},
		{"a file apply never wrote, in the way, is overwritten by hand", one, nil, project.Record{p: project.Digest(b)},
			[]project.Change{{Agent: "neo", Path: p, Kind: project.Overwrite, ByHand: true}}},
		{"a file written and no longer rendered is removed", map[string][]project.File{}, project.Record{p: project.Digest(a)}, project.Record{p: project.Digest(a)},
			[]project.Change{{Agent: "neo", Path: p, Kind: project.Remove}}},
		{"a file written, edited, and no longer rendered is removed by hand", map[string][]project.File{}, project.Record{p: project.Digest(a)}, project.Record{p: project.Digest(b)},
			[]project.Change{{Agent: "neo", Path: p, Kind: project.Remove, ByHand: true}}},
		{"a file written and already gone is no change", map[string][]project.File{}, project.Record{p: project.Digest(a)}, nil, nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, next, err := project.ChangesOf(c.rendered, c.record, c.disk)
			if err != nil || len(got) != len(c.want) {
				t.Fatalf("ChangesOf = %+v, %v; want %+v", got, err, c.want)
			}
			for i := range got {
				if got[i].Agent != c.want[i].Agent || got[i].Path != c.want[i].Path || got[i].Kind != c.want[i].Kind || got[i].ByHand != c.want[i].ByHand {
					t.Errorf("change %d = %+v; want %+v", i, got[i], c.want[i])
				}
			}
			if len(next) != len(c.rendered["neo"]) {
				t.Errorf("record = %v; want one digest per rendered file", next)
			}
		})
	}
}

func TestARenderedPathNeverLeavesItsRunFolder(t *testing.T) {
	for _, bad := range []string{"../spwn.yaml", "/etc/passwd", "", ".", "..", "a/../../b", "a//b"} {
		if _, _, err := project.ChangesOf(map[string][]project.File{"neo": {file(bad, "x")}}, nil, nil); !errors.Is(err, project.ErrOutside) {
			t.Errorf("ChangesOf(%q) = %v; want ErrOutside", bad, err)
		}
	}
}

func TestADigestTellsTheModeAndALinkApart(t *testing.T) {
	plain := file("x", "target")
	exec := project.File{Path: "x", Data: []byte("target"), Mode: 0o755}
	link := project.File{Path: "x", Data: []byte("target"), Link: true}
	if project.Digest(plain) == project.Digest(exec) || project.Digest(plain) == project.Digest(link) {
		t.Error("a digest must differ by mode and by link")
	}
}

func TestGoesNamesEachLoss(t *testing.T) {
	said := project.Goes([]project.Change{
		{Path: ".spwn/agents/neo/CLAUDE.md", Kind: project.Overwrite, ByHand: true},
		{Path: ".spwn/agents/neo/old.md", Kind: project.Remove},
	})
	if said != "overwrites .spwn/agents/neo/CLAUDE.md, changed by hand; removes .spwn/agents/neo/old.md" {
		t.Errorf("Goes = %q", said)
	}
}
