package machinefile_test

import (
	"bytes"
	"flag"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
)

var updateSkill = flag.Bool("update", false, "rewrite testdata/spwn-skill.golden from the fixture machine")

func builtinApps(t *testing.T) []catalog.App {
	t.Helper()
	c, err := catalog.Builtin()
	if err != nil {
		t.Fatal(err)
	}
	return c.Apps
}

func picked(t *testing.T, names ...string) []catalog.App {
	t.Helper()
	var out []catalog.App
	for _, a := range builtinApps(t) {
		if slices.Contains(names, a.Name) {
			out = append(out, a)
		}
	}
	if len(out) != len(names) {
		t.Fatalf("the catalog holds %d of %v", len(out), names)
	}
	return out
}

// value: the fixture machine: a workspace, with a mount and port
// rights, and a few apps, one of them with no agent line.
func fixtureWhere() machinefile.Where {
	return machinefile.Where{
		Name: "dev-1", Kind: catalog.RoleWorkspace,
		System: catalog.System{OS: catalog.OSUbuntu, Version: "24.04", Arch: catalog.ArchARM64},
		Values: []machinefile.Value{
			{Key: "vm.cpus", Text: "4"},
			{Key: "vm.mounts", Text: "~/code at /code"},
			{Key: "ports.open", Text: "3000-3999"},
			{Key: "ports.to", Text: "this-mac"},
		},
	}
}

func TestTheSpwnSkillOfTheFixtureMachineIsItsGolden(t *testing.T) {
	got := machinefile.Skill(fixtureWhere(), picked(t, "ripgrep", "bat", "node", "jq", "lazygit"), catalog.SpwnDrive)
	const golden = "testdata/spwn-skill.golden"
	if *updateSkill {
		if err := os.WriteFile(golden, got, 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("the skill drifted from %s (go test -run %s -update rewrites it):\n%s", golden, t.Name(), got)
	}
}

func TestTheSpwnSkillIsTheSameBytesWhateverTheOrderItIsHanded(t *testing.T) {
	apps := picked(t, "ripgrep", "bat", "node", "jq", "lazygit")
	one := machinefile.Skill(fixtureWhere(), apps, catalog.SpwnDrive)
	slices.Reverse(apps)
	if two := machinefile.Skill(fixtureWhere(), apps, catalog.SpwnDrive); !bytes.Equal(one, two) {
		t.Error("two runs over the same apps wrote different bytes")
	}
}

func TestTheSpwnSkillHoldsWithinItsBoundOnTheWorstCatalog(t *testing.T) {
	where := fixtureWhere()
	where.Values = append(where.Values,
		machinefile.Value{Key: "vm.mounts", Text: strings.Repeat("~/projects/one-long-folder at /projects/one-long-folder, ", 4)},
		machinefile.Value{Key: "ports.except", Text: "22, 5432, 6379"},
	)
	every := builtinApps(t)
	got := machinefile.Skill(where, every, catalog.SpwnDrive)
	if len(got) > machinefile.SkillMax {
		t.Fatalf("with every package present the skill is %d bytes, past %d", len(got), machinefile.SkillMax)
	}
	t.Logf("with all %d packages present the skill is %d bytes", len(every), len(got))
	for _, a := range every {
		if !strings.Contains(string(got), "`"+a.Name+"`") {
			t.Errorf("past the bound %s is not even named", a.Name)
		}
	}
	if !strings.Contains(string(got), "`station apps <name>` prints one app's detail") {
		t.Error("past the bound the skill does not say how to read one app")
	}
}

func TestATooLongListNamesTheAppsWithNoAgentLineFirst(t *testing.T) {
	many := make([]catalog.App, 0, 60)
	for i := range 60 {
		many = append(many, catalog.App{Name: "app-" + string(rune('a'+i/26)) + string(rune('a'+i%26)), Category: catalog.Categories[0], Description: strings.Repeat("d", 120)})
	}
	many[0].Agent = "kept whole: app-aa <file>"
	got := string(machinefile.Skill(fixtureWhere(), many, "drive"))
	if !strings.Contains(got, "- `app-aa`: kept whole: app-aa <file>\n") || strings.Contains(got, strings.Repeat("d", 120)) {
		t.Errorf("past the bound the agent lines stay and the descriptions go:\n%s", got)
	}
}
