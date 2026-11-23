package wire

import (
	"maps"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

// value: the label of the bundle's job args run, when they are the engine's.
func jobOf(args []string) string {
	if slices.Equal(args, catalog.EngineArgs) {
		return catalog.BundledEngineLabel
	}
	return ""
}

// invariant: the job of Spwn.app's engine starts with launchd's bare
// environment, since its definition in the sealed bundle holds none, so it
// takes the environment `station engine start` recorded, which wins over
// launchd's, and writes to its log (ADR-073, mode A). Any other run, and
// the bare binary's job, keep the environment they were given.
func asBundledJob(args, env []string) []string {
	label := jobOf(args)
	if label == "" || local.BundleOf(local.Executable(), catalog.BundledEnginePath) == "" {
		return env
	}
	home := ""
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, "HOME="); ok {
			home = v
		}
	}
	if home == "" {
		return env
	}
	start, err := local.ReadJobStart(catalog.JobStartPath(home))
	if err != nil {
		return env
	}
	if log := start.Logs[label]; log != "" {
		_ = local.LogTo(log)
	}
	return withStart(env, start.Env)
}

func withStart(env []string, start map[string]string) []string {
	kept := make([]string, 0, len(env)+len(start))
	for _, kv := range env {
		if name, _, _ := strings.Cut(kv, "="); start[name] == "" {
			kept = append(kept, kv)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(start)) {
		kept = append(kept, name+"="+start[name])
	}
	return kept
}
