package wire

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	machinesfake "github.com/jterrazz/studio-station/apps/engine/machines/adapter/fake"
	machinesjsonfile "github.com/jterrazz/studio-station/apps/engine/machines/adapter/jsonfile"
	"github.com/jterrazz/studio-station/apps/engine/machines/adapter/kernel"
	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	machinescontract "github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	setupcontract "github.com/jterrazz/studio-station/apps/engine/setup/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

// value: set in a child of TestTwoCreatesInTwoProcessesNeverBothTakeTheMemoryLeft:
// the folder both share, this create's name and the other's.
var createChild = flag.String("create-child", "", "folder,name,other: run one create as a child process")

type notedPrefs map[string][]byte

func (p notedPrefs) Read(group string, v any) error {
	if data, ok := p[group]; ok {
		return json.Unmarshal(data, v)
	}
	return nil
}

func (p notedPrefs) Write(group string, v any) error {
	data, err := json.Marshal(v)
	p[group] = data
	return err
}

type unbuilt struct{}

func (unbuilt) Latest(_ context.Context, image string) (ref, version string, err error) {
	return "", "", &model.NoImageError{Image: image}
}
func (unbuilt) KeyPath() string { return "/nowhere/station" }

type answering struct{}

func (answering) Ping(context.Context) (string, error) { return "1.0.0", nil }
func (answering) VMs(context.Context) (map[string]machinescalls.EngineVM, error) {
	return map[string]machinescalls.EngineVM{}, nil
}
func (answering) Do(context.Context, string, string) error { return errors.New("runs no VM") }
func (answering) EnvPorts(context.Context, string) ([]model.Reserved, error) {
	return nil, nil
}
func (answering) GuestExec(context.Context, string, string, string) (machinescalls.GuestRan, error) {
	return machinescalls.GuestRan{}, errors.New("runs no VM")
}

// value: the records' disk, whose every listing by one process waits, up
// to a second, until the other process has listed them as many times, and
// whose every record write waits, up to a second, until the other has read
// that listing: two creates read the records in lockstep, so neither writes
// its record before the other has judged from them, unless a lock keeps
// the other out.
type lockstep struct {
	lockable
	sys        local.System
	meet, me   string
	other      string
	listings   int
	recordsDir string
}

func (l *lockstep) await(mark string, n int) {
	wait, stop := context.WithTimeout(context.Background(), time.Second)
	defer stop()
	for wait.Err() == nil {
		data, _ := l.sys.FS.ReadFile(filepath.Join(l.meet, mark+"."+l.other))
		if got, _ := strconv.Atoi(string(data)); got >= n {
			return
		}
		_ = l.sys.Clock.Sleep(wait, 10*time.Millisecond)
	}
}

func (l *lockstep) ReadDir(path string) ([]fs.DirEntry, error) {
	if path != l.recordsDir {
		return l.lockable.ReadDir(path)
	}
	l.listings++
	_ = l.sys.FS.WriteFile(filepath.Join(l.meet, "listed."+l.me), []byte(strconv.Itoa(l.listings)), 0o600)
	l.await("listed", l.listings)
	entries, err := l.lockable.ReadDir(path)
	_ = l.sys.FS.WriteFile(filepath.Join(l.meet, "read."+l.me), []byte(strconv.Itoa(l.listings)), 0o600)
	return entries, err
}

func (l *lockstep) WriteFile(path string, data []byte, perm fs.FileMode) error {
	if filepath.Dir(path) == l.recordsDir {
		l.await("read", l.listings)
	}
	return l.lockable.WriteFile(path, data, perm)
}

