package macos

import (
	"context"
	"errors"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

func (m *Mac) exclusionsWanted() []settings.KeyValue {
	out := make([]settings.KeyValue, len(m.o.BackupExclusions))
	for i, p := range m.o.BackupExclusions {
		out[i] = settings.KeyValue{Key: p, Value: "excluded"}
	}
	return out
}

func (m *Mac) observeExclusions(ctx context.Context, _ []settings.KeyValue) settings.Observation {
	now, err := m.exclusionValues(ctx)
	if errors.Is(err, local.ErrNotFound) {
		return settings.Observation{State: catalog.StateUnknown, Detail: "tmutil not found", After: m.exclusionsWanted()}
	}
	if err != nil {
		return settings.Observation{State: catalog.StateUnknown, Detail: err.Error(), After: m.exclusionsWanted()}
	}
	return judged(now, m.exclusionsWanted(), "excluded from Time Machine", "one of the two is backed up")
}

func (m *Mac) applyExclusions(ctx context.Context) (settings.Change, error) {
	paths := m.o.BackupExclusions
	for _, p := range paths {
		if err := m.files.MkdirAll(p, 0o755); err != nil {
			return settings.Change{}, err
		}
	}
	if _, err := m.must(ctx, "tmutil", append([]string{"addexclusion"}, paths...)...); err != nil {
		return settings.Change{}, err
	}
	return settings.Change{Steps: []settings.ChangeStep{done("Time Machine", "excludes "+strings.Join(paths, ", "))}}, nil
}

func (m *Mac) undoExclusions(ctx context.Context) (settings.Change, error) {
	paths := m.o.BackupExclusions
	if _, err := m.must(ctx, "tmutil", append([]string{"removeexclusion"}, paths...)...); err != nil {
		return settings.Change{}, err
	}
	return settings.Change{Steps: []settings.ChangeStep{done("Time Machine", "backs up "+strings.Join(paths, ", ")+" again")}}, nil
}

// value: each path station keeps out of Time Machine, excluded or backed
// up as tmutil reads it now.
func (m *Mac) exclusionValues(ctx context.Context) ([]settings.KeyValue, error) {
	out, err := m.run.Output(ctx, local.Cmd{Name: "tmutil", Args: append([]string{"isexcluded"}, m.o.BackupExclusions...), Timeout: readBudget})
	if err != nil {
		return nil, err
	}
	excluded := map[string]bool{}
	for _, line := range strings.Split(out.Stdout, "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "[Excluded]"); ok {
			excluded[strings.TrimSpace(rest)] = true
		}
	}
	values := make([]settings.KeyValue, len(m.o.BackupExclusions))
	for i, p := range m.o.BackupExclusions {
		values[i] = settings.KeyValue{Key: p, Value: "backed up"}
		if excluded[p] {
			values[i].Value = "excluded"
		}
	}
	return values, nil
}
