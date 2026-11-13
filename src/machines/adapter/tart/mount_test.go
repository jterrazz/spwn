package tart_test

import (
	"context"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/platform/local"
)

// value: the guest's mount commands, played over a folder of the test's
// own: a mount is a file under mounted/, and each check and mount is
// logged with the run that asked it, slowly, so two runs at once overlap.
const stubs = `mountpoint|n=$(printf %s "$2" | tr / _); echo "$PPID" >> "$T/asked"; sleep 0.05; [ -e "$T/mounted/$n" ]
mount|n=$(printf %s "$1" | tr / _); echo "$PPID" >> "$T/asked"; sleep 0.05; touch "$T/mounted/$n"
umount|n=$(printf %s "${2:-$1}" | tr / _); rm -f "$T/mounted/$n"
chown|true`

// value: a guest over a folder of the test's own: the mount commands
// stubbed, an fstab of many system lines, and the mount script with its
// fstab and its lock moved into the folder.
type mountGuest struct {
	sys    local.System
	dir    string
	script string
	kept   []string
}

func newMountGuest(t *testing.T) mountGuest {
	t.Helper()
	g := mountGuest{dir: t.TempDir()}
	g.sys = local.New([]string{"PATH=" + filepath.Join(g.dir, "bin") + ":/usr/bin:/bin", "T=" + g.dir, "HOME=" + g.dir}, nil, nil, nil)
	if _, err := g.sys.FS.Stat("/usr/bin/flock"); err != nil {
		t.Skip("the guest mount script runs on Linux, where flock is")
	}
	if err := g.sys.FS.MkdirAll(filepath.Join(g.dir, "mounted"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := g.sys.FS.MkdirAll(filepath.Join(g.dir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(stubs, "\n") {
		name, body, _ := strings.Cut(line, "|")
		if err := g.sys.FS.WriteFile(filepath.Join(g.dir, "bin", name), []byte("#!/bin/bash\n"+body+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for i := range 4000 {
		g.kept = append(g.kept, "UUID=disk-"+strconv.Itoa(i)+" /srv/d"+strconv.Itoa(i)+" ext4 defaults 0 2")
	}
	if err := g.sys.FS.WriteFile(filepath.Join(g.dir, "fstab"), []byte(strings.Join(g.kept, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, g.script = model.MountLine(nil)
	for _, at := range []string{"/etc/fstab", "/run/station-mounts.lock"} {
		if !strings.Contains(g.script, "="+at+"\n") {
			t.Fatalf("the script names no %s of its own", at)
		}
		g.script = strings.Replace(g.script, "="+at+"\n", "="+filepath.Join(g.dir, filepath.Base(at))+"\n", 1)
	}
	return g
}

// value: the script run as the guest runs it, its line the one MountLine
// gives, without its sudo.
func (g mountGuest) mount(t *testing.T, mounts ...model.Mount) {
	t.Helper()
	line, _ := model.MountLine(mounts)
	_, asRoot, ok := strings.Cut(line, " bash -s ")
	if !ok {
		t.Fatalf("the mount line %q runs no bash -s as root", line)
	}
	c := local.Command("bash", "-c", "bash -s "+asRoot)
	c.Stdin = g.script
	out, err := g.sys.Run.Output(context.Background(), c)
	if err != nil || out.ExitCode != 0 {
		t.Errorf("the mount script: %v, exit %d: %s", err, out.ExitCode, out.Stderr)
	}
}

// value: the fstab's system lines and station's, as it reads now.
func (g mountGuest) fstab(t *testing.T) (system, station []string) {
	t.Helper()
	data, err := g.sys.FS.ReadFile(filepath.Join(g.dir, "fstab"))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		if strings.HasPrefix(line, "station-") {
			station = append(station, line)
		} else if line != "" {
			system = append(system, line)
		}
	}
	return system, station
}

func TestTheGuestMountScriptsTakeTheFstabOneAtATime(t *testing.T) {
	// invariant: the engine's mount step and a first start's may run at
	// once in one guest; each holds the lock for its whole run, so their
	// checks and mounts never interleave, the system's lines are never
	// lost, and station's lines end as one run asked.
	g := newMountGuest(t)
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Go(func() { g.mount(t, model.Mount{Guest: filepath.Join(g.dir, "g"+strconv.Itoa(i))}) })
	}
	wg.Wait()
	system, station := g.fstab(t)
	if !slices.Equal(system, g.kept) {
		t.Errorf("the fstab kept %d of its %d system lines", len(system), len(g.kept))
	}
	if len(station) != 1 || !strings.HasPrefix(station[0], "station-1 "+filepath.Join(g.dir, "g")) {
		t.Errorf("station's lines %q; want the one line of one run", station)
	}
	asked, err := g.sys.FS.ReadFile(filepath.Join(g.dir, "asked"))
	if err != nil {
		t.Fatal(err)
	}
	var runs []string
	for _, pid := range strings.Fields(string(asked)) {
		if len(runs) > 0 && runs[len(runs)-1] == pid {
			continue
		}
		if slices.Contains(runs, pid) {
			t.Fatalf("two runs held the fstab at once: %q", strings.Fields(string(asked)))
		}
		runs = append(runs, pid)
	}
}

func TestTheGuestFstabIsNeverReadHalfWritten(t *testing.T) {
	// invariant: the system reads /etc/fstab without station's lock (mount
	// -a, systemd's generator), so a run replaces it by a rename: a read
	// at any moment holds every system line.
	g := newMountGuest(t)
	var done atomic.Bool
	var reads, torn int
	read := make(chan struct{})
	go func() {
		defer close(read)
		for !done.Load() {
			data, err := g.sys.FS.ReadFile(filepath.Join(g.dir, "fstab"))
			if err != nil || strings.Count(string(data), "UUID=disk-") != len(g.kept) {
				torn++
			}
			reads++
		}
	}()
	a, b := model.Mount{Guest: filepath.Join(g.dir, "a")}, model.Mount{Guest: filepath.Join(g.dir, "b")}
	for i := range 20 {
		if i%2 == 0 {
			g.mount(t, a)
		} else {
			g.mount(t, a, b)
		}
	}
	done.Store(true)
	<-read
	if torn > 0 {
		t.Errorf("%d of %d reads of the fstab lacked system lines", torn, reads)
	}
}

func TestTheGuestMountScriptBringsTheFstabToTheMountsAsked(t *testing.T) {
	// invariant: a run makes station's lines exactly the ones asked: one no
	// mount asks is dropped and unmounted, one asked is mounted, and the
	// system's lines stay as they were.
	g := newMountGuest(t)
	notes, data := filepath.Join(g.dir, "notes"), filepath.Join(g.dir, "data")
	g.mount(t, model.Mount{Guest: notes, ReadOnly: true}, model.Mount{Guest: data})
	if _, station := g.fstab(t); len(station) != 2 || !strings.Contains(station[0], " virtiofs ro,nofail ") {
		t.Fatalf("station's lines %q; want both, the first read-only", station)
	}
	g.mount(t, model.Mount{Guest: data})
	system, station := g.fstab(t)
	if want := []string{"station-1 " + data + " virtiofs rw,nofail 0 0"}; !slices.Equal(station, want) || !slices.Equal(system, g.kept) {
		t.Errorf("station's lines %q and %d system lines; want %q and %d", station, len(system), want, len(g.kept))
	}
	for at, mounted := range map[string]bool{notes: false, data: true} {
		_, err := g.sys.FS.Stat(filepath.Join(g.dir, "mounted", strings.ReplaceAll(at, "/", "_")))
		if (err == nil) != mounted {
			t.Errorf("%s mounted %v; want %v", at, err == nil, mounted)
		}
	}
}
