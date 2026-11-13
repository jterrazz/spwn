package wire

import (
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/agents"
	"github.com/jterrazz/studio-station/apps/engine/agents/adapter/claudecode"
	"github.com/jterrazz/studio-station/apps/engine/agents/adapter/codex"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local/localtest"
	settingscontract "github.com/jterrazz/studio-station/apps/engine/settings/contract"
)

func TestEverySettingOfTheCatalogueIsWrittenByAnApplierOnARealMac(t *testing.T) {
	system := localtest.Mac(localtest.NewRunner(t), localtest.NewFS(), localtest.Env{HomeDir: "/home/u", OS: "darwin"}, localtest.NewNet(), &localtest.Clock{})

	appliers := settingscontract.NewAppliers(settingAppliers(machine{sys: system, vars: map[string]string{}, stores: &stores{}})...)

	c, err := catalog.Builtin()
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range c.Settings {
		if _, ok := appliers.For(it); !ok {
			t.Errorf("%s: its effect %v is written by no Applier", it.Name, it.Effect)
		}
	}
}

func TestOnARealWorkspaceEachAppsHookIsWrittenInItsOwnSettingsFile(t *testing.T) {
	const home = "/home/admin"
	at := func(file string) string { return strings.Replace(file, "~", home, 1) }
	for app, file := range map[agents.App][2]string{codex.ID: {at(codex.File), at(claudecode.File)}, claudecode.ID: {at(claudecode.File), at(codex.File)}} {
		files := localtest.NewFS()
		system := localtest.Mac(localtest.NewRunner(t), files, localtest.Env{HomeDir: home, OS: "linux"}, localtest.NewNet(), &localtest.Clock{})
		m := machine{sys: system, vars: map[string]string{}, stores: &stores{}}

		if err := m.agentHooks(app).Install(t.Context()); err != nil {
			t.Fatalf("%s: installing its hook failed: %v", app, err)
		}
		if data, err := files.ReadFile(file[0]); err != nil || !strings.Contains(string(data), "record") || !strings.Contains(string(data), string(app)) {
			t.Errorf("%s: its hook is not in %s (%q, %v)", app, file[0], data, err)
		}
		if _, err := files.ReadFile(file[1]); err == nil {
			t.Errorf("%s: installing its hook wrote %s, the other app's", app, file[1])
		}
	}
}
