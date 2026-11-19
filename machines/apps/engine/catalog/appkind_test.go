package catalog_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func TestASkillIsListedAsAnAgentSkillAndAManagerAsAManager(t *testing.T) {
	c, err := catalog.Builtin()
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]catalog.AppKind{"better-layout": catalog.AppKindSkill, "homebrew": catalog.AppKindManager, "jq": catalog.AppKindApp} {
		app, ok := c.App(name)
		if !ok {
			t.Fatalf("the catalog has no %s", name)
		}
		if app.AppKind != want {
			t.Errorf("%s is listed as %q, want %q", name, app.AppKind, want)
		}
	}
	for _, app := range c.Apps {
		if (app.Category == catalog.CategoryAgentSkills) != (app.AppKind == catalog.AppKindSkill) {
			t.Errorf("%s is in %q and listed as %q: a skill is an agent skill", app.Name, app.Category, app.AppKind)
		}
	}
}
