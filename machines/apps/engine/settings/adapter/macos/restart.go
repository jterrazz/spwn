package macos

import (
	"context"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

// outside: on Apple Silicon SleepOnPowerButton and hibernatemode are firmware-controlled
// and absent from `pmset -g custom`, so they are neither set nor checked.
func (m *Mac) observeRestart(ctx context.Context, want []settings.KeyValue) settings.Observation {
	current, _ := m.pmset(ctx)
	return judged(restartValues(current, want), want, restartDetail(current), "partly applied")
}

// value: what a Mac that restarts reads as, naming the key that starts it
// when autorestart itself reads 0.
func restartDetail(current map[string]string) string {
	const ok = "restarts after a power cut"
	if current[restartOnLoss] == "0" && current[restartAtConnect] == "1" {
		return ok + ": starts when plugged in (autorestartatconnect 1, autorestart 0)"
	}
	return ok
}

func (m *Mac) applyRestart(ctx context.Context, it catalog.Setting, _ map[string]string) (settings.Change, error) {
	policy := settings.Sets(it)
	current, _ := m.pmset(ctx)
	change := settings.Change{Before: restartValues(current, policy)}
	args := make([]string, 0, 3+2*len(policy))
	args = append(args, "-a")
	for _, kv := range policy {
		key := kv.Key
		if _, known := current[key]; key == restartAtConnect && !known {
			key = restartOnLoss
		}
		args = append(args, key, kv.Value)
		change.Steps = append(change.Steps, done(key+"="+kv.Value, ""))
	}
	if _, err := m.must(ctx, "pmset", args...); err != nil {
		change.Steps = nil
		return change, err
	}
	after, _ := m.pmset(ctx)
	change.After = restartValues(after, policy)
	return change, nil
}

func (m *Mac) undoRestart(ctx context.Context, it catalog.Setting) (settings.Change, error) {
	policy := settings.Sets(it)
	current, _ := m.pmset(ctx)
	change := settings.Change{Before: restartValues(current, policy)}
	args := make([]string, 0, 3+2*len(policy))
	args = append(args, "-a")
	for _, kv := range policy {
		keys := []string{kv.Key}
		if kv.Key == restartAtConnect {
			keys = restartKeys(current)
		}
		for _, key := range keys {
			args = append(args, key, "0")
			change.Steps = append(change.Steps, done(key+"=0", ""))
		}
	}
	if _, err := m.must(ctx, "pmset", args...); err != nil {
		change.Steps = nil
		return change, err
	}
	after, _ := m.pmset(ctx)
	change.After = restartValues(after, policy)
	return change, nil
}

// outside: pmset(1) names two ways a Mac starts by itself, autorestart
// ("automatic restart on power loss") and autorestartatconnect ("automatic
// restart on power connect"); System Settings' "Start up when power is
// connected: Always" writes autorestartatconnect alone, and a Mac whose
// pmset lists no autorestartatconnect knows autorestart only.
const (
	restartAtConnect = "autorestartatconnect"
	restartOnLoss    = "autorestart"
)

// value: the keys this Mac's pmset starts it by, autorestart alone where it
// lists neither.
func restartKeys(current map[string]string) []string {
	var keys []string
	for _, key := range []string{restartAtConnect, restartOnLoss} {
		if _, known := current[key]; known {
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 {
		return []string{restartOnLoss}
	}
	return keys
}

func (m *Mac) pmset(ctx context.Context) (map[string]string, bool) {
	out, ok := m.output(ctx, "pmset", "-g", "custom")
	if !ok {
		return nil, false
	}
	current := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		if fields := strings.Fields(line); len(fields) >= 2 {
			current[fields[0]] = fields[1]
		}
	}
	return current, true
}

// value: the policy's keys as pmset reads them now, a key it does not print
// empty; autorestartatconnect reads 1 when either key starts the Mac.
func restartValues(current map[string]string, policy []settings.KeyValue) []settings.KeyValue {
	if current == nil {
		return nil
	}
	values := make([]settings.KeyValue, 0, len(policy))
	for _, kv := range policy {
		v, known := current[kv.Key]
		if kv.Key == restartAtConnect && (current[restartOnLoss] == "1" || !known) {
			v = current[restartOnLoss]
		}
		values = append(values, settings.KeyValue{Key: kv.Key, Value: v})
	}
	return values
}