func TestCreateChildProcess(t *testing.T) {
	if *createChild == "" {
		t.Skip("run by TestTwoCreatesInTwoProcessesNeverBothTakeTheMemoryLeft")
	}
	parts := strings.Split(*createChild, ",")
	meet, me, other := parts[0], parts[1], parts[2]
	sys := local.New(nil, nil, nil, nil)
	outcome := func(s string) {
		if err := sys.FS.WriteFile(filepath.Join(meet, "outcome."+me), []byte(s), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	w := &world.World{ThisMac: "mac"}
	w.Mac("mac").Images = []string{"station-linux"}
	prefs := notedPrefs{}
	p := machinescalls.Prefs{Name: setupcontract.Name(prefs), Macs: machinescontract.Macs(prefs), VMUser: machinescontract.VMUser(prefs)}
	if err := p.Name.Set("mac"); err != nil {
		t.Fatal(err)
	}
	records := filepath.Join(meet, "vms")
	store := machinesjsonfile.New(&lockstep{lockable: lockable{sys.FS}, sys: sys, meet: meet, me: me, other: other, recordsDir: records}, records)
	linux := catalog.Image{Name: "station-linux", Kind: catalog.Linux, Defaults: catalog.ImageDefaults{CPUs: 4, MemoryBytes: 4 << 30, DiskBytes: 20e9}}
	m := machinescalls.New(macSystem{sys}, machinesfake.HypervisorOf(w, "mac", "hypervisor", nil), p, store, nil, unbuilt{}, answering{}, nil, machinescalls.Options{
		ThisMachine: catalog.Mac, Images: []catalog.Image{linux}, KnownHosts: filepath.Join(meet, me+".known_hosts"), Logs: filepath.Join(meet, me+".logs"),
		Hardware:  func(context.Context) coremachine.Hardware { return coremachine.Hardware{Memory: 16 << 30, CPUs: 8} },
		Processes: kernel.Processes{},
	})
	built := false
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	_, err := m.Create(ctx, model.CreateRequest{Image: "station-linux", Name: me, Size: model.Resources{MemoryBytes: 8 << 30}}, func(ctx context.Context, _ func(string)) error {
		built = true
		_ = sys.FS.WriteFile(filepath.Join(meet, "built."+me), nil, 0o600)
		for ctx.Err() == nil {
			if _, err := sys.FS.Stat(filepath.Join(meet, "release")); err == nil {
				break
			}
			_ = sys.Clock.Sleep(ctx, 20*time.Millisecond)
		}
		return errors.New("the test builds no image")
	}, nil)
	switch {
	case built:
		outcome("built")
	case errors.As(err, new(*model.TooBigError)):
		outcome("refused")
	default:
		outcome("error: " + err.Error())
	}
}

// invariant: two creates of 8 GB on a Mac with room for one, run by two
// processes at once, their reads of the records in lockstep, end with one
// built and one refused for memory: the records' lock is every process's,
// and the second judges from the first's record, whose process runs.
func TestTwoCreatesInTwoProcessesNeverBothTakeTheMemoryLeft(t *testing.T) {
	sys := local.New(nil, nil, nil, nil)
	meet := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	ran := make(chan error, 2)
	for _, pair := range [][2]string{{"api", "web"}, {"web", "api"}} {
		go func() {
			out, err := sys.Run.Output(ctx, local.Cmd{Name: sys.Env.Executable(), Args: []string{"-test.run=^TestCreateChildProcess$", "-test.count=1", "-create-child=" + meet + "," + pair[0] + "," + pair[1]}})
			if err != nil {
				err = errors.New(err.Error() + ": " + out.Stdout + out.Stderr)
			}
			ran <- err
		}()
	}
	outcomes := map[string]string{}
	for ctx.Err() == nil && len(outcomes) < 2 {
		for _, name := range []string{"api", "web"} {
			if data, err := sys.FS.ReadFile(filepath.Join(meet, "outcome."+name)); err == nil {
				outcomes[name] = string(data)
			}
		}
		_, apiBuilds := sys.FS.Stat(filepath.Join(meet, "built.api"))
		_, webBuilds := sys.FS.Stat(filepath.Join(meet, "built.web"))
		if len(outcomes) == 1 || apiBuilds == nil && webBuilds == nil {
			_ = sys.FS.WriteFile(filepath.Join(meet, "release"), nil, 0o600)
		}
		_ = sys.Clock.Sleep(ctx, 20*time.Millisecond)
	}
	for range 2 {
		select {
		case err := <-ran:
			if err != nil {
				t.Fatalf("a create's process: %v", err)
			}
		case <-ctx.Done():
			t.Fatal("a create's process did not end in 2 minutes")
		}
	}
	got := []string{outcomes["api"], outcomes["web"]}
	if (got[0] != "built" || got[1] != "refused") && (got[0] != "refused" || got[1] != "built") {
		t.Errorf("two creates of 8 GB in two processes on a Mac with room for one ended %q; want one built, one refused for memory", got)
	}
}
