// Package build flattens a spwn project into a reproducible build
// artifact under .spwn/build/.
//
// The artifact is a self-contained snapshot of "what the next spawn
// would see". It lets `spwn up` skip straight to container creation
// when nothing changed, and it lets teammates / CI ship an exact
// pinned project by tar-ing the directory.
//
// Layout:
//
//	.spwn/build/
//	├── build.json        - metadata (version, timestamps, hashes)
//	├── manifest.json     - normalized spwn.yaml
//	├── agents/
//	│   └── <name>/       - flattened agent tree (every file the
//	│                       runtime will read, including resolved
//	│                       @-imports)
//	└── worlds/
//	    └── <name>.yaml   - world config (verbatim)
//
// The Docker image digest, when a build includes one, goes into
// build.json. Image bytes stay in the Docker daemon - the artifact
// just pins the reference.
package build

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	intmanifest "spwn.sh/packages/manifest/internal/manifest"
)

// Result is the outcome of a successful Build.
type Result struct {
	// Dir is the absolute path to the build artifact directory
	// (typically <projectRoot>/.spwn/build/).
	Dir string

	// Agents lists the agents that were flattened into the artifact.
	Agents []string

	// World is the name of the world config written to the artifact.
	World string

	// FileCount is the total number of files copied.
	FileCount int

	// CreatedAt is when the build was written.
	CreatedAt time.Time
}

// Opts configures Build.
type Opts struct {
	// Root is the absolute project root (the directory containing
	// spwn.yaml).
	Root string

	// Manifest is the parsed manifest. Required.
	Manifest *intmanifest.Manifest

	// AgentPaths is the absolute path to each agent in the manifest
	// (positional match with Manifest.Agents).
	AgentPaths []string

	// WorldPath is the absolute path to the world config file.
	WorldPath string

	// ImageDigest, when non-empty, pins the Docker image the build
	// was produced against. Lands in build.json.
	ImageDigest string
}

// Metadata is the shape of build.json on disk.
type Metadata struct {
	Version     int       `json:"version"`
	Project     string    `json:"project"`
	CreatedAt   time.Time `json:"created_at"`
	Agents      []string  `json:"agents"`
	World       string    `json:"world"`
	ImageDigest string    `json:"image_digest,omitempty"`
	FileCount   int       `json:"file_count"`
	ContentHash string    `json:"content_hash"`
}

// Build writes the artifact under .spwn/build/ and returns a Result.
// The directory is wiped and recreated fresh on every call - the
// artifact is atomic from the caller's perspective: either it reflects
// the current project state or the call errors out.
func Build(opts Opts) (*Result, error) {
	if opts.Manifest == nil {
		return nil, fmt.Errorf("manifest is required")
	}
	if opts.Root == "" {
		return nil, fmt.Errorf("root is required")
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

	// 2. Flattened agent trees
	for i, name := range opts.Manifest.Agents {
		if i >= len(opts.AgentPaths) {
			continue
		}
		src := opts.AgentPaths[i]
		dst := filepath.Join(buildDir, "agents", name)
		n, err := copyTree(src, dst, hasher)
		if err != nil {
			return nil, fmt.Errorf("copy agent %s: %w", name, err)
		}
		fileCount += n
	}

	// 3. World config
	if opts.WorldPath != "" {
		data, err := os.ReadFile(opts.WorldPath)
		if err != nil {
			return nil, fmt.Errorf("read world %s: %w", opts.WorldPath, err)
		}
		worldDst := filepath.Join(buildDir, "worlds", opts.Manifest.World+".yaml")
		if err := writeFile(worldDst, data); err != nil {
			return nil, err
		}
		hasher.Write(data)
		fileCount++
	}

	// 4. build.json metadata
	now := time.Now().UTC()
	meta := Metadata{
		Version:     1,
		Project:     opts.Manifest.Name,
		CreatedAt:   now,
		Agents:      opts.Manifest.Agents,
		World:       opts.Manifest.World,
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
		Agents:    opts.Manifest.Agents,
		World:     opts.Manifest.World,
		FileCount: fileCount,
		CreatedAt: now,
	}, nil
}

// copyTree walks src and replicates it under dst. Returns the number
// of files copied. Skips anything matching a .spwn/ or node_modules/
// pattern so artifacts don't balloon when the agent directory happens
// to share a root with a larger repo.
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

// shouldSkip filters out directories that are never part of a spwn
// agent at runtime.
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

// LoadMetadata reads build.json from an existing artifact. Returns
// (nil, nil) when the file doesn't exist - callers use this to
// decide whether a build is needed.
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
