package wire

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/apps"
	appsfake "github.com/jterrazz/studio-station/apps/engine/apps/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/apps/adapter/mise"
	appscalls "github.com/jterrazz/studio-station/apps/engine/apps/calls"
	"github.com/jterrazz/studio-station/apps/engine/apps/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/platform/local/localtest"
	"github.com/jterrazz/studio-station/apps/engine/world"
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
		w.Managers[method] = appsfake.Named(adapter)
		w.Mac("vm").Apps[method] = &world.Manager{Absent: method != in.Method() && method != nodeIn.Method(), Installed: map[string]string{}}
		_, lined := adapter.(contract.Lined)
		registry[method] = appsfake.InstallerOf(w, "vm", method, lined)
	}
	w.Mac("vm").Apps[nodeIn.Method()].Installed[nodeIn.Key()] = "22.11.0"
	foot := appsfake.FootprintOf(w, "vm", cat)
	system.Run, system.FS = appsfake.Seen{Runner: system.Run, Foot: foot, Home: "/home/u"}, appsfake.SeenFS{FS: system.FS, Foot: foot, Home: "/home/u"}
	shelf := appscalls.NewShelf(appsMachine{h: system, run: system.Run}, cat, nil, appscalls.ShelfOptions{ThisMachine: catalog.MacOS, Root: "/", Installers: registry, Bundler: appsfake.BundlerOf(w, "vm")})
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
	run := &everywhere{calls: &[]local.Cmd{}}
	system := localtest.Mac(localtest.NewRunner(t), localtest.NewFS(), localtest.Env{HomeDir: "/home/u", OS: "darwin"}, localtest.NewNet(), &localtest.Clock{})
	system.Run = run
	m := machine{sys: system, vars: map[string]string{}}
	registry, _ := installers(m, false)
	cat, err := catalog.Builtin()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	appsMachineOf(m).RunBriefly(ctx, "true", nil, time.Second)
	if len(*run.calls) != 1 {
		t.Fatalf("an app's version read ran %d commands; want 1", len(*run.calls))
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
			install, remove := i.Install, i.Remove
			if h, ok := i.(contract.Holder); ok {
				f := contract.For{App: a.Name, Word: true}
				install = func(ctx context.Context, in []catalog.Install, logs contract.Logs) error {
					return h.InstallFor(ctx, f, in, logs)
				}
				remove = func(ctx context.Context, in []catalog.Install, logs contract.Logs) error {
					return h.RemoveFor(ctx, f, in, logs)
				}
			}
			_ = install(ctx, []catalog.Install{in}, nil)
			before := len(*run.calls)
			_ = remove(ctx, []catalog.Install{in}, nil)
			removed[in.Method()] = len(*run.calls) > before
		}
	}
	for method, i := range registry {
		// value: a release station fetches itself, handed no Runner: it
		// runs no command for the environment to reach.
		if method == catalog.MethodGitHubRelease {
			continue
		}
		if i.Can().Remove && !removed[method] {
			t.Errorf("%s: no removal ran a command, so the test reads nothing of it", method)
		}
	}
	for _, c := range *run.calls {
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

// value: a Runner that answers every command with an empty document and
// records it with the environment it was given before its own Env.
type everywhere struct {
	calls *[]local.Cmd
	env   []string
}

func (e *everywhere) Output(_ context.Context, c local.Cmd) (local.Output, error) {
	c.Env = append(append([]string{}, e.env...), c.Env...)
	*e.calls = append(*e.calls, c)
	return local.Output{Stdout: "{}"}, nil
}

func (e *everywhere) Attach(ctx context.Context, c local.Cmd, _ local.Stream) (local.Attached, error) {
	_, err := e.Output(ctx, c)
	return local.Attached{}, err
}

func (e *everywhere) Start(c local.Cmd, _ string) (int, error) {
	_, err := e.Output(context.Background(), c)
	return 0, err
}

func (e *everywhere) Within(env []string) local.Runner { return &everywhere{calls: e.calls, env: env} }

func (e *everywhere) LookPath(name string) (string, bool) { return "/bin/" + name, true }
func (e *everywhere) Alive(int) bool                      { return false }
func (e *everywhere) Signal(int, local.Sig) error         { return nil }
func (e *everywhere) Reap(string) int                     { return 0 }

func TestAManagerAnswersForTheGlobalVersionsWhateverPathStationWasGiven(t *testing.T) {
	// invariant: a shell where the runtime manager is activated puts a
	// project's install folder first on PATH, so a package manager found
	// there would list what that runtime holds, while the runtime's removal
	// takes back every global one: the lookup and every command's PATH are
	// the global ones', and the packages are read and removed in each of
	// their folders.
	home := "/home/u"
	data := home + "/" + catalog.AppData
	project := data + "/installs/node/22.1.0/bin"
	shims := data + "/" + catalog.AppShims
	cat, err := catalog.Builtin()
	if err != nil {
		t.Fatal(err)
	}
	vars := map[string]string{"HOME": home, "PATH": project + ":/usr/bin"}
	run := localtest.NewRunner(t)
	run.Env = []string{"HOME=" + home, "PATH=" + vars["PATH"]}
	system := localtest.Mac(run, localtest.NewFS(), localtest.Env{HomeDir: home, OS: "darwin"}, localtest.NewNet(), &localtest.Clock{})
	m := machine{sys: system, vars: vars}
	registry, _ := installers(m, false)
	runtime, packages := installsOf(t, cat, registry, "node")
	runtimes, pkgs := registry[runtime.Method()], registry[packages.Method()]
	rt, pm := runtimes.Manager().Name, pkgs.Manager().Name
	older, global := data+"/installs/node/20.0.0", data+"/installs/node/24.22.2"
	run.Programs = []string{project + "/" + pm, shims + "/" + pm, "/usr/bin/" + rt}
	run.On(rt, "ls", "--global", "--json").Reply(local.Output{Stdout: `{"node":[{"version":"20.0.0","install_path":"` + older + `","installed":true},{"version":"24.22.2","install_path":"` + global + `","installed":true}]}`})
	run.On(pm, "ls", "-g", "--depth=0", "--json", "--prefix", older).Reply(local.Output{Stdout: `{"dependencies":{}}`})
	run.On(pm, "ls", "-g", "--depth=0", "--json", "--prefix", global).Reply(local.Output{Stdout: `{"dependencies":{"owner-pkg":{"version":"1.0.0"}}}`})
	run.On(pm, "uninstall", "-g", "--prefix", older, packages.Key())
	run.On(pm, "uninstall", "-g", "--prefix", global, packages.Key())
	run.On(rt, "unuse", "--global", "--no-prune", runtime.Key())
	ctx := context.Background()

	if found, _ := managing(m).Run.LookPath(pm); found != shims+"/"+pm {
		t.Errorf("%s is looked up at %q; want the shim %s", pm, found, shims+"/"+pm)
	}
	if held, err := pkgs.Inventory(ctx); err != nil || len(held) != 1 {
		t.Errorf("%s lists %v (%v); want the package of the second global node", pm, held, err)
	}
	_ = pkgs.Remove(ctx, []catalog.Install{packages}, nil)
	_ = runtimes.Remove(ctx, []catalog.Install{runtime}, nil)

	if len(run.Calls) != 7 {
		t.Fatalf("ran %d commands; want each read of the global nodes, the read and the removal of the package in each, and the runtime's removal", len(run.Calls))
	}
	for _, c := range run.Calls {
		argv := c.Name + " " + strings.Join(c.Args, " ")
		search := ""
		for _, kv := range c.Env {
			if v, ok := strings.CutPrefix(kv, "PATH="); ok {
				search = v
			}
		}
		if search != shims+":/usr/bin" {
			t.Errorf("%s runs with PATH %q; want the shims, then station's PATH without the project's folder", argv, search)
		}
		if c.Dir != home {
			t.Errorf("%s runs from %q; want the home folder", argv, c.Dir)
		}
	}
}

func TestAPackageIsLookedForAndReadOnTheOwnersPath(t *testing.T) {
	// invariant: a manager command's PATH leaves nvm's versions out, so an
	// item looked for there would read absent, never present another way.
	home := "/home/u"
	own := home + "/.nvm/versions/node/v24.22.2/bin"
	vars := map[string]string{"HOME": home, "PATH": own + ":/usr/bin"}
	run := localtest.NewRunner(t)
	run.Env = []string{"HOME=" + home, "PATH=" + vars["PATH"]}
	run.Programs = []string{own + "/node"}
	run.On("node", "--version").Reply(local.Output{Stdout: "v24.22.2"})
	system := localtest.Mac(run, localtest.NewFS(), localtest.Env{HomeDir: home, OS: "darwin"}, localtest.NewNet(), &localtest.Clock{})
	a := appsMachineOf(machine{sys: system, vars: vars})

	if found, ok := a.LookPath("node"); !ok || found != own+"/node" {
		t.Errorf("node is found at %q; want nvm's, on the owner's PATH", found)
	}
	if out, code, ran := a.RunBriefly(context.Background(), "node", []string{"--version"}, time.Second); !ran || code != 0 || out != "v24.22.2" {
		t.Errorf("node's version read %q, exit %d, ran %v; want nvm's node's answer", out, code, ran)
	}
	for _, c := range run.Calls {
		if c.Dir != home {
			t.Errorf("%s runs from %q; want the home folder", c.Name, c.Dir)
		}
		for _, kv := range c.Env {
			if v, ok := strings.CutPrefix(kv, "PATH="); ok && !strings.Contains(v, own) {
				t.Errorf("%s runs on PATH %q, which has not nvm's node", c.Name, v)
			}
		}
	}
}

// value: the runtime app's install on a Linux workspace, and one install of
// the method whose manager that runtime is, on a macOS workspace.
func installsOf(t *testing.T, cat catalog.Catalog, registry contract.Installers, runtime string) (own, held catalog.Install) {
	t.Helper()
	for _, a := range cat.Apps {
		if a.Name == runtime {
			own, _ = a.InstallOn(catalog.Linux)
		}
		if in, ok := a.InstallOn(catalog.MacOS); ok && held == nil && registry[in.Method()].Manager().App == runtime {
			held = in
		}
	}
	if own == nil || held == nil {
		t.Fatalf("the catalogue has no %s install, or no install it manages", runtime)
	}
	return own, held
}
