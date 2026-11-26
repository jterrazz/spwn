package wire

import (
	"context"
	"io/fs"
	"maps"
	"path"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/darwin"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/platform/local/localtest"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

// value: the most an idle Mac engine may spawn a program or open a file:
// idlePerMinute for each minute it is up (no cadence of the engine is shorter
// than 9 s), plus idleStartup in all for the first readings and the readings
// of the quarter hour. A reader that asks every tick, as the sleep timer's
// did, is over it within the test's quarter hour.
const (
	idlePerMinute = 7
	idleStartup   = 8
)

type idleClock struct {
	mu sync.Mutex
	at time.Time
}

func (c *idleClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.at
}

func (c *idleClock) advance(d time.Duration) {
	c.mu.Lock()
	c.at = c.at.Add(d)
	c.mu.Unlock()
}

func (c *idleClock) Sleep(ctx context.Context, _ time.Duration) error { return ctx.Err() }

func (c *idleClock) Await(ctx context.Context, _ <-chan struct{}, _ time.Duration) error {
	return ctx.Err()
}

func (c *idleClock) AwaitWhile(ctx context.Context, _, _ <-chan struct{}, _ time.Duration) error {
	return ctx.Err()
}

type spawnCount struct {
	mu       sync.Mutex
	spawned  map[string]int
	inflight int
}

func (r *spawnCount) begin(c local.Cmd) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.spawned[path.Base(c.Name)]++
	r.inflight++
}

func (r *spawnCount) end() {
	r.mu.Lock()
	r.inflight--
	r.mu.Unlock()
}

func (r *spawnCount) quiet() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.inflight == 0
}

func (r *spawnCount) settle() {
	for range 100000 {
		runtime.Gosched()
		if r.quiet() {
			runtime.Gosched()
			runtime.Gosched()
			if r.quiet() {
				return
			}
		}
	}
}

func (r *spawnCount) Output(_ context.Context, c local.Cmd) (local.Output, error) {
	defer r.end()
	r.begin(c)
	return local.Output{}, nil
}

func (r *spawnCount) Attach(_ context.Context, c local.Cmd, _ local.Stream) (local.Attached, error) {
	defer r.end()
	r.begin(c)
	return local.Attached{}, nil
}

func (r *spawnCount) Start(c local.Cmd, _ string) (int, error) {
	defer r.end()
	r.begin(c)
	return 0, local.ErrNotFound
}

func (r *spawnCount) LookPath(name string) (string, bool) { return "/usr/bin/" + name, true }
func (r *spawnCount) Within([]string) local.Runner        { return r }
func (r *spawnCount) Alive(int) bool                      { return false }
func (r *spawnCount) Signal(int, local.Sig) error         { return nil }
func (r *spawnCount) Reap(string) int                     { return 0 }

type openCount struct {
	*localtest.FS
	mu    sync.Mutex
	opens map[string]int
}

func (f *openCount) lock() func() {
	f.mu.Lock()
	return f.mu.Unlock
}

func (f *openCount) ReadFile(p string) ([]byte, error) {
	defer f.lock()()
	f.opens[strings.TrimPrefix(p, "/")]++
	return f.FS.ReadFile(p)
}

func (f *openCount) Open(p string) (fs.File, error) {
	defer f.lock()()
	f.opens[strings.TrimPrefix(p, "/")]++
	return f.FS.Open(p)
}

func (f *openCount) WriteFile(p string, data []byte, perm fs.FileMode) error {
	defer f.lock()()
	return f.FS.WriteFile(p, data, perm)
}

func (f *openCount) Append(p string, data []byte, perm fs.FileMode) error {
	defer f.lock()()
	return f.FS.Append(p, data, perm)
}

func (f *openCount) Replace(p string, data []byte, perm fs.FileMode) error {
	defer f.lock()()
	return f.FS.Replace(p, data, perm)
}

func (f *openCount) MkdirAll(p string, perm fs.FileMode) error {
	defer f.lock()()
	return f.FS.MkdirAll(p, perm)
}

func (f *openCount) Remove(p string) error {
	defer f.lock()()
	return f.FS.Remove(p)
}

func (f *openCount) RemoveAll(p string) error {
	defer f.lock()()
	return f.FS.RemoveAll(p)
}

func (f *openCount) Rename(from, to string) error {
	defer f.lock()()
	return f.FS.Rename(from, to)
}

func (f *openCount) Stat(p string) (fs.FileInfo, error) {
	defer f.lock()()
	return f.FS.Stat(p)
}

