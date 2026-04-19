package compile

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"strings"

	"spwn.sh/packages/container/backend"
	"spwn.sh/packages/compile/internal/dockerfile"
	"spwn.sh/packages/dependency/resolver"
	"spwn.sh/packages/dependency/tool"
)

// Builder composes Docker images from tool definitions.
type Builder struct {
	registry *resolver.Registry
	backend  backend.Backend
}

// New creates a Builder with the given registry and Docker backend.
func New(registry *resolver.Registry, b backend.Backend) *Builder {
	return &Builder{registry: registry, backend: b}
}

// BuildRequest describes what image to build.
type BuildRequest struct {
	// BaseDockerfile is the raw base Dockerfile content.
	BaseDockerfile []byte

	// Tools is the list of tool names to include (e.g., ["spwn:unix", "spwn:node", "spwn:qmd"]).
	Tools []string

	// Tag is the Docker image tag to apply.
	Tag string

	// ForceRebuild bypasses the content hash cache check and rebuilds
	// from scratch. The default is content-addressed caching: if the
	// generated Dockerfile + extra-files bytes match what the
	// currently tagged image was built from, the build is a no-op.
	ForceRebuild bool

	// SkipVerify disables post-build verification.
	SkipVerify bool

	// ExtraSkills is an opt-in map of container-path → content for
	// project-local skills (i.e. `skill:<name>` refs that Hydrate
	// strips before the registry sees them). Keys should be absolute
	// container paths beginning with "/world/skills/<name>/SKILL.md".
	// Merged into the image's skills layer alongside tool-shipped
	// skills so Claude Code's native discovery finds both kinds.
	// Nil or empty map means "no local skills".
	ExtraSkills map[string][]byte

	// LogWriter receives build output. If nil, output is discarded.
	LogWriter io.Writer
}

// Validate returns a non-nil error when BuildRequest is missing
// required fields. Called at the top of Build before any
// side-effectful work (Dockerfile gen, docker build, probe).
func (req *BuildRequest) Validate() error {
	if len(req.BaseDockerfile) == 0 {
		return fmt.Errorf("BuildRequest.BaseDockerfile is required")
	}
	if req.Tag == "" {
		return fmt.Errorf("BuildRequest.Tag is required")
	}
	return nil
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
	if err := req.Validate(); err != nil {
		return nil, err
	}
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
			AptPackages:  spec.AptPackages,
			Commands:     spec.Commands,
			UserCommands: spec.UserCommands,
			Env:          spec.Env,
			Files:        spec.Files,
			HasSkills:    t.Skills() != nil,
		}
	}

	// Collect extra files for build context (tool files + skills)
	extraFiles := make(map[string][]byte)

	for _, t := range resolved {
		spec := t.Install()
		for path, content := range spec.Files {
			contextPath := fmt.Sprintf("tools/%s%s", t.Name(), path)
			extraFiles[contextPath] = content
		}
	}

	// Collect and add skills. Tool-shipped skills come from resolved
	// tools' Skills() fs.FS; project-local skills come pre-baked in
	// req.ExtraSkills (caller reads spwn/skills/*.md and passes a
	// path→content map). Merged here so both populate the same
	// /world/skills/ tree the image-build layer COPYs into place.
	skills, err := resolver.CollectSkills(resolved)
	if err != nil {
		return nil, fmt.Errorf("collect skills: %w", err)
	}
	for path, content := range skills {
		contextPath := fmt.Sprintf("skills%s", path)
		extraFiles[contextPath] = content
	}
	for absPath, content := range req.ExtraSkills {
		// Container-side path like "/world/skills/foo/SKILL.md" maps
		// to build-context path "skills/foo/SKILL.md" so the Dockerfile
		// generator's `COPY skills/ /world/skills/` rule delivers it.
		rel := strings.TrimPrefix(absPath, "/world/")
		if rel == absPath {
			// Permissive: if the caller handed us a non-/world path,
			// prefix with "skills/" so it at least lands somewhere
			// predictable. Shouldn't happen with the architect caller.
			rel = "skills/" + strings.TrimPrefix(absPath, "/")
		}
		extraFiles[rel] = content
	}

	// Content-addressed versioning: hash the Dockerfile we'd
	// generate + the extra-files tarball contents. Any change to the
	// base Dockerfile, a tool's install spec, the generator logic,
	// or the selected tool set flows through to a different hash and
	// automatically triggers a rebuild. No manual version bumps.
	//
	// Two-pass generation: first pass without the version label so
	// the LABEL value itself doesn't affect the hash, then regenerate
	// with the hash embedded as the label.
	dfNoLabel := dockerfile.Generate(req.BaseDockerfile, toolInputs, "")
	version := hashBuildContext(dfNoLabel, extraFiles)
	df := dockerfile.Generate(req.BaseDockerfile, toolInputs, version)

	if req.ForceRebuild {
		_ = b.backend.ImageRemove(ctx, req.Tag)
	}

	// Build image
	rebuilt, err := b.backend.EnsureImageWithContext(ctx, req.Tag, version, df, extraFiles, logw)
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
		Cached:     !rebuilt,
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

// hashBuildContext produces a 12-char hex digest of the Dockerfile +
// sorted extra-files contents. Used as the content-addressed image
// version label so every input change invalidates the cache without
// any manual bumping. Order-independent: sorts file names and hashes
// each path:content pair so map iteration order can't affect the
// result.
func hashBuildContext(dockerfile []byte, extraFiles map[string][]byte) string {
	h := sha256.New()
	h.Write([]byte("dockerfile:"))
	h.Write(dockerfile)
	h.Write([]byte{0})

	names := make([]string, 0, len(extraFiles))
	for name := range extraFiles {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		h.Write([]byte("file:" + name + ":"))
		h.Write(extraFiles[name])
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:12]
}

// verify creates a temporary container from the built image and runs each tool's
// verify commands inside it.
func (b *Builder) verify(ctx context.Context, imageTag string, tools []tool.Tool, logw io.Writer) error {
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
