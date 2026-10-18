package compile

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"strings"

	"spwn.sh/packages/compile/internal/dockerfile"
	"spwn.sh/packages/container/backend"
	"spwn.sh/packages/dependency/resolver"
	"spwn.sh/packages/dependency/tool"
)

// Builder composes Docker images from tool definitions.
type Builder struct {
	registry *resolver.Registry
	backend  backend.Backend
}

// ToolPolicy mirrors agent.DepPolicy at the compile layer so the
// compile package doesn't import agent (would cycle).
type ToolPolicy struct {
	Allow []string
	Deny  []string
}

func shortNameFromRef(ref string) string {
	if i := strings.Index(ref, ":"); i >= 0 {
		return ref[i+1:]
	}
	return ref
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

	// Policies is the per-tool allow/deny filter inferred from the
	// agent's manifest. Keyed by the tool ref (e.g. "spwn:x"). Tools
	// without an entry are unfiltered. Materialized at image build
	// time as /etc/spwn/policy/<short>.json; catalog tool wrappers
	// consult that file at runtime.
	Policies map[string]ToolPolicy

	// Tag is the Docker image tag to apply.
	Tag string

	// ForceRebuild bypasses the content hash cache check and rebuilds
	// from scratch. The default is content-addressed caching: if the
	// generated Dockerfile + extra-files bytes match what the
	// currently tagged image was built from, the build is a no-op.
	ForceRebuild bool

	// SkipVerify disables post-build verification.
	SkipVerify bool

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
	Tag    string
	Tools  []string
	Cached bool
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

	resolved, err := b.registry.Resolve(req.Tools)
	if err != nil {
		return nil, fmt.Errorf("resolve tools: %w", err)
	}

	resolvedNames := make([]string, len(resolved))
	for i, t := range resolved {
		resolvedNames[i] = t.Name()
	}

	fmt.Fprintf(logw, "Resolved tools: %s\n", strings.Join(resolvedNames, ", "))

	toolInputs := make([]dockerfile.ToolInput, len(resolved))
	for i, t := range resolved {
		spec := t.Install()
		ti := dockerfile.ToolInput{
			Name:     t.Name(),
			Packages: spec.Packages,
			Commands: spec.Commands,
			Env:      spec.Env,
			Files:    spec.Files,
		}
		if pol, ok := req.Policies[t.Name()]; ok && (len(pol.Allow) > 0 || len(pol.Deny) > 0) {
			ti.Policy = &dockerfile.Policy{
				Short: shortNameFromRef(t.Name()),
				Allow: pol.Allow,
				Deny:  pol.Deny,
			}
		}
		toolInputs[i] = ti
	}

	extraFiles := make(map[string][]byte)

	for _, t := range resolved {
		spec := t.Install()
		for path, content := range spec.Files {
			contextPath := fmt.Sprintf("tools/%s%s", t.Name(), path)
			extraFiles[contextPath] = content
		}
	}

	dfNoLabel := dockerfile.Generate(req.BaseDockerfile, toolInputs, "")
	version := hashBuildContext(dfNoLabel, extraFiles)
	df := dockerfile.Generate(req.BaseDockerfile, toolInputs, version)

	if req.ForceRebuild {
		_ = b.backend.ImageRemove(ctx, req.Tag)
	}

	rebuilt, err := b.backend.EnsureImageWithContext(ctx, req.Tag, version, df, extraFiles, logw)
	if err != nil {
		return nil, &BuildError{Tag: req.Tag, Cause: err}
	}

	if !req.SkipVerify {
		fmt.Fprintf(logw, "Verifying tools...\n")
		if err := b.verify(ctx, req.Tag, resolved, logw); err != nil {
			return nil, err
		}
	}

	result := &BuildResult{
		Tag:    req.Tag,
		Tools:  resolvedNames,
		Cached: !rebuilt,
	}

	return result, nil
}

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
