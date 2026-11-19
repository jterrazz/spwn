package wire

import (
	"testing"

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
