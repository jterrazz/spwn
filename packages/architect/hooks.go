package architect

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"spwn.sh/packages/dependency/refs"
)

// runLifecycleHooks executes every `hook:<phase>` ref from the world
// manifest whose name matches `phase`. Hooks run on the HOST, cwd set
// to projectRoot, with inherited env. A non-zero exit aborts the
// lifecycle (the caller decides whether that's fatal).
//
// Names carry the semantics: `hook:pre-spawn` runs before the
// container starts, `hook:post-destroy` after it's torn down, etc.
// Callers invoke this once per phase with the matching name.
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
		cmd := exec.CommandContext(ctx, scriptPath)
		cmd.Dir = projectRoot
		cmd.Stdout = os.Stderr // non-JSON stepper output already goes to stderr
		cmd.Stderr = os.Stderr
		cmd.Env = os.Environ()
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("hook:%s failed: %w", ref.Name, err)
		}
	}
	return nil
}
