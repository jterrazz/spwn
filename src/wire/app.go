package wire

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"charm.land/fang/v2"
	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/src/agents/adapter/herdr"
	awakemacos "github.com/jterrazz/studio-station/src/awake/adapter/macos"
	awakecontract "github.com/jterrazz/studio-station/src/awake/contract"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call/lanes"
	"github.com/jterrazz/studio-station/src/core/fail"
	coremachine "github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/engine/process"
	"github.com/jterrazz/studio-station/src/output"
	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/platform/darwin"
	"github.com/jterrazz/studio-station/src/platform/local"
	"github.com/jterrazz/studio-station/src/setup"
	"github.com/jterrazz/studio-station/src/surface/cli"
	appsinstall "github.com/jterrazz/studio-station/src/surface/cli/apps/install"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
	shellcmd "github.com/jterrazz/studio-station/src/surface/cli/shell"
	"github.com/jterrazz/studio-station/src/ui/help"
	"github.com/jterrazz/studio-station/src/ui/token"
)

func Run(args, env []string, stdin, stdout, stderr *os.File) int {
	env = withAppBins(env, len(args) > 0 && args[0] == "guest")
	frames, frameIn := stdout, stdin
	var childIn io.Reader = stdin
	if serving(args) {
		// invariant: a run that serves frames owns stdin and stdout for them:
		// every other write of it goes to stderr, no child reads its stdin.
		stdout, childIn = stderr, strings.NewReader("")
	}
	vars := map[string]string{}
	for _, kv := range env {
		if k, v, ok := strings.Cut(kv, "="); ok {
			vars[k] = v
		}
	}
	inTTY, outTTY, errTTY := token.IsTerminal(stdin), token.IsTerminal(stdout), token.IsTerminal(stderr)
	session, opened := openWorld(vars, slices.Contains(args, "--page"))
	dark := true
	if inTTY && outTTY && len(args) > 0 {
		// hazard: the background query writes an OSC request and reads the reply
		// on stdin, so it runs only when both ends are a terminal that answers; bare
		// `station` opens the TUI, which asks itself, and a key typed early must
		// reach it rather than be swallowed as the reply.
		dark = token.DetectDark(stdin, stdout)
	}
	writer := output.NewWriter(
		output.Stream{W: stdout, IsTerminal: outTTY, Width: widthOf(stdout, outTTY), Height: heightOf(stdout, outTTY), Dark: dark},
		output.Stream{W: stderr, IsTerminal: errTTY, Width: widthOf(stderr, errTTY), Dark: dark},
		vars,
	)
	if opened != nil {
		writer.Failure(kit.Translate(opened, ""))
		return int(fail.ExitFailure)
	}
	var world *worldMac
	if session != nil {
		world = session.thisMac()
	}
	cat, loaded := catalog.Builtin()
	if loaded != nil {
		writer.Failure(kit.Translate(loaded, ""))
		return int(fail.ExitFailure)
	}
	thisMac := resolve(local.NewSearching(env, childIn, stdout, stderr, landing(cat, vars)), darwin.NewDoor(env), vars, world, cat)
	kenv := kit.Env{
		Out:         writer,
		Prompt:      output.Terminal(stdin, stderr, vars["TERM"] == "dumb"),
		Interactive: inTTY && errTTY,
		Screen:      inTTY && outTTY,
		On:          &kit.On{ThisMachine: thisMac.thisMac(), Known: thisMac.known},
		Use:         thisMac.uses(),
		Stdin:       frameIn,
		Stdout:      frames,
		Self:        thisMac.profile.Ref,
		Kind:        thisMac.profile.Kind,
		Now:         thisMac.sys.Clock.Now,
		Locate:      thisMac.locate,
	}
	router := thisMac.router(kenv)
	defer router.Close()
	kenv.Calls = router
	kenv.Install = func(ctx context.Context, item string, o kit.IO) error {
		return appsinstall.Run(ctx, appsinstall.Deps{ThisMachine: thisMac.where, Now: thisMac.sys.Clock.Now}, kenv, []string{item}, o)
	}
	if session != nil {
		world.engineFrom(func() *process.Engine { return thisMac.engine(kenv) })
		session.farFrom(func(w *worldMac) farMac { return thisMac.farMac(kenv, w) })
	}
	kenv.On.Refresh = func(ctx context.Context) error {
		allMachines := thisMac.machines(false)
		f, err := allMachines.AllMachines(ctx, false)
		if err != nil {
			return err
		}
		return allMachines.SSHBlocks(ctx, f)
	}

	commands := []*cobra.Command{
		activityCommand(kenv, thisMac),
		appsCommand(kenv, thisMac),
		settingsCommand(kenv, thisMac),
		diskCommand(kenv, thisMac),
	}
	commands = append(commands, portsCommands(kenv, thisMac)...)
	commands = append(commands,
		accountsCommand(kenv, thisMac),
		backgroundCommand(kenv, thisMac),
		awakeCommand(kenv, thisMac),
		machinesCommand(kenv, thisMac),
		imagesCommand(kenv, thisMac),
		engineCommand(kenv, thisMac),
		useCommand(kenv, thisMac),
	)
	commands = append(commands, initCommands(kenv, thisMac)...)
	commands = append(commands,
		statusCommand(kenv, thisMac),
		appsAlias(kenv, thisMac),
		storageAlias(kenv, thisMac),
		secretsAlias(kenv, thisMac),
		retiredEngineCommand(kenv, thisMac),
		guestCommand(kenv, thisMac),
		agentsCommand(kenv, thisMac.where, agentsDeps{Terminal: thisMac.vars[herdr.TerminalVar], Attach: thisMac.multiplexer().Attach}),
		runCommand(kenv, thisMac),
		shellcmd.Command(kenv),
	)
	commands = append(commands, engineAliases(kenv, thisMac)...)
	commands = append(commands, settingsAliases(kenv, thisMac)...)
	commands = append(commands, awakeAliases(kenv, thisMac)...)
	if session != nil {
		commands = append(commands, session.command())
	}
	root := cli.NewRoot(kenv, tuiOpener(kenv, thisMac, vars["NO_COLOR"] != ""), overviewCommand(kenv, thisMac), commands...)
	root.SetArgs(args)
	root.SetIn(stdin)
	root.SetOut(stdout)
	root.SetErr(stderr)

	err := fang.Execute(context.Background(), root,
		fang.WithVersion(Version()),
		fang.WithCommit(Commit()),
		fang.WithErrorHandler(func(_ io.Writer, _ fang.Styles, err error) {
			worded := kit.Translate(err, "")
			if wantsJSON(args) {
				writer.FailureJSON(worded)
			}
			writer.Failure(worded)
		}),
		fang.WithColorSchemeFunc(help.FangScheme),
		fang.WithNotifySignal(os.Interrupt, syscall.SIGTERM),
	)
	thisMac.pool.Close()
	if session != nil {
		if closed := session.close(context.Background()); closed != nil && err == nil {
			err = closed
			writer.Failure(kit.Translate(err, ""))
		}
	}
	if err == nil {
		return int(fail.ExitOK)
	}
	return int(fail.ExitOf(kit.Translate(err, "")))
}

