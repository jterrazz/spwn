package architect

import (
	"context"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"

	"spwn.sh/packages/container/backend"
	"spwn.sh/packages/dependency/resolver"
	"spwn.sh/packages/dependency/tool"
)

// injectRuntimeConfig computes the merged runtime config for the
// world's runtime backend and writes it into the container's
// runtime settings file.
//
// Current scope: only spwn:claude-code has a known settings path
// (/home/spwn/.claude/settings.json). The container's baseline
// settings file — written by the claude_code tool's UserCommands
// at image build time — is read back, shallow-merged with every
// dependency's Config(runtime) output (last write wins), and
// rewritten in place.
//
// When no dependency targets the runtime, this is a no-op: the
// baseline settings.json stays untouched.
//
// Additional runtimes can grow their own branch here as
// dependencies for them materialise.
func injectRuntimeConfig(ctx context.Context, be backend.Backend, containerID string, resolved []tool.Tool) error {
	// The dependency-facing runtime identifier is the same as the
	// image builder's runtime tool name. Spawn always installs
	// spwn:claude-code, so hard-code it here until a second
	// runtime lands (codex is built but has no dependency target
	// yet).
	const runtimeName = "spwn:claude-code"
	const settingsPath = "/home/spwn/.claude/settings.json"

	configs := resolver.CollectRuntimeConfigs(resolved, runtimeName)
	if len(configs) == 0 {
		return nil
	}

	// Read the container's baseline settings.json. Missing file is
	// fine — an empty base layer merges cleanly.
	baseStdout, _ := be.ExecOutput(ctx, containerID, []string{"sh", "-c", "cat " + settingsPath + " 2>/dev/null || true"})
	base := []byte(strings.TrimSpace(baseStdout))

	merged, err := resolver.MergeRuntimeConfig(base, configs...)
	if err != nil {
		return fmt.Errorf("merge config: %w", err)
	}

	// Encode the merged JSON as base64 and pipe it through the
	// shell so we don't have to worry about escaping JSON inside
	// sh -c.
	encoded := base64.StdEncoding.EncodeToString(merged)
	script := fmt.Sprintf(
		"mkdir -p %s && printf '%%s' '%s' | base64 -d > %s",
		filepath.Dir(settingsPath), encoded, settingsPath,
	)
	if _, err := be.ExecOutput(ctx, containerID, []string{"sh", "-c", script}); err != nil {
		return fmt.Errorf("write merged settings: %w", err)
	}
	return nil
}
