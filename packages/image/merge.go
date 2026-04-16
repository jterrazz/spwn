package image

import (
	"encoding/json"
	"fmt"
)

// MergeRuntimeConfig shallow-merges JSON objects: parses each input as
// map[string]any and overlays later values on top of earlier ones at
// the top level. Used at spawn time to combine a runtime's baseline
// config (e.g. the claude-code settings.json that ships in the image)
// with the per-pack config snippets returned by Tool.Config.
//
// Merge rule: last write wins per top-level key. Values are not
// recursively merged — a later pack that declares `mcpServers: {foo:
// ...}` replaces an earlier pack's `mcpServers` entirely. This is
// deliberate: deep merge hides conflicts, shallow merge surfaces them
// and keeps pack authoring trivial to reason about. Revisit if real
// packs start needing deeper composition.
//
// nil inputs are skipped. An empty result (no non-nil inputs) round-
// trips as "{}".
func MergeRuntimeConfig(base []byte, additions ...[]byte) ([]byte, error) {
	merged := map[string]any{}

	all := make([][]byte, 0, 1+len(additions))
	if len(base) > 0 {
		all = append(all, base)
	}
	for _, add := range additions {
		if len(add) > 0 {
			all = append(all, add)
		}
	}

	for i, raw := range all {
		var layer map[string]any
		if err := json.Unmarshal(raw, &layer); err != nil {
			return nil, fmt.Errorf("merge layer %d: %w", i, err)
		}
		for k, v := range layer {
			merged[k] = v
		}
	}

	out, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal merged config: %w", err)
	}
	return out, nil
}

// CollectRuntimeConfigs walks a list of resolved tools, filters to
// Packs whose Runtimes() includes the given runtime, and returns
// the non-nil Config() outputs in resolution order (last wins).
//
// This helper exists so the architect doesn't need to replicate the
// Pack runtime-gating logic.
func CollectRuntimeConfigs(resolved []Tool, runtime string) [][]byte {
	var out [][]byte
	for _, t := range resolved {
		cfg := PluginConfig(t, runtime)
		if cfg != nil {
			out = append(out, cfg)
		}
	}
	return out
}
