package darwin

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const stubEvery = 100 * time.Millisecond

// value: the test seam STATION_POWER_STUB names: the reading is the files of
// its directory, and every call that would change the Mac is a line of
// $HOME/.calls instead; no spec ever takes a real assertion.
type stubPower struct {
	dir, calls string
	mu         sync.Mutex
}

func (s *stubPower) word(name, fallback string) string {
	data, err := os.ReadFile(filepath.Join(s.dir, name))
	if err != nil {
		return fallback
	}
	return strings.TrimSpace(string(data))
}

func (s *stubPower) record(line string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := os.OpenFile(s.calls, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	_, _ = f.WriteString(line + "\n")
	_ = f.Close()
}

// invariant: a stub directory that does not exist is a Mac with no power
// management, so a spec that never asked for Power records nothing.
func (s *stubPower) present() bool {
	info, err := os.Stat(s.dir)
	return err == nil && info.IsDir()
}

func (s *stubPower) HoldIdle(name string) (func(), error) {
	if !s.present() {
		return nil, ErrPowerUnsupported
	}
	s.record("power assert idle " + name)
	var once sync.Once
	return func() { once.Do(func() { s.record("power release idle") }) }, nil
}

func (s *stubPower) SetClamshellSleep(disabled bool) error {
	if !s.present() {
		return ErrPowerUnsupported
	}
	if disabled {
		s.record("power clamshell disabled")
	} else {
		s.record("power clamshell normal")
	}
	return nil
}

func (s *stubPower) Read() (Reading, error) {
	if !s.present() {
		return Reading{}, ErrPowerUnsupported
	}
	r := Reading{Lid: s.word("lid", ""), Source: s.word("source", "ac"), LowPower: s.word("lowpower", "0") == "1"}
	if n, err := strconv.Atoi(s.word("battery", "")); err == nil {
		r.HasBattery, r.Percent = true, n
	}
	r.Charging = s.word("charging", "0") == "1"
	if r.HasBattery {
		r.Health, _ = strconv.Atoi(s.word("health", "0"))
		r.Cycles, _ = strconv.Atoi(s.word("cycles", "0"))
	}
	r.Thermal, _ = strconv.Atoi(s.word("thermal", "0"))
	return r, nil
}

func (s *stubPower) Sleeping() bool { return s.present() && s.word("sleeping", "0") == "1" }

func (s *stubPower) Ticks(ctx context.Context) <-chan struct{} { return ticks(ctx, stubEvery, nil) }
