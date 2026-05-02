package architect

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"spwn.sh/packages/automation"
	"spwn.sh/packages/container/backend"
	"spwn.sh/packages/project"
	"spwn.sh/packages/runtimes"
	"spwn.sh/packages/world/models"
)

// maxDispatchOutputBytes caps the runtime output captured into a
// receipt. 32 KB covers the tail of a verbose claude/codex run + a
// final answer; pathological dispatchers writing 100 MB of debug
// noise are best truncated rather than poisoning the receipt log.
const maxDispatchOutputBytes = 32 * 1024

// commandSlugRe is the same kebab-case slug regex the validator
// enforces on command/<name> refs at parse time. The resolver
// re-applies it so a programmatic caller bypassing the validator
// (engine-side, today; or a future API consumer) can't smuggle in
// `command/../../etc/passwd` and read arbitrary .md files outside
// the project's commands directory. Defence-in-depth.
var commandSlugRe = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// AutomationDispatcher implements automation.Dispatcher by reaching
// into a running world's container and exec'ing the agent's runtime
// in one-shot mode with the rendered prompt.
//
// World lookup is by manifest key (worlds.<key>) — the engine speaks
// in the user's vocabulary, not in generated world IDs. Cold worlds
// are auto-spawned per the design: a 6am cron whose target world is
// down should bring the world up, run, and leave it warm.
//
// Phase 4 wires the find-and-exec path. Auto-spawn-when-cold is
// stubbed with a clear error so the next dev pass can lift it once
// the manifest plumbing is in place; the engine receipts the failure
// and the next fire retries.
type AutomationDispatcher struct {
	arc *Architect
}

// NewAutomationDispatcher constructs a dispatcher backed by the
// given Architect. The architect already owns the backend + state
// store; the dispatcher is a thin wrapper that maps the engine's
// "wake (world, agent)" intent onto the existing exec primitives.
func NewAutomationDispatcher(arc *Architect) *AutomationDispatcher {
	return &AutomationDispatcher{arc: arc}
}

// Dispatch satisfies automation.Dispatcher.
func (d *AutomationDispatcher) Dispatch(ctx context.Context, req automation.DispatchRequest) automation.DispatchResult {
	world, err := d.findRunningWorld(ctx, req.World)
	if err != nil {
		return automation.DispatchResult{Err: err}
	}

	// Verify the agent is actually in this world. Validation already
	// caught the static case, but an agent could have been removed
	// at runtime (or the manifest reloaded with a different roster);
	// re-check before exec.
	if !worldHasAgent(world, req.Agent) {
		return automation.DispatchResult{Err: fmt.Errorf("agent %q is not in world %q", req.Agent, req.World)}
	}

	rt, err := d.arc.resolveSpawner(world)
	if err != nil {
		return automation.DispatchResult{Err: fmt.Errorf("resolve runtime for world %s: %w", world.ID, err)}
	}

	// Build the runtime command in one-shot mode. SessionID is left
	// empty — automations are independent invocations, not a
	// continuous conversation. The renderer has already templated
	// the prompt; the runtime just wraps + execs.
	cmd := rt.BuildCommand(runtimes.SpawnConfig{
		AgentName: req.Agent,
		WorldID:   world.ID,
		Prompt:    req.Prompt,
	})
	cmd = rt.OneShotFlags(cmd, "")

	// Per-agent isolation: cwd + HOME + identity env vars match what
	// `spwn agent talk` does. Tools that respect $HOME (claude, git,
	// ssh) land in the agent's persistent home dir on the host.
	agentHome := "/agents/" + req.Agent
	env := append(automationEnv(),
		"HOME="+agentHome,
		"SPWN_AGENT_NAME="+req.Agent,
		"SPWN_WORLD_ID="+req.World,
	)

	// Capture runtime output into a bounded bytes.Buffer so we can
	// surface it in the receipt. ExecConfig.Stdout/Stderr go to
	// these writers instead of os.Stdout/Stderr; the limitWriter
	// caps the buffer at maxDispatchOutputBytes so a chatty runtime
	// can't blow up memory or the receipt log.
	var captured bytes.Buffer
	bound := &limitWriter{w: &captured, max: maxDispatchOutputBytes}

	exitCode, err := d.arc.backend.Exec(ctx, world.ContainerID, backend.ExecConfig{
		Cmd:    cmd,
		Env:    env,
		Stdout: bound,
		Stderr: bound,
		// No TTY — automations run unattended; the runtime adapter's
		// OneShotFlags has already switched to a non-interactive output
		// format.
	})
	output := captured.String()
	if err != nil {
		return automation.DispatchResult{Output: output, Err: fmt.Errorf("exec runtime in world %s: %w", world.ID, err)}
	}
	if exitCode != 0 {
		return automation.DispatchResult{Output: output, Err: fmt.Errorf("runtime exited with code %d", exitCode)}
	}
	return automation.DispatchResult{Output: output}
}

