package catalog_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func TestAFileIsHeldByTheAppWhoseDiskDeclaresItOrAFolderAboveIt(t *testing.T) {
	cat := catalog.Catalog{Apps: []catalog.App{
		{Name: "claude", Disk: catalog.Disk{Paths: []catalog.Stored{{Path: "~/.claude/settings.json"}, {Path: "~/.claude/projects"}}}},
		{Name: "rtk", Disk: catalog.Disk{Paths: []catalog.Stored{{Path: "~/.config/rtk"}}}},
	}}
	for _, c := range []struct {
		file string
		but  []string
		want string
	}{
		{"~/.claude/settings.json", nil, "claude"},
		{"~/.claude/projects/a/b.json", nil, "claude"},
		{"~/.claude/settings.json.bak", nil, ""},
		{"~/.claude/./settings.json", nil, "claude"},
		{"~/.config/rtk/filters.toml", []string{"rtk"}, ""},
		{"~/.claude/settings.json", []string{"rtk"}, "claude"},
		{"~/.caveman/cloud.json", nil, ""},
	} {
		if got := cat.HeldBy(c.file, c.but); got != c.want {
			t.Errorf("HeldBy(%q, %q) = %q, want %q", c.file, c.but, got, c.want)
		}
	}
}
