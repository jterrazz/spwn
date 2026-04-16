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
	"strings"

	runtimes "spwn.sh/packages/runtimes"
	"spwn.sh/catalog"
	ib "spwn.sh/packages/image"
	ibbase "spwn.sh/packages/image/base"
	"spwn.sh/packages/platform"
	"spwn.sh/packages/world/backend"
	"spwn.sh/packages/runtimes/claude_code/compile"
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

	// Resolve architect tools via the image package to generate install steps
	reg := ib.NewRegistry()
	if err := catalog.RegisterDefaults(reg); err != nil {
		return fmt.Errorf("register tools: %w", err)
	}
	if err := runtimes.RegisterDefaults(reg); err != nil {
		return fmt.Errorf("register runtimes: %w", err)
	}
	// The architect needs these tools installed in the image.
	// @spwn/cli is handled separately (cross-compiled binary), so we only need
	// the other tools that @spwn/architect depends on.
	architectTools := []string{"@spwn/unix", "@spwn/node", "@spwn/claude-code", "@spwn/docker-cli"}
	resolved, err := reg.Resolve(architectTools)
	if err != nil {
		return fmt.Errorf("resolve architect tools: %w", err)
	}

	// Convert to generator inputs and generate the Dockerfile.
	// SkipFooter: architect has its own COPY/entrypoint directives appended below.
	// User/Home: architect image uses "architect" user, not "spwn".
	toolInputs := ib.ToolsToInputs(resolved)
	df := ib.GenerateDockerfile(ibbase.ArchitectDockerfile, toolInputs, platform.ArchitectImageVersion, ib.GenerateOpts{
		SkipFooter: true,
		User:       "architect",
		Home:       "/home/architect",
	})

	// Append architect-specific Dockerfile directives (COPY binary, system files, entrypoint)
	// Collect and template UserCommands for the architect user
	var userCmdLines string
	for _, t := range resolved {
		for _, cmd := range t.Install().UserCommands {
			cmd = strings.ReplaceAll(cmd, "{{.Home}}", "/home/architect")
			cmd = strings.ReplaceAll(cmd, "{{.User}}", "architect")
			userCmdLines += fmt.Sprintf("RUN %s\n", cmd)
		}
	}

	architectFooter := `
# Architect: cross-compiled spwn binary
COPY spwn /usr/local/bin/spwn
RUN chmod +x /usr/local/bin/spwn

# Architect identity and skills
COPY system/architect/ARCHITECT.md /me/ARCHITECT.md
COPY system/AGENTS.md /me/AGENTS.md
COPY system/skills/ /me/skills/
COPY system/architect/skills/ /me/skills/
RUN chown -R architect:architect /me /home/architect

# Entrypoint aligns architect user groups with host docker.sock GID
COPY entrypoint.sh /usr/local/bin/architect-entrypoint.sh
RUN chmod +x /usr/local/bin/architect-entrypoint.sh

# User-level tool configuration (runs as architect)
USER architect
` + userCmdLines + `
# Switch back to root for entrypoint (needs usermod for docker socket GID)
# Claude Code runs via: docker exec -u architect
USER root
WORKDIR /me
ENTRYPOINT ["/usr/local/bin/architect-entrypoint.sh"]
CMD ["sleep", "infinity"]
`
	df = append(df, []byte(architectFooter)...)

	// Build the context map with architect-specific files
	contextFiles := make(map[string][]byte)

	// Cross-compiled spwn binary
	contextFiles["spwn"] = binaryData

	// Entrypoint script for Docker socket GID alignment
	contextFiles["entrypoint.sh"] = ibbase.ArchitectEntrypoint

	// Architect system files (ARCHITECT.md, AGENTS.md, skills, stack)
	for path, content := range claudecode.ArchitectSystemFiles() {
		contextFiles[path] = []byte(content)
	}

	// Build the image
	_, err = docker.EnsureImageWithContext(
		ctx,
		platform.ArchitectImage,
		platform.ArchitectImageVersion,
		df,
		contextFiles,
		logw,
	)
	return err
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
func findSourceRoot() (string, error) {
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, s := range bi.Settings {
			if s.Key == "vcs.directory" && s.Value != "" {
				if isSpwnRoot(s.Value) {
					return s.Value, nil
				}
			}
		}
	}

	if exe, err := os.Executable(); err == nil {
		if root := findRootUpward(filepath.Dir(exe)); root != "" {
			return root, nil
		}
	}

	if cwd, err := os.Getwd(); err == nil {
		if root := findRootUpward(cwd); root != "" {
			return root, nil
		}
	}

	return "", fmt.Errorf("cannot find spwn source tree.\n" +
		"The architect image requires cross-compilation from source.\n" +
		"Make sure you're running spwn from within the source tree, or that the binary was built with VCS info")
}

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

func isSpwnRoot(dir string) bool {
	goWork := filepath.Join(dir, "go.work")
	mainGo := filepath.Join(dir, "apps", "cli", "cmd", "spwn", "main.go")
	_, err1 := os.Stat(goWork)
	_, err2 := os.Stat(mainGo)
	return err1 == nil && err2 == nil
}
