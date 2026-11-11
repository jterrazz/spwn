package wire

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"sync"

	"github.com/jterrazz/studio-station/src/engine"
	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/platform/local"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
	"github.com/jterrazz/studio-station/src/surface/tui"
	"github.com/jterrazz/studio-station/src/ui/token"
)

// invariant: SPEC §8.2: the TUI opens on the machine this terminal uses,
// and moving it elsewhere changes nothing of that choice: its own reads and
// calls name their machine, never the terminal's.
func tuiOpener(env kit.Env, m machine, noColor bool) func(context.Context) error {
	return func(ctx context.Context) error {
		if err := m.world.handOver(ctx); err != nil {
			return err
		}
		b := binary{m: m}
		defer b.reads.end()
		var opens string
		if env.On != nil {
			opens = env.On.Used
		}
		return tui.Run(ctx, tui.Deps{ThisMachine: m.where, Settings: m.cat.Settings, Opens: opens, Calls: m.registry(env).Metas(), Pages: b.pages, Do: b.do, Run: b.call, Open: b.open, Terminal: b.terminal, NoColor: noColor, Stop: m.pool.Close, Watch: tuiWatch(ctx, m)})
	}
}

// invariant: a screen is the page its command prints and an action the
// command it runs: the TUI runs this very binary for the CLI's words, so a
// page and a gesture each have one function. An action runs alone, on
// every system: a read that started before it would show the state it is
// changing, and in a spec world, which each run reads at its start and
// writes back at its end, it would write the action's effect away.
type binary struct {
	m     machine
	turns sync.RWMutex
	reads reads
}

// invariant: a page read is its screen's, so it ends with the TUI: the exit
// stops the reads still running and waits for them, and no child outlives
// the screen to touch what its terminal's next command reads.
type reads struct {
	mu      sync.Mutex
	running sync.WaitGroup
	ended   bool
	over    context.Context
	stop    context.CancelFunc
}

func (r *reads) begin(ctx context.Context) (context.Context, func(), bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.ended {
		return nil, nil, false
	}
	if r.over == nil {
		r.over, r.stop = context.WithCancel(context.Background())
	}
	r.running.Add(1)
	ctx, cancel := context.WithCancel(ctx)
	unhook := context.AfterFunc(r.over, cancel)
	return ctx, func() {
		unhook()
		cancel()
		r.running.Done()
	}, true
}

func (r *reads) end() {
	r.mu.Lock()
	r.ended = true
	if r.stop != nil {
		r.stop()
	}
	r.mu.Unlock()
	r.running.Wait()
}

func (b *binary) pages(ctx context.Context, words []string) (page.Page, error) {
	ctx, done, ok := b.reads.begin(ctx)
	if !ok {
		return page.Page{}, context.Canceled
	}
	defer done()
	b.turns.RLock()
	defer b.turns.RUnlock()
	out, err := b.run(ctx, append(words, "--page"))
	if err != nil {
		return page.Page{}, err
	}
	if out.ExitCode != 0 {
		return page.Page{}, failed(out.Stderr)
	}
	return page.Decode([]byte(out.Stdout))
}

var typed = regexp.MustCompile(`--confirm=(\S+)`)

func (b *binary) do(ctx context.Context, words []string, a tui.Answer) ([]string, error) {
	b.turns.Lock()
	defer b.turns.Unlock()
	switch {
	case a.Typed != "":
		words = append(words, "--confirm="+a.Typed)
	case a.Yes:
		words = append(words, "--yes")
	}
	var stdin string
	if a.Values != nil {
		raw, err := json.Marshal(a.Values)
		if err != nil {
			return nil, err
		}
		words, stdin = append(words, "--"+kit.ValuesOnStdin), string(raw)
	}
	out, err := b.runIn(ctx, words, stdin)
	if err != nil {
		return nil, err
	}
	if out.ExitCode == 0 {
		return skipped(out.Stdout), nil
	}
	f := failed(out.Stderr)
	if why := broken(out.Stdout); why != "" {
		f.Message = why
	}
	if out.ExitCode == 2 && !a.Yes && a.Typed == "" {
		if name := typed.FindStringSubmatch(f.Hint); name != nil {
			return nil, &tui.Asked{Message: f.Message, Name: name[1]}
		}
		if strings.Contains(f.Hint, "--yes") {
			return nil, &tui.Asked{Message: f.Message}
		}
	}
	return nil, f
}

