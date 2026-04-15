package image

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// TreeTarballer is anything that can serialise itself as a tar stream.
// The compile package's *Tree satisfies this interface via its Tar
// method — we keep the contract minimal so packages/image avoids a
// hard dependency on packages/compile (which would cycle back through
// packages/world -> packages/image).
type TreeTarballer interface {
	Tar(w io.Writer) error
}

// BuildFromBaseRequest describes a "derive an image from a base +
// a compiled tree" build.
type BuildFromBaseRequest struct {
	// BaseImage is the existing image to derive from. Must be
	// locally available (pulled or built) — this function does not
	// pull it.
	BaseImage string

	// Tree is the compiled project tree to bake into the image.
	// Its contents land at TreeDestination inside the final image.
	Tree TreeTarballer

	// TreeDestination is the absolute path inside the image where
	// the tree lands. Must start with "/". Default: "/world".
	TreeDestination string

	// Tag is the resulting image tag (e.g. "spwn-myproj:latest").
	// Required.
	Tag string

	// Labels are Docker labels written onto the produced image.
	// Used for test cleanup (sh.spwn.test.run) and project identity
	// (sh.spwn.project).
	Labels map[string]string

	// NoCache disables the Docker build cache.
	NoCache bool

	// LogWriter receives docker build output. If nil, output is
	// discarded.
	LogWriter io.Writer
}

// BuildFromBaseResult is the outcome of a successful BuildFromBase.
type BuildFromBaseResult struct {
	// ImageID is the short sha256: ID of the resulting image.
	ImageID string
	// Tag is the tag that was applied.
	Tag string
}

// DockerImageBuilder is the subset of the Docker API used by
// BuildFromBase. The docker client satisfies it out of the box; tests
// can swap in a fake.
type DockerImageBuilder interface {
	ImageBuild(ctx context.Context, context io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error)
	ImageInspectWithRaw(ctx context.Context, imageID string) (types.ImageInspect, []byte, error)
}

// BuildFromBase derives a new Docker image from an existing base image
// plus a compiled Tree. The generated Dockerfile is:
//
//	FROM <BaseImage>
//	COPY <tree-prefix>/ <TreeDestination>/
//	LABEL <key>=<value> ...
//
// Everything is streamed into docker build as an in-memory tar — no
// disk materialisation required.
func BuildFromBase(ctx context.Context, cli DockerImageBuilder, req BuildFromBaseRequest) (*BuildFromBaseResult, error) {
	if req.BaseImage == "" {
		return nil, fmt.Errorf("BuildFromBase: BaseImage is required")
	}
	if req.Tree == nil {
		return nil, fmt.Errorf("BuildFromBase: Tree is required")
	}
	if req.Tag == "" {
		return nil, fmt.Errorf("BuildFromBase: Tag is required")
	}

	dest := req.TreeDestination
	if dest == "" {
		dest = "/world"
	}
	if !strings.HasPrefix(dest, "/") {
		return nil, fmt.Errorf("BuildFromBase: TreeDestination must be absolute, got %q", dest)
	}

	logw := req.LogWriter
	if logw == nil {
		logw = io.Discard
	}

	// Build the Dockerfile.
	df := generateDockerfile(req.BaseImage, dest, req.Labels)

	// Build the tar context: Dockerfile at root, tree entries under
	// "world/" (an arbitrary stable prefix chosen by us — the
	// Dockerfile references this same prefix).
	var ctxTar bytes.Buffer
	tw := tar.NewWriter(&ctxTar)

	// Dockerfile first so `docker build` finds it quickly.
	if err := writeTarFile(tw, "Dockerfile", df); err != nil {
		return nil, fmt.Errorf("tar Dockerfile: %w", err)
	}

	// Spool the tree into a temporary buffer so we can rewrite each
	// entry's name with the "world/" prefix before emitting it into
	// the build context tar.
	var treeBuf bytes.Buffer
	if err := req.Tree.Tar(&treeBuf); err != nil {
		return nil, fmt.Errorf("serialise tree: %w", err)
	}
	reader := tar.NewReader(&treeBuf)
	for {
		hdr, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read tree tar: %w", err)
		}
		// Skip anything that's not a regular file — trees are flat
		// today, but be defensive.
		if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != tar.TypeRegA {
			continue
		}
		data, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("read tree entry %s: %w", hdr.Name, err)
		}
		if err := writeTarFile(tw, path.Join("world", hdr.Name), data); err != nil {
			return nil, fmt.Errorf("tar tree entry %s: %w", hdr.Name, err)
		}
	}

	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("close build context tar: %w", err)
	}

	fmt.Fprintf(logw, "Building %s from %s...\n", req.Tag, req.BaseImage)

	resp, err := cli.ImageBuild(ctx, &ctxTar, types.ImageBuildOptions{
		Tags:       []string{req.Tag},
		Dockerfile: "Dockerfile",
		Remove:     true,
		NoCache:    req.NoCache,
		Labels:     req.Labels,
	})
	if err != nil {
		return nil, fmt.Errorf("image build: %w", err)
	}
	defer resp.Body.Close()

	// Stream build output — Docker returns JSON lines with a
	// "stream" field. We also watch for "error" which terminates
	// the build.
	type buildMsg struct {
		Stream string `json:"stream"`
		Error  string `json:"error"`
	}
	decoder := json.NewDecoder(resp.Body)
	for decoder.More() {
		var msg buildMsg
		if err := decoder.Decode(&msg); err != nil {
			break
		}
		if msg.Error != "" {
			return nil, fmt.Errorf("image build: %s", msg.Error)
		}
		if msg.Stream != "" {
			fmt.Fprint(logw, msg.Stream)
		}
	}

	// Inspect for the image ID.
	info, _, err := cli.ImageInspectWithRaw(ctx, req.Tag)
	if err != nil {
		// A missing image after a clean build is unusual but we
		// don't want to fail the whole command over a display
		// detail — return the tag without an ID.
		if client.IsErrNotFound(err) {
			return &BuildFromBaseResult{Tag: req.Tag}, nil
		}
		return nil, fmt.Errorf("inspect built image: %w", err)
	}

	return &BuildFromBaseResult{
		ImageID: info.ID,
		Tag:     req.Tag,
	}, nil
}

// generateDockerfile renders the tiny derived-image Dockerfile.
func generateDockerfile(baseImage, dest string, labels map[string]string) []byte {
	var b strings.Builder
	b.WriteString("# syntax=docker/dockerfile:1\n")
	fmt.Fprintf(&b, "FROM %s\n", baseImage)
	// Copy the tree entries under "world/" (our build context
	// prefix) into the requested destination. Docker normalises
	// trailing slashes.
	fmt.Fprintf(&b, "COPY world/ %s/\n", strings.TrimRight(dest, "/"))

	// Sort labels for deterministic output.
	if len(labels) > 0 {
		keys := make([]string, 0, len(labels))
		for k := range labels {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(&b, "LABEL %s=%q\n", k, labels[k])
		}
	}
	return []byte(b.String())
}

// writeTarFile appends a single regular-file entry to tw with fixed
// mode / modtime / uid / gid so the resulting tar is reproducible.
func writeTarFile(tw *tar.Writer, name string, data []byte) error {
	hdr := &tar.Header{
		Name:    name,
		Mode:    0o644,
		Size:    int64(len(data)),
		ModTime: time.Unix(0, 0).UTC(),
		Uid:     0,
		Gid:     0,
		Format:  tar.FormatUSTAR,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err := tw.Write(data)
	return err
}
