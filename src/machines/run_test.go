package machines

import (
	"testing"
	"time"
)

func TestAfterRunKeepsEveryVMAStopDidNotDeclare(t *testing.T) {
	start := time.Date(2026, 11, 7, 10, 0, 0, 0, time.UTC)
	at := func(s int) time.Time { return start.Add(time.Duration(s) * time.Second) }
	poweroff := RunEnd{Reason: GuestPoweroff}
	reboot := RunEnd{Reason: GuestReboot}
	for _, c := range []struct {
		name     string
		lifetime Lifetime
		end      RunEnd
		reruns   []time.Time
		now      time.Time
		want     Action
	}{
		{name: "the declared poweroff wins over the run's exit code", lifetime: Ephemeral, end: RunEnd{Reason: GuestPoweroff, ExitCode: 1}, now: at(0), want: ActDelete},
		{name: "a persistent workspace's declared poweroff keeps it stopped", lifetime: Persistent, end: poweroff, now: at(0), want: ActStopped},
		{name: "an ephemeral crash keeps it stale", lifetime: Ephemeral, end: RunEnd{Reason: Crashed, ExitCode: 137}, now: at(0), want: ActStale},
		{name: "an ephemeral run whose end nobody can read keeps it stale", lifetime: Ephemeral, end: RunEnd{Reason: Unknown, ExitCode: -1}, now: at(0), want: ActStale},
		{name: "a persistent crash keeps it stale: a workspace never restarts by itself", lifetime: Persistent, end: RunEnd{Reason: Crashed, ExitCode: 137}, now: at(0), want: ActStale},
		{name: "a second reboot within the window still runs", lifetime: Ephemeral, end: reboot, reruns: []time.Time{at(0)}, now: at(10), want: ActRerun},
		{name: "a third reboot within the window is a loop", lifetime: Ephemeral, end: reboot, reruns: []time.Time{at(0), at(10)}, now: at(20), want: ActBroken},
		{name: "a persistent reboot loop is stopped too", lifetime: Persistent, end: reboot, reruns: []time.Time{at(0), at(10)}, now: at(20), want: ActBroken},
		{name: "reboots older than the window are forgotten", lifetime: Ephemeral, end: reboot, reruns: []time.Time{at(0), at(10)}, now: at(65), want: ActRerun},
		{name: "the window is a strict 60 seconds", lifetime: Ephemeral, end: reboot, reruns: []time.Time{at(0), at(30)}, now: at(60), want: ActRerun},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got, _ := AfterRun(c.lifetime, c.end, c.reruns, c.now); got != c.want {
				t.Errorf("AfterRun = %d, want %d", got, c.want)
			}
		})
	}
}

func TestAServerComesBackUnlessSomethingInsidePoweredItOff(t *testing.T) {
	now := time.Date(2026, 11, 7, 10, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		end  RunEnd
		want Action
	}{
		{RunEnd{Reason: Crashed, ExitCode: 137}, ActRestart},
		{RunEnd{Reason: Unknown, ExitCode: -1}, ActRestart},
		{RunEnd{Reason: GuestPoweroff}, ActPoweredOff},
		{RunEnd{Reason: GuestReboot}, ActRerun},
	} {
		if got, _ := AfterRun(AlwaysOn, c.end, nil, now); got != c.want {
			t.Errorf("a server's %s end: %d, want %d", c.end.Reason, got, c.want)
		}
	}
	if got, _ := AfterRun(AlwaysOn, RunEnd{Reason: GuestReboot}, []time.Time{now, now.Add(10 * time.Second)}, now.Add(20*time.Second)); got != ActBroken {
		t.Errorf("a server's third reboot within 60 s: %d, want a loop", got)
	}
}

func TestAnEngineThatStartsAdoptsRunsAndBootsOnlyServers(t *testing.T) {
	for _, c := range []struct {
		lifetime   Lifetime
		want, live bool
		got        Convergence
	}{
		{AlwaysOn, true, true, Adopt},
		{Ephemeral, true, true, Adopt},
		{AlwaysOn, true, false, Boot},
		{AlwaysOn, false, false, Stay},
		{Ephemeral, true, false, EndedWithItsMac},
		{Persistent, true, false, EndedWithItsMac},
		{Persistent, false, false, Stay},
	} {
		if got := Converge(c.lifetime, c.want, c.live); got != c.got {
			t.Errorf("Converge(%s, wanted running %v, live %v) = %d, want %d", c.lifetime, c.want, c.live, got, c.got)
		}
	}
}
