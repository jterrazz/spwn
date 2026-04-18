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
	"spwn.sh/packages/runtimes"
)

// injectRuntimeConfig computes the merged runtime config for the
// world's runtime backend and writes it into the container's runtime
// settings file.
//
// The runtime adapter (runtimes.Adapter, resolved by runtimeName)
// owns two pieces of knowledge:
//
//   - CatalogRef — the dep-facing identifier ("spwn:claude-code")
//     that tool authors key their runtime-config: blocks against.
//   - Spawn.ContainerConfigPath — the container-side path to the
//     baseline settings file the adapter wants merged in place.
//
// When the resolved adapter has no Spawn or no ContainerConfigPath,
// the injection is skipped — the runtime doesn't participate in the
// runtime-config merge path (codex today).
//
// The merge is shallow (last write wins per top-level key). The
// baseline settings file — written by the runtime tool's UserCommands
// at image-build time — is read back, shallow-merged with every
// dependency's Config(runtimeRef) output, and rewritten in place.
func injectRuntimeConfig(ctx context.Context, be backend.Backend, containerID, runtimeName string, resolved []tool.Tool) error {
	adapter, ok := runtimes.Get(runtimeName)
	if !ok || adapter.Spawn == nil {
		return nil
	}
	settingsPath := adapter.Spawn.ContainerConfigPath()
	if settingsPath == "" {
		return nil
	}
	ref := adapter.CatalogRef
	if ref == "" {
		return nil
	}

	configs := resolver.CollectRuntimeConfigs(resolved, ref)
	if len(configs) == 0 {
		return nil
	}

	// Read the container's baseline settings file. Missing file is
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
