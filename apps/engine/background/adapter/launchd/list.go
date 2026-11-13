package launchd

import (
	"context"
	"errors"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/background"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

type folder struct {
	dir    string
	scope  string
	system bool
}

func (j *Jobs) folders() []folder {
	return []folder{
		{dir: path.Join(j.o.Home, "Library", "LaunchAgents"), scope: background.ScopeUser},
		{dir: path.Join("/", j.o.Root, "Library", "LaunchAgents"), scope: background.ScopeRoot},
		{dir: path.Join("/", j.o.Root, "Library", "LaunchDaemons"), scope: background.ScopeRoot, system: true},
	}
}

func systemJob(label string) bool {
	for _, p := range catalog.LaunchSystemPrefixes {
		if strings.HasPrefix(label, p) {
			return true
		}
	}
	return false
}

type listed struct {
	pid  string
	exit string
}

// outside: `launchctl list` prints `<pid or -><TAB><last exit><TAB><label>` per loaded job.
func parseList(text string) map[string]listed {
	out := map[string]listed{}
	for _, line := range strings.Split(text, "\n") {
		f := strings.Fields(line)
		if len(f) < 3 || f[0] == "PID" {
			continue
		}
		out[f[2]] = listed{pid: f[0], exit: f[1]}
	}
	return out
}

type reply struct {
	out   local.Output
	found bool
	late  bool
}

func (j *Jobs) ask(ctx context.Context, args ...string) reply {
	if j.o.Budget > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, j.o.Budget)
		defer cancel()
	}
	out, err := j.launchctl(ctx, args...)
	switch {
	case errors.Is(err, local.ErrTimeout), errors.Is(ctx.Err(), context.DeadlineExceeded):
		return reply{late: true}
	case errors.Is(err, local.ErrNotFound):
		return reply{}
	case err != nil:
		return reply{found: true, out: local.Output{ExitCode: -1}}
	}
	return reply{found: true, out: out}
}

func (j *Jobs) List(ctx context.Context) ([]background.Job, error) {
	// invariant: a read runs no verb but `list` and `print`; removing a job is
	// the owner's, through the command its Removal names.
	jobs := []background.Job{}
	var running map[string]listed
	if r := j.ask(ctx, "list"); r.found && !r.late && r.out.ExitCode == 0 {
		running = parseList(r.out.Stdout)
	}
	seen := map[string]bool{}
	for _, d := range j.folders() {
		entries, err := j.files.ReadDir(d.dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			label, ok := strings.CutSuffix(e.Name(), ".plist")
			if !ok || systemJob(label) || seen[label] {
				continue
			}
			seen[label] = true
			file := path.Join(d.dir, e.Name())
			job := background.Job{Label: label, Scope: d.scope, File: file}
			switch {
			case d.system:
				j.printed(ctx, &job)
			case running == nil:
				job.Unread = true
			default:
				l, found := running[label]
				listedAs(&job, l, found)
			}
			job.Program = j.programOf(file)
			j.livesOn(&job)
			jobs = append(jobs, job)
		}
	}
	for label, l := range running {
		if seen[label] || systemJob(label) || l.pid == "-" {
			continue
		}
		job := background.Job{Label: label, Scope: background.ScopeUser}
		listedAs(&job, l, true)
		j.livesOn(&job)
		jobs = append(jobs, job)
	}
	slices.SortFunc(jobs, func(a, b background.Job) int { return strings.Compare(a.Label, b.Label) })
	return jobs, nil
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func (l listed) failing() bool { return l.exit != "0" && l.exit != "-" }

func listedAs(job *background.Job, l listed, found bool) {
	job.Loaded = found
	switch {
	case !found:
	case l.pid != "-":
		job.Running, job.PID = true, atoi(l.pid)
	case l.failing():
		job.LastExit = atoi(l.exit)
	}
}

// outside: `launchctl print system/<label>` answers without root; its
// top-level keys are one tab deep (`\tstate = running`, `\tpid = 529`,
// `\tlast exit code = 1` or `(never exited)`); an unloaded label exits non-zero.
func (j *Jobs) printed(ctx context.Context, job *background.Job) {
	r := j.ask(ctx, "print", "system/"+job.Label)
	if r.late || !r.found {
		job.Unread = true
		return
	}
	if r.out.ExitCode != 0 {
		return
	}
	job.Loaded = true
	var state, pid, exit string
	for _, line := range strings.Split(r.out.Stdout, "\n") {
		if !strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "\t\t") {
			continue
		}
		k, v, ok := strings.Cut(strings.TrimSpace(line), " = ")
		if !ok {
			continue
		}
		switch k {
		case "state":
			state = v
		case "pid":
			pid = v
		case "last exit code":
			exit = v
		}
	}
	switch {
	case state == "running" && pid != "":
		job.Running, job.PID = true, atoi(pid)
	case exit != "" && exit != "0" && !strings.HasPrefix(exit, "("):
		job.LastExit = atoi(exit)
	case state == "":
		job.Unread = true
	}
}

func (j *Jobs) programOf(file string) string {
	body, err := j.files.ReadFile(file)
	if err != nil {
		return ""
	}
	return programOf(body)
}

func (j *Jobs) exists(p string) bool {
	_, err := j.files.Stat(p)
	return err == nil
}

func (j *Jobs) livesOn(job *background.Job) {
	job.OwnerExists = true
	if owner, known := catalog.LaunchOwnerOf(job.Label); known && owner.Bundle != "" {
		job.OwnerExists = j.exists(path.Join("/", j.o.Root, owner.Bundle))
	}
	job.ProgramExists = job.Program == "" || !strings.HasPrefix(job.Program, "/") || j.exists(j.rooted(job.Program))
	job.Removal = j.removal(*job)
}

// value: a program path under the user's home stays where it is; the rest of
// the machine reads under the system root, as /Applications does.
func (j *Jobs) rooted(p string) string {
	if strings.HasPrefix(p, j.o.Home+"/") {
		return p
	}
	return path.Join("/", j.o.Root, p)
}

// value: the path the owner types: the system root dropped, the home as `~`.
func (j *Jobs) unrooted(p string) string {
	if root := path.Join("/", j.o.Root); root != "/" {
		if rest, ok := strings.CutPrefix(p, root+"/"); ok {
			return "/" + rest
		}
	}
	if rest, ok := strings.CutPrefix(p, j.o.Home+"/"); ok {
		return "~/" + rest
	}
	return p
}

func (j *Jobs) removal(job background.Job) string {
	gui := "gui/" + strconv.Itoa(j.o.UID) + "/" + job.Label
	shown := j.unrooted(job.File)
	switch {
	case job.File == "":
		return ""
	case strings.HasPrefix(shown, "/Library/LaunchDaemons/"):
		return catalog.AsRoot(command+" bootout system/"+job.Label) + " && " + catalog.AsRoot("rm "+shown)
	case strings.HasPrefix(shown, "/Library/LaunchAgents/"):
		return command + " bootout " + gui + " && " + catalog.AsRoot("rm "+shown)
	}
	return command + " bootout " + gui + " && rm " + shown
}
