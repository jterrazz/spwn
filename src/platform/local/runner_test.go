package local

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestReapKillsOnlyTheProcessThatWasRecorded(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command("/bin/sleep", "30")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); _ = cmd.Wait() })
	file := filepath.Join(dir, strconv.Itoa(cmd.Process.Pid))
	r := &runner{}

	recycled := time.Now().Add(-time.Hour).Unix()
	if err := os.WriteFile(file, []byte(strconv.FormatInt(recycled, 10)), 0o600); err != nil {
		t.Fatal(err)
	}
	if n := r.Reap(dir); n != 0 || syscall.Kill(cmd.Process.Pid, 0) != nil {
		t.Fatalf("a pid whose start does not match was killed (%d)", n)
	}

	if err := os.WriteFile(file, []byte(strconv.FormatInt(time.Now().Unix(), 10)), 0o600); err != nil {
		t.Fatal(err)
	}
	if n := r.Reap(dir); n != 1 {
		t.Fatalf("the recorded process was not killed (%d)", n)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatal("the record outlives its reap")
	}
}

func TestACommandSilentPastItsAnswerBoundIsStoppedWithWhatItStarted(t *testing.T) {
	var said bytes.Buffer
	r := &runner{dirs: []string{"/bin"}, stdout: &said, stderr: &said}
	start := time.Now()

	_, err := r.Attach(context.Background(), Cmd{Name: "sh", Args: []string{"-c", "sleep 30 & sleep 30"}, Answer: 200 * time.Millisecond}, ToStderr)

	if !errors.Is(err, ErrSilent) || time.Since(start) > 4*time.Second {
		t.Fatalf("a silent command ran %s and ended as %v", time.Since(start), err)
	}
}

func TestACommandThatAnswersInTimeMayThenTakeItsTime(t *testing.T) {
	var said bytes.Buffer
	r := &runner{dirs: []string{"/bin"}, stdout: &said, stderr: &said}

	ran, err := r.Attach(context.Background(), Cmd{Name: "sh", Args: []string{"-c", "echo Downloading; sleep 1"}, Answer: 300 * time.Millisecond}, ToStderr)

	if err != nil || ran.ExitCode != 0 || !strings.Contains(said.String(), "Downloading") {
		t.Fatalf("an answering command was stopped: %v %+v %q", err, ran, said.String())
	}
}

func TestAChildThatCtrlCEndedIsAnInterruptionBeforeAnyContextIsCancelled(t *testing.T) {
	var said bytes.Buffer
	r := &runner{dirs: []string{"/bin"}, stdout: &said, stderr: &said}

	for name, script := range map[string]string{"signaled": "kill -INT $$; sleep 5", "said 130": "exit 130"} {
		ran, err := r.Attach(context.Background(), Cmd{Name: "sh", Args: []string{"-c", script}}, ToStderr)
		if !errors.Is(err, context.Canceled) || ran.ExitCode != -1 {
			t.Fatalf("%s: a Ctrl-C'd child ended as %v %+v", name, err, ran)
		}
	}
	ran, err := r.Attach(context.Background(), Cmd{Name: "sh", Args: []string{"-c", "exit 3"}}, ToStderr)
	if err != nil || ran.ExitCode != 3 {
		t.Fatalf("an ordinary failure must stay one: %v %+v", err, ran)
	}
}

func TestAChildGivenAnEnvironmentHasItAloneAndIsFoundOnItsPath(t *testing.T) {
	// invariant: a command run within an environment station built is the
	// one that environment's PATH finds, and it sees nothing of the
	// environment station was started with.
	started, given := t.TempDir(), t.TempDir()
	for _, dir := range []string{started, given} {
		script := "#!/bin/sh\necho " + filepath.Base(dir) + " \"$KEPT\" \"$DROPPED\"\n"
		tool := filepath.Join(dir, "tool")
		if err := os.WriteFile(tool, []byte(script), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(tool, 0o700); err != nil { //nolint:gosec // value: a program the lookup finds must be executable.
			t.Fatal(err)
		}
	}
	run := NewSearching([]string{"PATH=" + started, "DROPPED=x", "STATION_SYSTEM_DIRS="}, nil, nil, nil, Search{}).Run
	within := run.Within([]string{"PATH=" + given, "KEPT=k"})

	if found, _ := within.LookPath("tool"); found != filepath.Join(given, "tool") {
		t.Errorf("tool is looked up at %q; want the given PATH's", found)
	}
	out, err := within.Output(context.Background(), Command("tool"))
	if want := filepath.Base(given) + " k"; err != nil || strings.TrimSpace(out.Stdout) != want {
		t.Errorf("the child printed %q (%v); want %q", out.Stdout, err, want)
	}
}