// value: the records a command printed for what it left alone (`–	infra
//
//	skipped (server)`), each as `infra skipped (server)`.
func skipped(stdout string) []string {
	var out []string
	for _, line := range strings.Split(stdout, "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) >= 3 && fields[0] == token.MarkNotFor && strings.HasPrefix(fields[len(fields)-1], "skipped") {
			out = append(out, fields[1]+" "+fields[len(fields)-1])
		}
	}
	return out
}

func (b *binary) run(ctx context.Context, args []string) (local.Output, error) {
	return b.runIn(ctx, args, "")
}

func (b *binary) runIn(ctx context.Context, args []string, stdin string) (local.Output, error) {
	return b.m.sys.Run.Output(ctx, local.Cmd{Name: b.m.sys.Env.Executable(), Args: args, Env: []string{"STATION_TERMINAL="}, Stdin: stdin})
}

func (b *binary) open(ctx context.Context, target string) error {
	return b.exec(ctx, local.Cmd{Name: "open", Args: []string{target}})
}

// invariant: a new terminal is Terminal.app told to run the line, as the
// menu bar's Open in Terminal does.
func (b *binary) terminal(ctx context.Context, line string) error {
	quoted := strings.ReplaceAll(strings.ReplaceAll(line, `\`, `\\`), `"`, `\"`)
	return b.exec(ctx, local.Cmd{Name: "osascript", Args: []string{"-e", `tell application "Terminal" to do script "` + quoted + `"`, "-e", `tell application "Terminal" to activate`}})
}

func (b *binary) exec(ctx context.Context, c local.Cmd) error {
	out, err := b.m.sys.Run.Output(ctx, c)
	if err != nil {
		return err
	}
	if out.ExitCode != 0 {
		return failed(out.Stderr)
	}
	return nil
}

// value: why the one record a command printed broken failed, in its own
// words (the install's log), rather than the run's summing-up line.
func broken(stdout string) string {
	var why []string
	for _, line := range strings.Split(stdout, "\n") {
		if strings.HasPrefix(line, token.MarkBroken+"\t") {
			fields := strings.Split(line, "\t")
			why = append(why, strings.TrimSpace(fields[len(fields)-1]))
		}
	}
	if len(why) != 1 {
		return ""
	}
	return why[0]
}

// invariant: a key's call is the CLI's command for its words, run as this
// very binary and alone, as a gesture is, so a call has one function on
// every surface.
func (b *binary) call(ctx context.Context, words []string) (string, error) {
	b.turns.Lock()
	defer b.turns.Unlock()
	out, err := b.run(ctx, words)
	if err != nil {
		return "", err
	}
	if out.ExitCode != 0 {
		return "", failed(out.Stderr)
	}
	return out.Stdout, nil
}

// invariant: the TUI's watch is the menu bar's: engine.watch held open on
// this Mac's engine for the machines it shows, This Mac by the engine's own
// name for it, until the screen leaves them.
func tuiWatch(ctx context.Context, m machine) func([]string) func() {
	return func(names []string) func() {
		held := make([]string, 0, len(names))
		for _, n := range names {
			if n == m.where.Machine {
				n = engine.ThisMac
			}
			held = append(held, n)
		}
		if m.world != nil {
			return m.world.watch(held)
		}
		watch, cancel := context.WithCancel(ctx)
		done := make(chan struct{})
		go func() {
			defer close(done)
			updates, err := m.engineAPI().c.Watch(watch, engine.CallWatch, engine.WatchIn{Machines: held})
			if err != nil {
				return
			}
			for range updates {
			}
		}()
		return func() {
			cancel()
			<-done
		}
	}
}

func failed(stderr string) *tui.Failed {
	f := &tui.Failed{}
	for _, line := range strings.Split(strings.TrimSpace(stderr), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "Error:"):
			f.Message = strings.TrimSpace(strings.TrimPrefix(line, "Error:"))
		case strings.HasPrefix(line, "Hint:"):
			f.Hint = strings.TrimSpace(strings.TrimPrefix(line, "Hint:"))
		case f.Message == "":
			f.Message = line
		}
	}
	return f
}
