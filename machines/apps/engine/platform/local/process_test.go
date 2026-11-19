package local_test

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

func TestAProcessRunsWhileItsPidLivesWithTheStartItWasReadWith(t *testing.T) {
	var procs local.Processes
	pid, start, err := procs.Self()
	if err != nil || start == "" {
		t.Fatalf("Self = %d %q, %v; want this process and its start", pid, start, err)
	}
	if again, readErr := local.StartOf(pid); readErr != nil || again != start {
		t.Fatalf("this process's start read again = %q, %v; want %q, the same", again, readErr, start)
	}
	if !procs.Runs(pid, start) {
		t.Error("this process does not run, as it reads")
	}
	if procs.Runs(pid, start+"1") {
		t.Error("this pid with another start runs; want a pid taken again to be another process")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, "sleep", "30")
	if err = child.Start(); err != nil {
		t.Fatal(err)
	}
	childStart, err := local.StartOf(child.Process.Pid)
	if err != nil || !procs.Runs(child.Process.Pid, childStart) {
		t.Fatalf("a child that sleeps: start %q, %v; want it running", childStart, err)
	}
	if err := child.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for procs.Runs(child.Process.Pid, childStart) {
		select {
		case <-tick.C:
		case <-ctx.Done():
			t.Fatal("a child killed, not yet reaped, still runs after 20 s; want a zombie read as ended")
		}
	}
	_ = child.Wait()
	if procs.Runs(child.Process.Pid, childStart) {
		t.Error("a child killed and reaped still runs")
	}
}