// value: a run that asked --json, before any `--`: its refusal is a
// document too, even one cobra refused before any command ran.
func wantsJSON(args []string) bool {
	for _, a := range args {
		switch a {
		case "--":
			return false
		case "--json", "--json=true":
			return true
		}
	}
	return false
}

// value: commands 16 at a time (each is its own process; status starts ~70),
// walks one at a time (one walk is eight workers, the measured knee).
const (
	commandLane = 16
	walkLane    = 1
)

func heightOf(f *os.File, isTerminal bool) int {
	if !isTerminal {
		return 0
	}
	_, h := token.Size(f)
	return h
}

func widthOf(f *os.File, isTerminal bool) int {
	if !isTerminal {
		return 0
	}
	w, _ := token.Size(f)
	return w
}

// value: the shell loader puts the global bins of the catalogue on an
// interactive PATH; station appends them to its own, so what it installed
// there answers a non-interactive caller too, and the caller's PATH still
// wins. A VM's guest is started over ssh with the system's PATH, which also
// lacks the shell block's ~/.local/bin and the app manager's runtimes: it
// appends those first.
func withAppBins(env []string, guest bool) []string {
	home, apps, path, at := "", "", "", -1
	for i, kv := range env {
		switch k, v, _ := strings.Cut(kv, "="); k {
		case "HOME":
			home = v
		case catalog.AppDataVar:
			apps = v
		case "PATH":
			path, at = v, i
		}
	}
	if home == "" {
		return env
	}
	apps = appData(home, apps)
	have := strings.Split(path, ":")
	var dirs []string
	if guest {
		dirs = append(dirs, filepath.Join(home, ".local", "bin"), filepath.Join(apps, catalog.AppShims))
	}
	for _, bin := range catalog.GlobalBins {
		dirs = append(dirs, filepath.Join(home, bin))
	}
	for _, dir := range dirs {
		if !slices.Contains(have, dir) {
			if path != "" {
				path += ":"
			}
			path += dir
		}
	}
	out := slices.Clone(env)
	if at < 0 {
		return append(out, "PATH="+path)
	}
	out[at] = "PATH=" + path
	return out
}

