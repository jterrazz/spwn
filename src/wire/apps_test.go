package wire

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/apps"
	appsfake "github.com/jterrazz/studio-station/src/apps/adapter/fake"
	"github.com/jterrazz/studio-station/src/apps/adapter/mise"
	appscalls "github.com/jterrazz/studio-station/src/apps/calls"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/platform/local"
	"github.com/jterrazz/studio-station/src/platform/local/localtest"
	"github.com/jterrazz/studio-station/src/world"
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
	byName := map[string]catalog.App{}
	c, err := catalog.Builtin()
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range c.Apps {
		byName[a.Name] = a
	}
	for m, i := range registry {
		name := i.Manager().App
		if name == "" {
			continue
		}
		app, ok := byName[name]
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

func TestEveryManagerAnAdapterNamesIsTheCatalogs(t *testing.T) {
	// invariant: the app an adapter names when its manager is missing is
	// the one the catalog says manages its method, the one apps.install
	// brings: the two never disagree.
	system := localtest.Mac(localtest.NewRunner(t), localtest.NewFS(), localtest.Env{HomeDir: "/home/u", OS: "darwin"}, localtest.NewNet(), &localtest.Clock{})
	registry, _ := installers(machine{sys: system, vars: map[string]string{}}, false)
	c, err := catalog.Builtin()
	if err != nil {
		t.Fatal(err)
	}
	for m, i := range registry {
		if owner, _ := c.Manager(string(m)); owner != i.Manager().App {
			t.Errorf("%s: its adapter names the manager %q, the catalog %q", m, i.Manager().App, owner)
		}
	}
}

func TestInstallBringsWhatTheCatalogLinksAndNoMore(t *testing.T) {
	// invariant: an install brings what the catalog links the app to by
	// the install this machine carries, never what its method's adapter
	// names: playwright-mcp on a macOS VM leans on node, which is there,
	// so with every other manager missing, nothing else joins the run.
	cat, err := catalog.Builtin()
	if err != nil {
		t.Fatal(err)
	}
	app, _ := cat.App("playwright-mcp")
	in, _ := app.InstallOn(catalog.MacOS)
	node, _ := cat.App("node")
	nodeIn, _ := node.InstallOn(catalog.MacOS)
	system := localtest.Mac(localtest.NewRunner(t), localtest.NewFS(), localtest.Env{HomeDir: "/home/u", OS: "darwin"}, localtest.NewNet(), &localtest.Clock{})
	adapters, _ := installers(machine{sys: system, vars: map[string]string{}}, false)
	w := &world.World{ThisMac: "vm", Managers: map[catalog.Method]world.Named{}}
	w.Mac("vm").Apps = map[catalog.Method]*world.Manager{}
	registry := contract.Installers{}
	for method, adapter := range adapters {
		w.Managers[method] = world.Named{Name: adapter.Manager().Name, App: adapter.Manager().App}
		w.Mac("vm").Apps[method] = &world.Manager{Absent: method != in.Method() && method != nodeIn.Method(), Installed: map[string]string{}}
		_, lined := adapter.(contract.Lined)
		registry[method] = appsfake.InstallerOf(w, "vm", method, lined)
	}
	w.Mac("vm").Apps[nodeIn.Method()].Installed[nodeIn.Key()] = "22.11.0"
	shelf := appscalls.NewShelf(appsMachine{system}, cat, nil, appscalls.ShelfOptions{ThisMachine: catalog.MacOS, Root: "/", Installers: registry, Bundler: appsfake.BundlerOf(w, "vm")})
	out, err := shelf.Install(context.Background(), apps.InstallRequest{Names: []string{app.Name}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	ran := make([]string, 0, len(out))
	for _, o := range out {
		ran = append(ran, o.Name)
	}
	if !slices.Equal(ran, []string{app.Name}) {
		t.Errorf("the run was %v; want [%s]: node is there, and nothing else is linked", ran, app.Name)
	}
}

func TestEveryManagerCommandRunsFromHomeOnTheGlobalVersions(t *testing.T) {
	// invariant: a manager answers for the owner's global tools wherever
	// station was started: a project folder that pins another runtime would
	// otherwise pick which manager lists what that runtime holds, while its
	// removal takes back the global one.
	run := &everywhere{}
	system := localtest.Mac(localtest.NewRunner(t), localtest.NewFS(), localtest.Env{HomeDir: "/home/u", OS: "darwin"}, localtest.NewNet(), &localtest.Clock{})
	system.Run = run
	m := machine{sys: system, vars: map[string]string{}}
	registry, _ := installers(m, false)
	cat, err := catalog.Builtin()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	appsMachine{managing(m)}.RunBriefly(ctx, "true", nil, time.Second)
	if len(run.calls) != 1 {
		t.Fatalf("an app's version read ran %d commands; want 1", len(run.calls))
	}
	removed := map[catalog.Method]bool{}
	for _, a := range cat.Apps {
		for _, k := range []catalog.MachineKind{catalog.MacOS, catalog.Linux} {
			in, ok := a.InstallOn(k)
			if !ok || removed[in.Method()] || !registry[in.Method()].Can().Remove {
				continue
			}
			i := registry[in.Method()]
			_, _ = i.Inventory(ctx)
			_ = i.Install(ctx, []catalog.Install{in}, nil)
			before := len(run.calls)
			_ = i.Remove(ctx, []catalog.Install{in}, nil)
			removed[in.Method()] = len(run.calls) > before
		}
	}
	for method, i := range registry {
		if i.Can().Remove && !removed[method] {
			t.Errorf("%s: no removal ran a command, so the test reads nothing of it", method)
		}
	}
	for _, c := range run.calls {
		argv := c.Name + " " + strings.Join(c.Args, " ")
		if c.Dir != "/home/u" {
			t.Errorf("%s runs from %q; want the home folder", argv, c.Dir)
		}
		for _, kv := range mise.Pinned("/home/u") {
			if !slices.Contains(c.Env, kv) {
				t.Errorf("%s runs without %s", argv, kv)
			}
		}
	}
}

type everywhere struct{ calls []local.Cmd }

func (e *everywhere) Output(_ context.Context, c local.Cmd) (local.Output, error) {
	e.calls = append(e.calls, c)
	return local.Output{Stdout: "{}"}, nil
}

func (e *everywhere) Attach(ctx context.Context, c local.Cmd, _ local.Stream) (local.Attached, error) {
	_, err := e.Output(ctx, c)
	return local.Attached{}, err
}

func (e *everywhere) Start(c local.Cmd, _ string) (int, error) {
	e.calls = append(e.calls, c)
	return 0, nil
}

func (e *everywhere) LookPath(name string) (string, bool) { return "/bin/" + name, true }
func (e *everywhere) Alive(int) bool                      { return false }
func (e *everywhere) Signal(int, local.Sig) error         { return nil }
func (e *everywhere) Reap(string) int                     { return 0 }
