package architect

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"

	"spwn.sh/core/foundation"
	"spwn.sh/core/universe/internal/backend"
	"spwn.sh/core/universe/internal/physics"
	"spwn.sh/platform/images"
)

// BuildArchitectImage cross-compiles the spwn binary for linux/amd64 and builds
// the architect Docker image with all required context files (system files,
// entrypoint script, and the cross-compiled binary).
func BuildArchitectImage(ctx context.Context, docker *backend.Docker, logw io.Writer) error {
	if logw == nil {
		logw = io.Discard
	}

	// Cross-compile the spwn binary for linux/amd64
	fmt.Fprintln(logw, "Cross-compiling spwn for linux/amd64...")
	spwnBinary, err := crossCompileSpwn(ctx, logw)
	if err != nil {
		return fmt.Errorf("cross-compile spwn: %w", err)
	}
	defer os.Remove(spwnBinary)

	binaryData, err := os.ReadFile(spwnBinary)
	if err != nil {
		return fmt.Errorf("read cross-compiled binary: %w", err)
	}

	// Build the context map with all files needed by the Dockerfile
	contextFiles := make(map[string][]byte)

	// Add the spwn binary
	contextFiles["spwn"] = binaryData

	// Add the entrypoint script
	contextFiles["entrypoint.sh"] = images.ArchitectEntrypoint

	// Add all architect system files (ARCHITECT.md, AGENTS.md, skills, etc.)
	for path, content := range physics.ArchitectSystemFiles() {
		contextFiles[path] = []byte(content)
	}

	// Build the image
	return docker.EnsureImageWithContext(
		ctx,
		foundation.ArchitectImage,
		foundation.ArchitectImageVersion,
		images.DockerfileArchitect,
		contextFiles,
		logw,
	)
}

// crossCompileSpwn builds the spwn CLI binary for linux/amd64.
// It returns the path to the temporary binary file. The caller must remove it.
func crossCompileSpwn(ctx context.Context, logw io.Writer) (string, error) {
	srcRoot, err := findSourceRoot()
	if err != nil {
		return "", err
	}

	// Determine target architecture: use the host's arch for Docker (works with
	// Docker Desktop which runs containers matching the host arch).
	goarch := runtime.GOARCH

	tmpFile, err := os.CreateTemp("", "spwn-linux-*")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	tmpFile.Close()

	cmd := exec.CommandContext(ctx, "go", "build", "-o", tmpFile.Name(), "./apps/cli/cmd/spwn")
	cmd.Dir = srcRoot
	cmd.Env = append(os.Environ(),
		"GOOS=linux",
		"GOARCH="+goarch,
		"CGO_ENABLED=0",
	)
	cmd.Stdout = logw
	cmd.Stderr = logw

	if err := cmd.Run(); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("go build failed: %w\nMake sure Go is installed and the spwn source tree is available", err)
	}

	return tmpFile.Name(), nil
}

// findSourceRoot locates the spwn workspace root directory.
// It tries several strategies:
// 1. Check build info for VCS directory
// 2. Walk up from the running executable looking for go.work
// 3. Walk up from the current working directory looking for go.work
func findSourceRoot() (string, error) {
	// Strategy 1: check build info VCS directory
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, s := range bi.Settings {
			if s.Key == "vcs.directory" && s.Value != "" {
				if isSpwnRoot(s.Value) {
					return s.Value, nil
				}
			}
		}
	}

	// Strategy 2: walk up from executable path
	if exe, err := os.Executable(); err == nil {
		if root := findRootUpward(filepath.Dir(exe)); root != "" {
			return root, nil
		}
	}

	// Strategy 3: walk up from cwd
	if cwd, err := os.Getwd(); err == nil {
		if root := findRootUpward(cwd); root != "" {
			return root, nil
		}
	}

	return "", fmt.Errorf("cannot find spwn source tree.\n" +
		"The architect image requires cross-compilation from source.\n" +
		"Make sure you're running spwn from within the source tree, or that the binary was built with VCS info")
}

// findRootUpward walks up from dir looking for a directory containing go.work
// and apps/cli/cmd/spwn/main.go (the spwn workspace root).
func findRootUpward(dir string) string {
	for {
		if isSpwnRoot(dir) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// isSpwnRoot checks if dir is the spwn workspace root.
func isSpwnRoot(dir string) bool {
	goWork := filepath.Join(dir, "go.work")
	mainGo := filepath.Join(dir, "apps", "cli", "cmd", "spwn", "main.go")
	_, err1 := os.Stat(goWork)
	_, err2 := os.Stat(mainGo)
	return err1 == nil && err2 == nil
}