// limitWriter wraps an io.Writer with a hard byte cap. Writes past
// the cap are silently dropped; the underlying buffer never grows
// past `max`. Used to stop a verbose runtime from poisoning the
// receipt log with 100 MB of debug output.
type limitWriter struct {
	w     interface{ Write(p []byte) (int, error) }
	max   int
	count int
}

func (l *limitWriter) Write(p []byte) (int, error) {
	if l.count >= l.max {
		// Already past cap — pretend we wrote everything so the
		// caller (Docker SDK) doesn't error out on a short write.
		return len(p), nil
	}
	remaining := l.max - l.count
	if len(p) > remaining {
		p = p[:remaining]
	}
	n, err := l.w.Write(p)
	l.count += n
	if err != nil {
		return n, err
	}
	return n, nil
}

// findRunningWorld looks up a world by its manifest key. Returns an
// error if the world is not currently running — auto-spawn-when-cold
// is decision 3a from the design, but the manifest needs to be
// plumbed to this layer before we can do it. For Phase 4 the engine
// receipts the failure and the next fire retries; once the user has
// cron triggers in production, this gets lifted.
func (d *AutomationDispatcher) findRunningWorld(ctx context.Context, configName string) (*models.World, error) {
	worlds, err := d.arc.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list worlds: %w", err)
	}
	for _, w := range worlds {
		if w.Config != configName {
			continue
		}
		if w.Status != models.StatusRunning && w.Status != models.StatusIdle {
			return nil, fmt.Errorf("world %q is in status %q (auto-spawn not yet implemented)", configName, w.Status)
		}
		// Re-fetch via Get to pick up the live ContainerID.
		full, err := d.arc.Inspect(ctx, w.ID)
		if err != nil {
			return nil, fmt.Errorf("inspect world %s: %w", w.ID, err)
		}
		return full, nil
	}
	return nil, fmt.Errorf("no running world found for config %q", configName)
}

// worldHasAgent reports whether the world's roster includes name.
// Mirrors the local helper in talk.go so the dispatcher doesn't need
// a cross-package import.
func worldHasAgent(w *models.World, name string) bool {
	if w == nil {
		return false
	}
	if w.Agent == name {
		return true
	}
	for _, a := range w.Agents {
		if a.Name == name {
			return true
		}
	}
	return false
}

// automationEnv returns the env block every automation exec inherits.
// Mirrors agentEnv() in agent.go but is kept separate so changes to
// the human-talk path don't accidentally widen the unattended
// path's env exposure.
func automationEnv() []string {
	return []string{
		"PATH=/usr/local/bin:/usr/bin:/bin",
	}
}

// ── CommandResolver ─────────────────────────────────────────────────

// CommandFileResolver implements automation.CommandResolver by reading
// `command/<name>` refs from <project-root>/spwn/commands/<name>.md.
// One file = one prompt template. Project-root is fixed at
// construction; multi-project setups instantiate one resolver per
// project.
type CommandFileResolver struct {
	ProjectRoot string
}

// NewCommandFileResolver constructs a resolver rooted at the project
// directory containing spwn/commands/.
func NewCommandFileResolver(projectRoot string) *CommandFileResolver {
	return &CommandFileResolver{ProjectRoot: projectRoot}
}

