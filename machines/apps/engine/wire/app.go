package wire

import (
	"context"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"charm.land/fang/v2"
	"github.com/spf13/cobra"

	awakemacos "github.com/jterrazz/studio-station/apps/engine/awake/adapter/macos"
	awakecontract "github.com/jterrazz/studio-station/apps/engine/awake/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call/lanes"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/engine/process"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/output"
	overviewcalls "github.com/jterrazz/studio-station/apps/engine/overview/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/platform/darwin"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/remote"
	"github.com/jterrazz/studio-station/apps/engine/setup"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli"
	appsinstall "github.com/jterrazz/studio-station/apps/engine/surface/cli/apps/install"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
	shellcmd "github.com/jterrazz/studio-station/apps/engine/surface/cli/shell"
	"github.com/jterrazz/studio-station/apps/engine/ui/help"
	"github.com/jterrazz/studio-station/apps/engine/ui/token"
)

type Program struct {
	// value: the command tree a run answers with, and what closes after it.
	root     *cobra.Command
	writer   *output.Writer
	session  *worldRun
	thisMac  machine
	router   *remote.Router
	reported func()
	vars     map[string]string
}

func Run(args, env []string, stdin, stdout, stderr *os.File) int {
	collectFor(args, varsOf(env))
	p, code := Build(args, env, stdin, stdout, stderr)
	if p == nil {
		return code
	}
	writer := p.writer
	err := fang.Execute(call.WithBy(context.Background(), p.vars["STATION_SESSION"]), p.Root(),
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
	closed := p.Close()
	p.reported()
	if closed != nil && err == nil {
		err = closed
		writer.Failure(kit.Translate(err, ""))
	}
	if err == nil {
		return int(fail.ExitOK)
	}
	return int(fail.ExitOf(kit.Translate(err, "")))
}

func (p *Program) Root() *cobra.Command { return p.root }

func (p *Program) Close() error {
	// invariant: a world's engines take their turn on the command before its
	// pool closes, as a real engine runs beside a live one: a night's upgrade
	// reads this machine's apps.
	var closed error
	if p.session != nil {
		closed = p.session.close(context.Background())
	}
	p.thisMac.pool.Close()
	p.router.Close()
	return closed
}

func Build(args, env []string, stdin, stdout, stderr *os.File) (p *Program, exit int) {
	// value: a run's tree built as Run builds it, never executed; nil and the
	// exit code when this machine could not be read.
	env = asBundledJob(args, env)
	env = withAppBins(env, len(args) > 0 && args[0] == "guest")
	frames, frameIn := stdout, stdin
	var childIn io.Reader = stdin
	if serving(args) {
		// invariant: a run that serves frames owns stdin and stdout for them:
		// every other write of it goes to stderr, no child reads its stdin.
		stdout, childIn = stderr, strings.NewReader("")
	}
	vars := varsOf(env)
	inTTY, outTTY, errTTY := token.IsTerminal(stdin), token.IsTerminal(stdout), token.IsTerminal(stderr)
	session, opened := openWorld(vars, slices.Contains(args, "--page"))
	dark := true
	if inTTY && outTTY {
		// hazard: the background query writes an OSC request and reads the reply
		// on stdin, so it runs only when both ends are a terminal that answers.
		dark = token.DetectDark(stdin, stdout)
	}
	writer := output.NewWriter(
		output.Stream{W: stdout, IsTerminal: outTTY, Width: widthOf(stdout, outTTY), Height: heightOf(stdout, outTTY), Dark: dark},
		output.Stream{W: stderr, IsTerminal: errTTY, Width: widthOf(stderr, errTTY), Dark: dark},
		vars,
	)
	if opened != nil {
		writer.Failure(kit.Translate(opened, ""))
		return nil, int(fail.ExitFailure)
	}
	var world *worldMac
	if session != nil {
		world = session.thisMac()
	}
	builtin, loaded := catalog.Builtin()
	if loaded != nil {
		writer.Failure(kit.Translate(loaded, ""))
		return nil, int(fail.ExitFailure)
	}
	// invariant: the owner's packages are layered before the search is built,
	// so the folders an owner's package lands its commands in (`path:`) are
	// searched as a built-in package's are.
	invoking := local.Invoking{User: catalog.SudoUser, UID: catalog.SudoUID, GID: catalog.SudoGID}
	unsearched := local.NewSearching(env, nil, nil, nil, local.Search{}, invoking)
	cat, broken := layerCatalog(userSource(world, unsearched.FS, pathsOf(unsearched.Env.Home(), vars).Prefs), builtin)
	thisMac := resolve(local.NewSearching(env, childIn, stdout, stderr, landing(cat, vars), invoking), darwin.NewDoor(env), vars, world, cat)
	kenv := kit.Env{
		Out:         writer,
		Prompt:      output.Terminal(stdin, stderr, vars["TERM"] == "dumb"),
		Interactive: inTTY && errTTY,
		On:          &kit.On{ThisMachine: thisMac.thisMac(), Known: thisMac.known},
		Use:         thisMac.uses(),
		Stdin:       frameIn,
		Stdout:      frames,
		Self:        thisMac.profile.Ref,
		Kind:        thisMac.profile.Kind,
		Now:         thisMac.sys.Clock.Now,
		Locate:      thisMac.locate,
	}
	thisMac.refused = refusalOf(broken)
	router := thisMac.router(kenv)
	kenv.Calls = router
	kenv.Install = func(ctx context.Context, pkg string, o kit.IO) error {
		return appsinstall.Run(ctx, appsinstall.Deps{ThisMachine: thisMac.where, Now: thisMac.sys.Clock.Now}, kenv, []string{pkg}, o)
	}
	if session != nil {
		world.engineFrom(func() *process.Engine { return thisMac.startedEngine(kenv) })
		world.doorFrom(thisMac.admits)
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
		machinesCommand(kenv, thisMac),
		devicesCommand(kenv, thisMac),
		imagesCommand(kenv, thisMac),
		engineCommand(kenv, thisMac),
		appsCommand(kenv, thisMac),
		settingsCommand(kenv, thisMac),
		diskCommand(kenv, thisMac),
	}
	commands = append(commands, portsCommands(kenv, thisMac)...)
	commands = append(commands,
		accountsCommand(kenv, thisMac),
		backgroundCommand(kenv, thisMac),
		awakeCommand(kenv, thisMac),
		activityCommand(kenv, thisMac),
		agentsCommand(kenv, thisMac),
		teamsCommand(kenv, thisMac),
		automationsCommand(kenv, thisMac),
	)
	commands = append(commands, initCommands(kenv, thisMac)...)
	commands = append(commands, useCommand(kenv, thisMac), journalCommand(kenv, thisMac), machinefileCommand(kenv, thisMac),
		guestCommand(kenv, thisMac),
		shellcmd.Command(kenv),
	)
	if session != nil {
		commands = append(commands, session.command())
	}
	root := cli.NewRoot(kenv, overviewCommand(kenv, thisMac), commands...)
	reported := gateCatalog(root, writer, broken)
	root.SetArgs(args)
	root.SetIn(stdin)
	root.SetOut(stdout)
	root.SetErr(stderr)
	return &Program{root: root, writer: writer, session: session, thisMac: thisMac, router: router, reported: reported, vars: vars}, 0
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
		dirs = append(dirs, catalog.InstallDirIn(home), filepath.Join(apps, catalog.AppShims))
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
// owners and handed to the platform's search: each catalogue package's Path, of
// the system, and under home the bin a script installs into and the app
// manager's shims.
func landing(cat catalog.Catalog, vars map[string]string) local.Search {
	var s local.Search
	for _, a := range cat.Apps {
		s.System = append(s.System, a.Path...)
	}
	if home := vars["HOME"]; filepath.IsAbs(home) {
		s.Home = []string{catalog.InstallDirIn(home), filepath.Join(appData(home, vars[catalog.AppDataVar]), catalog.AppShims)}
	}
	if onMacOS(vars["STATION_MACHINE_KIND"]) {
		shims := catalog.XcodeShims()
		s.ShimsDir, s.Shims = shims.Dir, shims.Command
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
	cat catalog.Catalog
	// value: the owner's file this run refused, for Needs you; nil when none.
	refused *overviewcalls.Refusal
	profile coremachine.Profile
	vars    map[string]string
	kind    catalog.MachineKind
	paths   setup.Paths
	stores  *stores
	// value: the engine's kept reading of this Mac's sleep timer.
	sleep *sleepTimer
	where page.Context
	named func() string
	// value: this process's one device key, made at its first ask.
	key *heldKey
}

func resolve(raw local.System, door darwin.Door, vars map[string]string, world *worldMac, cat catalog.Catalog) machine {
	h := raw
	if world != nil {
		h = world.system(raw, cat)
	}
	paths := pathsOf(h.Env.Home(), vars)
	st := newStores(h.FS, paths, deviceAnchor(h, vars), enginePeer{socketAPI(h, engine.SocketPath(paths.State))})
	if world != nil {
		world.logAt(paths.State)
		world.stores(st)
	}
	registered, _ := st.name().Get()
	var budgets lanes.Clock = h.Clock
	base := context.Background()
	if world != nil {
		budgets, base = world.budgets(), world.poolContext(base)
	}
	m := machine{
		world:  world,
		raw:    raw,
		door:   door,
		far:    &late{},
		pool:   lanes.NewPool(base, budgets, commandLane, walkLane),
		sys:    h,
		cat:    cat,
		vars:   vars,
		paths:  paths,
		stores: st,
		key:    &heldKey{},
	}
	pin := vars["STATION_MACHINE_KIND"]
	self := selfOf(h, pin)
	m.powerPort = awakemacos.New(door, h.Run, m.systemRoot())
	if world != nil {
		// invariant: a world says what each of its machines is: its Self is
		// read, never the pin every document runs with.
		pin, self, m.powerPort = "", world.self(), world.power()
	}
	m.named = defaultName(h, world != nil)
	name := model.ThisMacName(registered, m.named)
	m.profile = profileOf(context.Background(), self, pin, name, m.powerPort, m.providers)
	m.kind = legacyKind(m.profile)
	m.sleep = newSleepTimer(h.Clock, m.readMacNeverSleeps)
	m.where = page.Context{Machine: name, Kind: string(m.profile.Kind), Registered: registered != ""}
	return m
}

// value: station's folders under a home, each moved by its XDG variable
// when that is an absolute path.
func pathsOf(home string, vars map[string]string) setup.Paths {
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
	return setup.Paths{
		Prefs: filepath.Join(roots.Prefs, catalog.StateFolder),
		Data:  filepath.Join(roots.Data, catalog.StateFolder),
		State: filepath.Join(roots.State, catalog.StateFolder),
		Cache: filepath.Join(roots.Cache, catalog.StateFolder),
		Bin:   catalog.InstallDirIn(home),
		Roots: roots,
	}
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
func (m machine) systemRoot() string { return systemRootOf(m.vars) }

func systemRootOf(vars map[string]string) string {
	if r := vars["STATION_SYSTEM_ROOT"]; filepath.IsAbs(r) {
		return r
	}
	return ""
}

func (m machine) thisMac() string { return m.where.Machine }
