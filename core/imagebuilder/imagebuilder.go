package imagebuilder

import (
	"context"
	"fmt"
	"io"
	"strings"

	"spwn.sh/core/imagebuilder/backend"
	"spwn.sh/core/imagebuilder/internal/dockerfile"
)

// Builder composes Docker images from tool definitions.
type Builder struct {
	registry *Registry
	backend  backend.Backend
}

// New creates a Builder with the given registry and Docker backend.
func New(registry *Registry, b backend.Backend) *Builder {
	return &Builder{registry: registry, backend: b}
}

// BuildRequest describes what image to build.
type BuildRequest struct {
	// BaseDockerfile is the raw base Dockerfile content.
	BaseDockerfile []byte

	// Tools is the list of tool names to include (e.g., ["@spwn/unix", "@spwn/node", "@spwn/qmd"]).
	Tools []string

	// Tag is the Docker image tag to apply.
	Tag string

	// Version is the image version label.
	Version string

	// SkipVerify disables post-build verification.
	SkipVerify bool

	// LogWriter receives build output. If nil, output is discarded.
	LogWriter io.Writer
}

// BuildResult describes the outcome of a successful build.
type BuildResult struct {
	Tag        string
	Tools      []string          // Resolved tool list (after dependency expansion)
	SkillPaths map[string]string // tool name → base path in image
	Cached     bool
}

// Build resolves tools, generates a Dockerfile, builds the image, and verifies it.
func (b *Builder) Build(ctx context.Context, req BuildRequest) (*BuildResult, error) {
	logw := req.LogWriter
	if logw == nil {
		logw = io.Discard
	}

	// Resolve tools (expand deps, topo sort)
	resolved, err := b.registry.Resolve(req.Tools)
	if err != nil {
		return nil, fmt.Errorf("resolve tools: %w", err)
	}

	resolvedNames := make([]string, len(resolved))
	for i, t := range resolved {
		resolvedNames[i] = t.Name()
	}

	fmt.Fprintf(logw, "Resolved tools: %s\n", strings.Join(resolvedNames, ", "))

	// Convert to generator input
	toolInputs := make([]dockerfile.ToolInput, len(resolved))
	for i, t := range resolved {
		spec := t.Install()
		toolInputs[i] = dockerfile.ToolInput{
			Name:         t.Name(),
			Kind:         string(t.Kind()),
			Packages:     spec.Packages,
			Commands:     spec.Commands,
			UserCommands: spec.UserCommands,
			Env:          spec.Env,
			Files:        spec.Files,
			HasSkills:    t.Skills() != nil,
		}
	}

	// Generate Dockerfile
	df := dockerfile.Generate(req.BaseDockerfile, toolInputs, req.Version)

	// Collect extra files for build context (tool files + skills)
	extraFiles := make(map[string][]byte)

	for _, t := range resolved {
		spec := t.Install()
		for path, content := range spec.Files {
			contextPath := fmt.Sprintf("tools/%s%s", t.Name(), path)
			extraFiles[contextPath] = content
		}
	}

	// Collect and add skills
	skills, err := CollectSkills(resolved)
	if err != nil {
		return nil, fmt.Errorf("collect skills: %w", err)
	}
	for path, content := range skills {
		contextPath := fmt.Sprintf("skills%s", path)
		extraFiles[contextPath] = content
	}

	// Build image
	err = b.backend.EnsureImageWithContext(ctx, req.Tag, req.Version, df, extraFiles, logw)
	if err != nil {
		return nil, &BuildError{Tag: req.Tag, Cause: err}
	}

	// Post-build verification
	if !req.SkipVerify {
		fmt.Fprintf(logw, "Verifying tools...\n")
		if err := b.verify(ctx, req.Tag, resolved, logw); err != nil {
			return nil, err
		}
	}

	result := &BuildResult{
		Tag:        req.Tag,
		Tools:      resolvedNames,
		SkillPaths: make(map[string]string),
	}

	for _, t := range resolved {
		if t.Skills() != nil {
			toolName := strings.TrimPrefix(t.Name(), "@")
			result.SkillPaths[t.Name()] = "/world/skills/" + toolName
		}
	}

	return result, nil
}

// verify creates a temporary container from the built image and runs each tool's
// verify commands inside it.
func (b *Builder) verify(ctx context.Context, imageTag string, tools []Tool, logw io.Writer) error {
	containerID, err := b.backend.Create(ctx, backend.ContainerConfig{
		Image: imageTag,
		Name:  "spwn-verify-temp",
	})
	if err != nil {
		return fmt.Errorf("create verify container: %w", err)
	}
	defer func() {
		_ = b.backend.Stop(ctx, containerID)
		_ = b.backend.Remove(ctx, containerID)
	}()

	if err := b.backend.Start(ctx, containerID); err != nil {
		return fmt.Errorf("start verify container: %w", err)
	}

	for _, t := range tools {
		for _, cmd := range t.Verify() {
			output, err := b.backend.ExecOutput(ctx, containerID, []string{"sh", "-c", cmd})
			if err != nil {
				return &VerifyError{
					Tool:    t.Name(),
					Command: cmd,
					Output:  output,
				}
			}
			fmt.Fprintf(logw, "  ✓ %s: %s\n", t.Name(), cmd)
		}
	}

	return nil
}
