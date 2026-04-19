package architect

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"spwn.sh/packages/dependency/refs"
)

// HookTimeout caps each individual hook script at 5 minutes. Longer
// than a reasonable warm-up, short enough that a runaway hook
// (infinite loop, hung network call, accidental `sleep infinity`)
// doesn't block `spwn up` forever. The operator sees a clear
// "hook: timed out after 5m" message and can debug from there.
var HookTimeout = 5 * time.Minute

// runLifecycleHooks executes every `hook:<phase>` ref from the world
// manifest whose name matches `phase`. Hooks run on the HOST, cwd set
// to projectRoot, with inherited env. A non-zero exit aborts the
// lifecycle (the caller decides whether that's fatal).
//
// Names carry the semantics: `hook:pre-spawn` runs before the
// container starts, `hook:post-destroy` after it's torn down, etc.
// Callers invoke this once per phase with the matching name.
//
// Each hook runs under a per-call context.WithTimeout(HookTimeout)
// so a runaway script can't hang the spawn pipeline indefinitely.
//
// No-ops silently when:
//   - projectRoot is empty (legacy global mode),
//   - no hook refs with the given phase are declared,
//   - the resolved script file is missing (spwn check is the
//     authoring-side gate; spawn is best-effort).
func runLifecycleHooks(ctx context.Context, projectRoot, phase string, deps []string) error {
	if projectRoot == "" || phase == "" {
		return nil
	}
	for _, raw := range deps {
		ref := refs.ParseRef(raw)
		if ref.Kind != refs.KindLocalHook || ref.Name != phase {
			continue
		}
		scriptPath := filepath.Join(projectRoot, "spwn", "hooks", ref.Name+".sh")
		if _, err := os.Stat(scriptPath); err != nil {
			// Missing script — skip silently.
			continue
		}
		hookCtx, cancel := context.WithTimeout(ctx, HookTimeout)
		cmd := exec.CommandContext(hookCtx, scriptPath)
		cmd.Dir = projectRoot
		cmd.Stdout = os.Stderr // non-JSON stepper output already goes to stderr
		cmd.Stderr = os.Stderr
		cmd.Env = os.Environ()
		err := cmd.Run()
		cancel()
		if err != nil {
			if errors.Is(hookCtx.Err(), context.DeadlineExceeded) {
				return fmt.Errorf("hook:%s timed out after %s — check for a runaway command", ref.Name, HookTimeout)
			}
			return fmt.Errorf("hook:%s failed: %w", ref.Name, err)
		}
	}
	return nil
}
