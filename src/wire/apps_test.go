package wire

import (
	"testing"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/platform/local/localtest"
)

func TestEveryMethodTheCatalogueKindsHasItsInstaller(t *testing.T) {
	system := localtest.Mac(localtest.NewRunner(t), localtest.NewFS(), localtest.Env{HomeDir: "/home/u", OS: "darwin"}, localtest.NewNet(), &localtest.Clock{})

	registry, _ := installers(machine{sys: system, vars: map[string]string{}}, false)

	for m := range catalog.MethodKinds {
		if i, ok := registry[m]; !ok || i.Method() != m {
			t.Errorf("%s: placed by the catalogue, installed by no adapter", m)
		}
	}
	if len(registry) != len(catalog.MethodKinds) {
		t.Errorf("%d installers for the %d methods the catalogue places", len(registry), len(catalog.MethodKinds))
	}
}

func TestEveryManagerIsAnAppThatLivesWhereItsMethodInstalls(t *testing.T) {
	// invariant: an install brings its method's manager, so the app that
	// provides it lives on every kind of machine the method installs on.
	system := localtest.Mac(localtest.NewRunner(t), localtest.NewFS(), localtest.Env{HomeDir: "/home/u", OS: "darwin"}, localtest.NewNet(), &localtest.Clock{})
	registry, _ := installers(machine{sys: system, vars: map[string]string{}}, false)
	apps := map[string]catalog.App{}
	c, err := catalog.Builtin()
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range c.Apps {
		apps[a.Name] = a
	}
	for m, i := range registry {
		name := i.Manager().App
		if name == "" {
			continue
		}
		app, ok := apps[name]
		if !ok {
			t.Errorf("%s: its manager is the unknown app %q", m, name)
			continue
		}
		for _, k := range catalog.MethodKinds[m] {
			if _, placed := app.InstallOn(k); !placed {
				t.Errorf("%s installs on %s, where its manager %s does not live", m, k, name)
			}
		}
	}
}
