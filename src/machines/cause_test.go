package machines

import (
	"slices"
	"testing"
)

func TestAServerDownIsToldWhyByItsConsoleThenByHowItsRunEnded(t *testing.T) {
	oom := "[ 4127.551] Out of memory: Killed process 812 (k3s-server) total-vm:2318764kB"
	kernelPanic := "[   92.004] Kernel panic - not syncing: Attempted to kill init! exitcode=0x00000009"
	cases := []struct {
		name string
		end  RunEnd
		want string
	}{
		{"an out-of-memory kill, whatever the end", RunEnd{Reason: Crashed, LastLines: []string{"etcd: slow fdatasync", oom}}, "out of memory"},
		{"an older kernel's kill", RunEnd{Reason: Unknown, LastLines: []string{"Out of memory: Kill process 812 (k3s-server) score 901"}}, "out of memory"},
		{"a kernel panic before a reboot", RunEnd{Reason: GuestReboot, LastLines: []string{kernelPanic, "CPU: 0 PID: 1 Comm: init"}}, "kernel panic"},
		{"the last cause told wins", RunEnd{Reason: Crashed, LastLines: []string{oom, kernelPanic}}, "kernel panic"},
		{"a poweroff its guest declared", RunEnd{Reason: GuestPoweroff, LastLines: []string{"Reached target Power-Off."}}, "powered off from inside"},
		{"a crash no line explains", RunEnd{Reason: Crashed, ExitCode: 1, LastLines: []string{"Error: the VM stopped"}}, "the hypervisor stopped it"},
		{"a reboot from inside", RunEnd{Reason: GuestReboot}, "rebooted from inside"},
		{"an end nobody could read", RunEnd{Reason: Unknown, ExitCode: -1}, "it stopped"},
	}
	for _, c := range cases {
		if got := DownCause(c.end); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
	if got := CheckCause([]string{"systemd[1]: Started k3s.service", oom}); got != "out of memory" {
		t.Errorf("a check failing after an out-of-memory kill: %q", got)
	}
	if got := CheckCause([]string{"systemd[1]: Started k3s.service"}); got != CheckFailed {
		t.Errorf("a check failing with a quiet console: %q, want %q", got, CheckFailed)
	}
}

func TestTheLinesKeptAreTheLastThreeWithTheCauseAmongThem(t *testing.T) {
	lines := []string{"a", "b", "Out of memory: Killed process 812 (k3s-server)", "c", "d", "e", "f"}
	cases := []struct {
		name  string
		lines []string
		want  []string
	}{
		{"none", nil, nil},
		{"fewer than three", []string{"a", "b"}, []string{"a", "b"}},
		{"no cause: the last three", []string{"a", "b", "c", "d"}, []string{"b", "c", "d"}},
		{"the cause among the last three", lines[:5], lines[2:5]},
		{"the cause before them: it and the two after it", lines, lines[2:5]},
	}
	for _, c := range cases {
		if got := LastLines(c.lines); !slices.Equal(got, c.want) {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

func TestABlankOrTimestampOnlyLineTellsNothing(t *testing.T) {
	for line, want := range map[string]bool{
		"":                          false,
		"   \t":                     false,
		"[ 4127.551234]":            false,
		"2026-11-07T14:02:11Z":      false,
		"14:02:11.512":              false,
		"[ 4127.551] oom_reaper":    true,
		"Reached target Power-Off.": true,
	} {
		if got := Meaningful(line); got != want {
			t.Errorf("%q tells something: %v, want %v", line, got, want)
		}
	}
}
