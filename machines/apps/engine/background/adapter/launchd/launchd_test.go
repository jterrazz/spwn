package launchd_test

import (
	"context"
	"io/fs"
	"maps"
	"path"
	"regexp"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jterrazz/studio-station/apps/engine/background/adapter/launchd"
	"github.com/jterrazz/studio-station/apps/engine/background/contract"
	"github.com/jterrazz/studio-station/apps/engine/background/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/platform/local/localtest"
)

type files struct{ m fstest.MapFS }

func rel(p string) string { return strings.TrimPrefix(path.Clean(p), "/") }

func (f files) ReadFile(p string) ([]byte, error) { return fs.ReadFile(f.m, rel(p)) }

func (f files) WriteFile(p string, data []byte, _ fs.FileMode) error {
	f.m[rel(p)] = &fstest.MapFile{Data: data}
	return nil
}

func (files) MkdirAll(string, fs.FileMode) error { return nil }

func (f files) Remove(p string) error {
	if _, ok := f.m[rel(p)]; !ok {
		return fs.ErrNotExist
	}
	delete(f.m, rel(p))
	return nil
}

func (f files) Stat(p string) (fs.FileInfo, error)      { return fs.Stat(f.m, rel(p)) }
func (f files) ReadDir(p string) ([]fs.DirEntry, error) { return fs.ReadDir(f.m, rel(p)) }

var labelKey = regexp.MustCompile(`<key>Label</key>\s*<string>([^<]+)</string>`)

// value: launchd as its command line answers, over the definitions on disk:
// a domain's loaded labels and the running ones.
type launchctl struct {
	files   files
	loaded  map[string]bool
	running map[string]bool
	// value: every command line but the reads, in order.
	calls []string
}

func (l *launchctl) LookPath(name string) (string, bool) { return "/bin/" + name, name == "launchctl" }

func (l *launchctl) Output(_ context.Context, c local.Cmd) (local.Output, error) {
	a := c.Args
	if a[0] != "print" && a[0] != "list" && a[0] != "print-disabled" {
		l.calls = append(l.calls, strings.Join(a, " "))
	}
	label := func(target string) string { return target[strings.LastIndex(target, "/")+1:] }
	switch {
	case len(a) == 1 && a[0] == "list":
		var out strings.Builder
		out.WriteString("PID\tStatus\tLabel\n")
		for _, name := range slices.Sorted(maps.Keys(l.loaded)) {
			pid := "-"
			if l.running[name] {
				pid = "4242"
			}
			out.WriteString(pid + "\t0\t" + name + "\n")
		}
		return local.Output{Stdout: out.String()}, nil
	case len(a) == 2 && a[0] == "print":
		name := label(a[1])
		if !l.loaded[name] {
			return local.Output{ExitCode: 113, Stderr: "Could not find service"}, nil
		}
		state := "waiting"
		if l.running[name] {
			state = "running"
		}
		return local.Output{Stdout: name + " = {\n\tstate = " + state + "\n}\n"}, nil
	case len(a) == 3 && a[0] == "bootstrap":
		body, _ := l.files.ReadFile(a[2])
		m := labelKey.FindSubmatch(body)
		if m == nil {
			return local.Output{ExitCode: 5, Stderr: "Bootstrap failed: 5: Input/output error"}, nil
		}
		l.loaded[string(m[1])] = true
		return local.Output{}, nil
	case a[0] == "kickstart":
		name := label(a[len(a)-1])
		if !l.loaded[name] {
			return local.Output{ExitCode: 113, Stderr: "Could not find service"}, nil
		}
		l.running[name] = true
		return local.Output{}, nil
	case len(a) == 2 && a[0] == "bootout":
		name := label(a[1])
		if !l.loaded[name] {
			return local.Output{ExitCode: 5, Stderr: "Boot-out failed: 5: Input/output error"}, nil
		}
		delete(l.loaded, name)
		delete(l.running, name)
		return local.Output{}, nil
	}
	return local.Output{ExitCode: 64, Stderr: "unexpected: " + strings.Join(a, " ")}, nil
}

func TestJobs(t *testing.T) {
	contracttest.Jobs(t, func(*testing.T) contracttest.Subject {
		const home, uid = "/Users/owner", 501
		f := files{m: fstest.MapFS{}}
		run := &launchctl{files: f, loaded: map[string]bool{}, running: map[string]bool{}}
		jobs := launchd.New(run, f, &localtest.Clock{}, launchd.Options{Home: home, Root: "/", UID: uid})
		return contracttest.Subject{Jobs: jobs, UID: uid, Dir: home + "/Library/LaunchAgents"}
	})
}

func TestARestartLoadsTheJobAgainSoLaunchdReadsItsDefinitionAndItsBinaryAfresh(t *testing.T) {
	f := files{m: fstest.MapFS{}}
	run := &launchctl{files: f, loaded: map[string]bool{}, running: map[string]bool{}}
	jobs := launchd.New(run, f, &localtest.Clock{}, launchd.Options{Home: "/Users/owner", UID: 501})
	spec := contract.JobSpec{Label: "com.example.agent", Program: []string{"/usr/local/bin/agent"}, Log: "/Users/owner/agent.log", File: "/Users/owner/Library/LaunchAgents/com.example.agent.plist"}
	ref := contract.Ref{Domain: contract.Domain{UID: 501}, Label: spec.Label, File: spec.File}
	ctx := context.Background()
	if _, err := jobs.Install(ctx, spec); err != nil {
		t.Fatal(err)
	}
	for _, step := range []func(context.Context, contract.Ref) error{jobs.Load, jobs.Start} {
		if err := step(ctx, ref); err != nil {
			t.Fatal(err)
		}
	}
	run.calls = nil
	if err := jobs.Restart(ctx, ref); err != nil {
		t.Fatalf("Restart: %v", err)
	}
	want := "bootout gui/501/com.example.agent | bootstrap gui/501 " + spec.File + " | kickstart gui/501/com.example.agent"
	if got := strings.Join(run.calls, " | "); got != want {
		t.Errorf("a restart ran\n%s\nwant\n%s", got, want)
	}
	if !run.running[spec.Label] {
		t.Error("the restarted job is not running")
	}
}
