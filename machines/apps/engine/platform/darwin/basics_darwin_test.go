//go:build darwin && !cgo

package darwin

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestCountersWithoutCgoReadTheCountAndTheMemory(t *testing.T) {
	read := func(key string) uint64 {
		out, err := exec.Command("sysctl", "-n", key).Output()
		if err != nil {
			t.Fatal(err)
		}
		n, err := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	c, err := Kernel{}.Counters()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := c.Cores(), int(read("hw.ncpu")); got != want {
		t.Errorf("cores = %d, hw.ncpu = %d (NumCPU %d)", got, want, runtime.NumCPU())
	}
	if want := read("hw.memsize"); c.Memsize != want {
		t.Errorf("memsize = %d, hw.memsize = %d", c.Memsize, want)
	}
}