func appData(home, moved string) string {
	if filepath.IsAbs(moved) {
		return moved
	}
	return filepath.Join(home, catalog.AppData)
}

// value: the folders an install lands its commands in, declared by their
// owners and handed to the platform's search: each catalogue item's Path, of
// the system, and under home the bin a script installs into and the app
// manager's shims.
func landing(cat catalog.Catalog, vars map[string]string) local.Search {
	var s local.Search
	for _, a := range cat.Apps {
		s.System = append(s.System, a.Path...)
	}
	if home := vars["HOME"]; filepath.IsAbs(home) {
		s.Home = []string{filepath.Join(home, ".local", "bin"), filepath.Join(appData(home, vars[catalog.AppDataVar]), catalog.AppShims)}
	}
	return s
}

type machine struct {
	// value: this machine's Mac of the spec world, its fakes' slice; nil in
	// every real run.
	world *worldMac
	// value: the system and power door before any world wraps them, from which
	// another Mac of the world is built.
	raw       local.System
	door      darwin.Door
	far       *late
	pool      *lanes.Pool
	sys       local.System
	powerPort awakecontract.Power
	// value: the catalog this run reads, built once.
	cat     catalog.Catalog
	profile coremachine.Profile
	vars    map[string]string
	kind    catalog.MachineKind
	paths   setup.Paths
	stores  *stores
	where   page.Context
}

func resolve(raw local.System, door darwin.Door, vars map[string]string, world *worldMac, cat catalog.Catalog) machine {
	h := raw
	if world != nil {
		h = world.system(raw)
	}
	home := h.Env.Home()
	root := func(name, fallback string) string {
		if dir := vars[name]; filepath.IsAbs(dir) {
			return dir
		}
		return filepath.Join(home, fallback)
	}
	roots := setup.Roots{
		Prefs: root("XDG_CONFIG_HOME", ".config"),
		Data:  root("XDG_DATA_HOME", filepath.Join(".local", "share")),
		State: root("XDG_STATE_HOME", filepath.Join(".local", "state")),
		Cache: root("XDG_CACHE_HOME", ".cache"),
	}
	paths := setup.Paths{
		Prefs: filepath.Join(roots.Prefs, "station"),
		Data:  filepath.Join(roots.Data, "station"),
		State: filepath.Join(roots.State, "station"),
		Cache: filepath.Join(roots.Cache, "station"),
		Bin:   filepath.Join(home, ".local", "bin"),
		Roots: roots,
	}
	st := newStores(h.FS, paths)
	if world != nil {
		world.stores(st)
	}
	registered, _ := st.name().Get()
	var budgets lanes.Clock = h.Clock
	if world != nil {
		budgets = world.budgets()
	}
	m := machine{
		world:  world,
		raw:    raw,
		door:   door,
		far:    &late{},
		pool:   lanes.NewPool(context.Background(), budgets, commandLane, walkLane),
		sys:    h,
		cat:    cat,
		vars:   vars,
		paths:  paths,
		stores: st,
	}
	pin := vars["STATION_MACHINE_KIND"]
	self := selfOf(h, pin)
	m.powerPort = awakemacos.New(door, h.Run, m.systemRoot())
	if world != nil {
		// invariant: a world says what each of its machines is: its Self is
		// read, never the pin every document runs with.
		pin, self, m.powerPort = "", world.self(), world.power()
	}
	name := thisMacName(registered, h.Env.Hostname())
	m.profile = profileOf(context.Background(), self, pin, name, m.powerPort, m.providers)
	m.kind = legacyKind(m.profile)
	m.where = page.Context{Machine: name, Kind: string(m.profile.Kind), Registered: registered != ""}
	return m
}

// value: the name the fleet shows this machine by: the one setup gave it, else
// its hostname, as machines names it (one name in every page, §13.5).
func thisMacName(registered, hostname string) string {
	if registered != "" {
		return registered
	}
	return strings.ToLower(hostname)
}

func (m machine) or(name, fallback string) string {
	if v := m.vars[name]; v != "" {
		return v
	}
	return fallback
}

// value: the root absolute catalogue paths (/Applications, /opt/homebrew,
// /Library, /usr/local) are read under; the spec harness points it into the
// sandbox.
func (m machine) systemRoot() string {
	if r := m.vars["STATION_SYSTEM_ROOT"]; filepath.IsAbs(r) {
		return r
	}
	return ""
}

func (m machine) thisMac() string {
	thisMac, _ := m.stores.name().Get()
	return thisMac
}
