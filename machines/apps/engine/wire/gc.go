package wire

import (
	"runtime/debug"
	"strings"
)

// value: the heap a short command may reach before the collector runs:
// one allocates a few MB and exits, so this is the net, never the budget.
const shortHeap = 256 << 20

// invariant: a run that lives (the engine, its attach over ssh, a VM's
// guest serving its calls) keeps the runtime's collector; every other run
// is short, and spends no CPU collecting garbage its exit frees, the heap
// bounded by shortHeap. GOGC or GOMEMLIMIT in the environment is the
// owner's word and is kept.
func shortLived(args []string, vars map[string]string) bool {
	if _, set := vars["GOGC"]; set {
		return false
	}
	if _, set := vars["GOMEMLIMIT"]; set {
		return false
	}
	return !serving(args) && (len(args) < 2 || args[0] != "engine" || args[1] != "run")
}

func varsOf(env []string) map[string]string {
	vars := map[string]string{}
	for _, kv := range env {
		if k, v, ok := strings.Cut(kv, "="); ok {
			vars[k] = v
		}
	}
	return vars
}

func collectFor(args []string, vars map[string]string) {
	if shortLived(args, vars) {
		debug.SetGCPercent(-1)
		debug.SetMemoryLimit(shortHeap)
	}
}