func (f *openCount) ReadDir(p string) ([]fs.DirEntry, error) {
	defer f.lock()()
	f.opens[strings.TrimPrefix(p, "/")]++
	return f.FS.ReadDir(p)
}

func (f *openCount) Glob(p string) ([]string, error) {
	defer f.lock()()
	return f.FS.Glob(p)
}

func (f *openCount) WalkDir(root string, fn fs.WalkDirFunc) error {
	defer f.lock()()
	return f.FS.WalkDir(root, fn)
}

type idleSockets struct{ local.Sockets }

func (idleSockets) SendUDP(context.Context, string, []byte) error { return nil }

type noPower struct{}

func (noPower) HoldIdle(string) (func(), error)       { return func() {}, nil }
func (noPower) SetClamshellSleep(bool) error          { return nil }
func (noPower) Read() (darwin.Reading, error)         { return darwin.Reading{}, darwin.ErrPowerUnsupported }
func (noPower) Ticks(context.Context) <-chan struct{} { return nil }
func (noPower) Sleeping() bool                        { return false }

// invariant: an idle Mac engine, nine hundred ticks on a clock that moves a
// second a tick, spawns each program and opens each file no more often than the
// tables above say, whatever the wall time the test takes.
func TestAnIdleMacEngineSpawnsAndOpensWithinItsBudgets(t *testing.T) {
	const ticks = 900
	const prefs = "{\n  \"v\": 1,\n  \"name\": \"here\"\n}\n"
	const record = "{\n  \"v\": 2,\n  \"name\": \"api\",\n  \"image\": \"studio-linux\",\n  \"lifetime\": \"permanent\"\n}\n"
	clock := &idleClock{at: time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)}
	run := &spawnCount{spawned: map[string]int{}}
	files := &openCount{FS: localtest.NewFS(), opens: map[string]int{}}
	_ = files.FS.WriteFile("/home/u/.config/station/preferences.json", []byte(prefs), 0o600)
	_ = files.FS.WriteFile("/home/u/.local/state/station/vms/api.json", []byte(record), 0o600)
	env := localtest.Env{HomeDir: "/home/u", Login: "u", UID: 501, OS: "darwin", Arch: "arm64", Exe: "/usr/local/bin/station", Mac: "here"}
	sys := local.System{Run: run, FS: files, Env: env, Net: localtest.NewNet(), Sockets: idleSockets{}, Clock: clock}
	cat, err := catalog.Builtin()
	if err != nil {
		t.Fatal(err)
	}
	vars := map[string]string{"HOME": "/home/u", "STATION_MACHINE_KIND": "mac"}
	m := resolve(sys, noPower{}, vars, nil, cat)
	kenv := kit.Env{Now: clock.Now, Self: m.profile.Ref, Kind: m.profile.Kind}
	kenv.Calls = m.router(kenv)
	d := m.engine(kenv)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := d.Begin(ctx, func(string) {}); err != nil {
		t.Fatal(err)
	}
	run.mu.Lock()
	clear(run.spawned)
	run.mu.Unlock()
	files.mu.Lock()
	clear(files.opens)
	files.mu.Unlock()
	for tick := 1; tick <= ticks; tick++ {
		clock.advance(time.Second)
		d.Step(ctx, tick)
		run.settle()
	}
	run.settle()
	cancel()
	minutes := ticks / 60
	run.mu.Lock()
	defer run.mu.Unlock()
	for _, name := range slices.Sorted(maps.Keys(run.spawned)) {
		budget := idlePerMinute*minutes + idleStartup
		t.Logf("spawned %s %d times in %d minutes (budget %d)", name, run.spawned[name], minutes, budget)
		if run.spawned[name] > budget {
			t.Errorf("an idle engine spawned %s %d times in %d minutes; its budget is %d", name, run.spawned[name], minutes, budget)
		}
	}
	files.mu.Lock()
	defer files.mu.Unlock()
	for _, name := range slices.Sorted(maps.Keys(files.opens)) {
		budget := idlePerMinute*minutes + idleStartup
		t.Logf("opened %s %d times in %d minutes (budget %d)", name, files.opens[name], minutes, budget)
		if files.opens[name] > budget {
			t.Errorf("an idle engine opened %s %d times in %d minutes; its budget is %d", name, files.opens[name], minutes, budget)
		}
	}
}

func (idleSockets) DialUnix(context.Context, string, time.Duration) (local.Conn, error) {
	return nil, local.ErrNotFound
}
