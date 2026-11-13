//go:build darwin && cgo

package darwin

import (
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestAProcessReadByItsPidSaysWhenItStarted(t *testing.T) {
	before := time.Now().Add(-time.Second)
	child := exec.Command("/bin/sleep", "30")
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = child.Process.Kill(); _ = child.Wait() })
	got, err := Kernel{}.Process(child.Process.Pid)
	if err != nil || len(got.List) != 1 {
		t.Fatalf("Process(child) = %+v, %v; want the child's process", got, err)
	}
	if started := got.List[0].Started; started.Before(before) || started.After(time.Now()) {
		t.Errorf("Started = %v; want the child's start, after %v", started, before)
	}
}

func TestThisMacsNetworkTotalsAreTheOnesNetstatReads(t *testing.T) {
	c, err := Kernel{}.Counters()
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range c.Links {
		if !strings.HasPrefix(l.Name, "en") {
			continue
		}
		out, err := exec.Command("/usr/sbin/netstat", "-ibn", "-I", l.Name).Output()
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		if len(lines) < 2 {
			continue
		}
		f := strings.Fields(lines[1])
		in, _ := strconv.ParseUint(f[len(f)-5], 10, 64)
		sent, _ := strconv.ParseUint(f[len(f)-2], 10, 64)
		const drift = 1 << 20
		if in < l.In || in-l.In > drift || sent < l.Out || sent-l.Out > drift {
			t.Errorf("%s: station reads in %d out %d; netstat, read after, %d and %d", l.Name, l.In, l.Out, in, sent)
		}
	}
}

func TestAProcessNamesTheFilesItHoldsOpen(t *testing.T) {
	dir := t.TempDir()
	child := exec.Command("/bin/sh", "-c", `: >"$0"; exec 3<"$0"; exec /bin/sleep 30`, dir+"/disk.img")
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = child.Process.Kill(); _ = child.Wait() })
	held := "/" + filepath.Base(dir) + "/disk.img"
	var files []string
	for range 100 {
		files, _ = Kernel{}.Files(child.Process.Pid)
		if slices.ContainsFunc(files, func(f string) bool { return strings.HasSuffix(f, held) }) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Errorf("the child holds %s open; its files read %v", held, files)
}