// Resolve loads the markdown body of a command ref. Returns a
// descriptive error when the ref is malformed or the file is
// missing — surfaces in the engine's failed-receipt path so the
// user sees "command file not found" in the dashboard, not a
// silent dispatch.
func (r *CommandFileResolver) Resolve(ref string) (string, error) {
	const prefix = "command/"
	if !strings.HasPrefix(ref, prefix) {
		return "", fmt.Errorf("ref %q must use the command/<name> form", ref)
	}
	name := strings.TrimPrefix(ref, prefix)
	if name == "" {
		return "", fmt.Errorf("empty command name")
	}
	// Path-traversal guard: enforce the same slug regex the
	// validator applies at parse time. Without this, a ref like
	// `command/../../etc/passwd` would walk above the project root
	// and read arbitrary .md files. The validator already rejects
	// such refs in static manifests, but the resolver is the only
	// thing the engine relies on at fire time — a programmatic
	// caller bypassing the validator would otherwise sneak through.
	if !commandSlugRe.MatchString(name) {
		return "", fmt.Errorf("command name %q must be a kebab-case slug (^[a-z][a-z0-9-]*$)", name)
	}
	path := filepath.Join(r.ProjectRoot, "spwn", "commands", name+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return string(data), nil
}

// ── Engine factory ──────────────────────────────────────────────────

// AutomationEngineConfig bundles the per-project bits the engine
// needs from the architect's perspective. Construction order matters:
// the engine needs a Dispatcher that knows about the architect, and
// a CommandResolver rooted at the project — both are derived here so
// callers don't have to wire them by hand.
type AutomationEngineConfig struct {
	// ProjectRoot is the directory containing spwn.yaml + spwn/.
	// Used to resolve command/<name> refs and to anchor the
	// receipts + state files.
	ProjectRoot string

	// Manifest is the parsed spwn.yaml. The factory reads
	// Manifest.Worlds[*].Automations.
	Manifest *project.Manifest

	// FS, when non-nil, enables fs triggers. Production passes the
	// fsnotify-backed source; tests can omit if the project has only
	// cron automations.
	FS *automation.FSWatcher

	// Clock defaults to RealClock when nil. Tests inject FakeClock.
	Clock automation.Clock
}

// NewAutomationEngine assembles the engine for one project. Wires:
//   - architect-backed Dispatcher
//   - file-based CommandResolver rooted at the project
//   - file-based ReceiptWriter at <root>/.spwn/runs.jsonl
//   - file-based StateStore at <root>/.spwn/automations/state.json
//   - the optional FSWatcher passed in
//
// Returns the engine; the caller is responsible for Register +
// Start + Stop. The factory does the wiring; lifecycle is the
// caller's. This separation lets the architect daemon hold one
// engine per project while the spwn-up CLI path can build a
// transient engine without daemonising.
func (a *Architect) NewAutomationEngine(cfg AutomationEngineConfig) (*automation.Engine, error) {
	if cfg.ProjectRoot == "" {
		return nil, fmt.Errorf("automations: ProjectRoot required")
	}
	if cfg.Manifest == nil {
		return nil, fmt.Errorf("automations: Manifest required")
	}
	clock := cfg.Clock
	if clock == nil {
		clock = automation.RealClock{}
	}

	receiptsPath := filepath.Join(cfg.ProjectRoot, ".spwn", "runs.jsonl")
	statePath := filepath.Join(cfg.ProjectRoot, ".spwn", "automations", "state.json")

	eng, err := automation.New(automation.Config{
		Clock:      clock,
		Dispatcher: NewAutomationDispatcher(a),
		Receipts:   automation.NewFileReceiptWriter(receiptsPath),
		State:      automation.NewFileStateStore(statePath),
		FS:         cfg.FS,
		Commands:   NewCommandFileResolver(cfg.ProjectRoot),
	})
	if err != nil {
		return nil, fmt.Errorf("automation engine: %w", err)
	}

	for wname, w := range cfg.Manifest.Worlds {
		if len(w.Automations) == 0 {
			continue
		}
		if err := eng.Register(wname, w.Automations); err != nil {
			return nil, fmt.Errorf("register world %s automations: %w", wname, err)
		}
	}
	return eng, nil
}

