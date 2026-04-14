// Package build flattens a spwn project into a reproducible build
// artifact under .spwn/build/.
//
// Layout:
//
//	.spwn/build/
//	├── build.json        - metadata (version, hashes, world+agents)
//	├── manifest.json     - normalized spwn.yaml
//	└── agents/
//	    └── <name>/       - flattened agent tree (every file the
//	                        runtime will read)
//
// A build is always against a single world: only the agents that
// world deploys, and the resolved tool union, are flattened.
package build

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"gopkg.in/yaml.v3"

	intmanifest "spwn.sh/packages/manifest/internal/manifest"
)

// Result is the outcome of a successful Build.
type Result struct {
	Dir       string
	World     string
	Agents    []string
	Tools     []string
	FileCount int
	CreatedAt time.Time
}

// Opts configures Build.
type Opts struct {
	Root        string
	Manifest    *intmanifest.Manifest
	World       string            // which world to build (empty → only)
	AgentPaths  map[string]string // agent name → absolute dir
	ImageDigest string
}

// Metadata is the shape of build.json on disk.
type Metadata struct {
	Version     int       `json:"version"`
	Project     string    `json:"project"`
	CreatedAt   time.Time `json:"created_at"`
	World       string    `json:"world"`
	Agents      []string  `json:"agents"`
	Tools       []string  `json:"tools"`
	ImageDigest string    `json:"image_digest,omitempty"`
	FileCount   int       `json:"file_count"`
	ContentHash string    `json:"content_hash"`
}

// Build writes the artifact under .spwn/build/ for the chosen world.
func Build(opts Opts) (*Result, error) {
	if opts.Manifest == nil {
		return nil, fmt.Errorf("manifest is required")
	}
	if opts.Root == "" {
		return nil, fmt.Errorf("root is required")
	}

	worldName, world, err := pickWorld(opts.Manifest, opts.World)
	if err != nil {
		return nil, err
	}

	buildDir := filepath.Join(opts.Root, ".spwn", "build")
	if err := os.RemoveAll(buildDir); err != nil {
		return nil, fmt.Errorf("clear %s: %w", buildDir, err)
	}
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", buildDir, err)
	}

	fileCount := 0
	hasher := sha256.New()

	// 1. Normalized manifest.json
	mBytes, err := json.MarshalIndent(opts.Manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal manifest: %w", err)
	}
	if err := writeFile(filepath.Join(buildDir, "manifest.json"), mBytes); err != nil {
		return nil, err
	}
	hasher.Write(mBytes)
	fileCount++

	// 2. Flattened agent trees (only this world's agents)
	for _, name := range world.Agents {
		src, ok := opts.AgentPaths[name]
		if !ok {
			continue
		}
		dst := filepath.Join(buildDir, "agents", name)
		n, err := copyTree(src, dst, hasher)
		if err != nil {
			return nil, fmt.Errorf("copy agent %s: %w", name, err)
		}
		fileCount += n
	}

	// 3. Compute the resolved tool union for this world.
	toolUnion := unionTools(opts.AgentPaths, world)

	// 4. build.json metadata
	now := time.Now().UTC()
	meta := Metadata{
		Version:     2,
		Project:     opts.Manifest.Name,
		CreatedAt:   now,
		World:       worldName,
		Agents:      append([]string(nil), world.Agents...),
		Tools:       toolUnion,
		ImageDigest: opts.ImageDigest,
		FileCount:   fileCount,
		ContentHash: hex.EncodeToString(hasher.Sum(nil)),
	}
	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal build.json: %w", err)
	}
	if err := writeFile(filepath.Join(buildDir, "build.json"), metaBytes); err != nil {
		return nil, err
	}

	return &Result{
		Dir:       buildDir,
		World:     worldName,
		Agents:    meta.Agents,
		Tools:     toolUnion,
		FileCount: fileCount,
		CreatedAt: now,
	}, nil
}

func pickWorld(m *intmanifest.Manifest, name string) (string, intmanifest.World, error) {
	if name != "" {
		w, ok := m.Worlds[name]
		if !ok {
			return "", intmanifest.World{}, fmt.Errorf("world %q not found in spwn.yaml", name)
		}
		return name, w, nil
	}
	if len(m.Worlds) == 0 {
		return "", intmanifest.World{}, fmt.Errorf("no worlds declared in spwn.yaml")
	}
	if len(m.Worlds) > 1 {
		return "", intmanifest.World{}, fmt.Errorf("multiple worlds declared; specify one explicitly")
	}
	for n, w := range m.Worlds {
		return n, w, nil
	}
	return "", intmanifest.World{}, fmt.Errorf("unreachable")
}

// unionTools reads each agent's agent.yaml tools list and unions them
// with the world's own tool augmentation, returning a sorted unique
// slice. Best-effort: agent.yaml read errors are silently skipped.
func unionTools(agentPaths map[string]string, world intmanifest.World) []string {
	type agentToolsView struct {
		Tools []string `yaml:"tools"`
	}
	seen := map[string]struct{}{}
	for _, name := range world.Agents {
		dir, ok := agentPaths[name]
		if !ok {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, "agent.yaml"))
		if err != nil {
			continue
		}
		var v agentToolsView
		if err := yaml.Unmarshal(data, &v); err != nil {
			continue
		}
		for _, t := range v.Tools {
			seen[t] = struct{}{}
		}
	}
	for _, t := range world.Tools {
		seen[t] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for t := range seen {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

func copyTree(src, dst string, hasher io.Writer) (int, error) {
	count := 0
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(src, path)
		if relErr != nil {
			return relErr
		}
		if rel == "." {
			return os.MkdirAll(dst, 0o755)
		}
		if shouldSkip(rel) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if hasher != nil {
			hasher.Write([]byte(rel))
			hasher.Write(data)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return err
		}
		count++
		return nil
	})
	return count, err
}

func shouldSkip(rel string) bool {
	switch rel {
	case ".spwn", "node_modules", ".git", ".DS_Store":
		return true
	}
	return false
}

func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// LoadMetadata reads build.json from an existing artifact.
func LoadMetadata(buildDir string) (*Metadata, error) {
	path := filepath.Join(buildDir, "build.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var m Metadata
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse build.json: %w", err)
	}
	return &m, nil
}
