// Package deploy owns the "materialise a compiled world into a
// running container" primitives — splitting a compile tree by
// prefix, docker-cp'ing agent home trees in, syncing the
// allowlisted memory dirs back out at graceful shutdown.
//
// Architect.Spawn composes these helpers into the full spawn
// pipeline; keeping them here makes each step independently
// testable against a mock backend.
package deploy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"spwn.sh/packages/transpile"
	"spwn.sh/packages/container/backend"
	"spwn.sh/packages/platform"
)

// MaterialiseTree splits a compiled tree by top-level prefix and
// delivers each half to its destination:
//
//   - world/*   — written to the host-side worldStateDir (surfaced
//                 into the container via a /world/ bind mount).
//   - agents/*  — docker-cp'd into the already-running container at
//                 /agents/<rest>.
//
// Any other prefix is an error; the caller must namespace every
// tree entry under one of the two.
func MaterialiseTree(ctx context.Context, be backend.Backend, containerID string, tree *transpile.Tree, worldStateDir string) error {
	var firstErr error
	tree.Walk(func(path string, content []byte) {
		if firstErr != nil {
			return
		}
		switch {
		case strings.HasPrefix(path, "world/"):
			full := filepath.Join(worldStateDir, filepath.FromSlash(strings.TrimPrefix(path, "world/")))
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				firstErr = fmt.Errorf("mkdir %s: %w", filepath.Dir(full), err)
				return
			}
			if err := os.WriteFile(full, content, 0o644); err != nil {
				firstErr = fmt.Errorf("write %s: %w", full, err)
				return
			}
		case strings.HasPrefix(path, "agents/"):
			containerPath := "/" + path
			if err := be.CopyTo(ctx, containerID, containerPath, content); err != nil {
				firstErr = fmt.Errorf("cp %s into container: %w", containerPath, err)
				return
			}
		default:
			firstErr = fmt.Errorf("unexpected tree path %q: tree entries must be namespaced under world/ or agents/", path)
			return
		}
	})
	return firstErr
}

// SyncIn copies each agent's host-side home tree (under
// platform.AgentsDir()/<name>/) into the container at
// agentHomes[name]. One-way snapshot at spawn; no live bind.
// Agents whose host dir doesn't exist (first-time scaffolds, memory
// still empty) are silently skipped.
//
// Note: docker's CopyToContainer extracts tar entries as root (tar
// headers don't carry uid/gid today), so every file under
// /agents/<name>/ lands owned root:root. The caller must re-own via
// ChownAgentHomes *after* all docker-cp operations complete —
// including the runtime default-config writes that happen downstream.
// A single chown here wouldn't catch those later writes.
func SyncIn(ctx context.Context, be backend.Backend, containerID string, agentHomes map[string]string) error {
	hostRoot := platform.AgentsDir()
	for agentName, containerHome := range agentHomes {
		hostDir := filepath.Join(hostRoot, agentName)
		if info, err := os.Stat(hostDir); err != nil || !info.IsDir() {
			continue
		}
		if err := be.CopyDirTo(ctx, containerID, containerHome, hostDir); err != nil {
			return fmt.Errorf("copy %s → %s: %w", hostDir, containerHome, err)
		}
	}
	return nil
}

// ChownAgentHomes re-owns every /agents/<name>/ tree to the non-root
// agent user. Must be called after every docker-cp into the agent
// home (SyncIn + runtime default-config writes + any other source).
// Without this, the claude-code PrelaunchShell's credential cp into
// $HOME/.claude/.credentials.json fails silently, and the agent then
// launches un-authenticated and exits 0 with empty output — the
// failure mode that made `spwn agent talk` look like a black hole.
//
// `docker exec` honours the image's USER directive, which the base
// Dockerfile sets to "spwn" — a non-root account that can't chown
// files owned by root. We route through sudo instead; the base image
// grants spwn NOPASSWD sudo specifically so post-cp bookkeeping like
// this stays straightforward.
func ChownAgentHomes(ctx context.Context, be backend.Backend, containerID string, agentHomes map[string]string) error {
	for _, containerHome := range agentHomes {
		if _, err := be.Exec(ctx, containerID, backend.ExecConfig{
			Cmd: []string{"sudo", "chown", "-R", "spwn:spwn", containerHome},
		}); err != nil {
			return fmt.Errorf("chown %s: %w", containerHome, err)
		}
	}
	return nil
}

// SyncOut copies the allowlisted memory subdirs (journal, playbooks)
// from each agent's container home back out to the host. Everything
// else (identity files that didn't change, dotfiles, runtime caches,
// rebuilt runtime-specific entrypoints) stays inside the container
// and is discarded with it.
//
// Knowledge is NOT in the list: it lives at /world/knowledge/ (world-
// scoped, bind-mounted from spwn/worlds/<name>/knowledge/), so
// in-container edits hit the project dir directly — no sync needed.
// Skills aren't synced either: they're build-time dependencies
// injected into /world/skills/, not a runtime-writable memory layer.
//
// Failures are collected as warnings rather than aborting the
// teardown — a best-effort snapshot is better than none, and the
// container is about to be removed anyway.
func SyncOut(ctx context.Context, be backend.Backend, containerID string, agentHomes map[string]string) []string {
	syncDirs := []string{"journal", "playbooks"}
	hostRoot := platform.AgentsDir()

	var warnings []string
	for agentName, containerHome := range agentHomes {
		hostDir := filepath.Join(hostRoot, agentName)
		for _, sub := range syncDirs {
			src := containerHome + "/" + sub
			dst := filepath.Join(hostDir, sub)
			if err := be.CopyDirFrom(ctx, containerID, src, dst); err != nil {
				warnings = append(warnings, fmt.Sprintf("sync %s/%s: %v", agentName, sub, err))
			}
		}
	}
	return warnings
}
