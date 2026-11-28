package wire

import (
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local/localtest"
	settingscontract "github.com/jterrazz/studio-station/apps/engine/settings/contract"
)

func TestASettingWritesNothingInStationsOwnFolderWhereXDGMovesIt(t *testing.T) {
	for _, root := range []string{"XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME", "XDG_CACHE_HOME"} {
		home := t.TempDir()
		vars := map[string]string{root: home + "/xdg"}
		system := localtest.Mac(localtest.NewRunner(t), localtest.NewFS(), localtest.Env{HomeDir: home, OS: "darwin"}, localtest.NewNet(), &localtest.Clock{})
		appliers := settingscontract.NewAppliers(settingAppliers(machine{sys: system, vars: vars, paths: pathsOf(home, vars), stores: &stores{}})...)
		for _, mode := range []catalog.WriteMode{catalog.WriteFile, catalog.WriteMerge} {
			it := catalog.Setting{Name: "fmt", Effect: catalog.Dotfiles{Mode: mode, Files: []catalog.Copy{{From: "pkg/x.json", To: "~/xdg/station/x.json"}}}}
			applier, ok := appliers.For(it)
			if !ok {
				t.Fatalf("%s: no applier", mode)
			}
			if _, err := applier.Apply(t.Context(), it, nil); err == nil || !strings.Contains(err.Error(), "station's own") {
				t.Errorf("%s %s: Apply: %v; want ~/xdg/station refused as station's own", root, mode, err)
			}
		}
	}
}

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
