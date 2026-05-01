package compile

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
)

type fakeImageBuilder struct {
	buildErr   error
	buildBody  string
	contextTar []byte
	options    types.ImageBuildOptions
	inspectErr error
}

func (f *fakeImageBuilder) ImageBuild(ctx context.Context, context io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
	data, err := io.ReadAll(context)
	if err != nil {
		return types.ImageBuildResponse{}, err
	}
	f.contextTar = data
	f.options = options
	if f.buildErr != nil {
		return types.ImageBuildResponse{}, f.buildErr
	}
	body := f.buildBody
	if body == "" {
		body = `{"stream":"step ok\n"}`
	}
	return types.ImageBuildResponse{Body: io.NopCloser(strings.NewReader(body))}, nil
}

func (f *fakeImageBuilder) ImageInspectWithRaw(ctx context.Context, imageID string) (types.ImageInspect, []byte, error) {
	if f.inspectErr != nil {
		return types.ImageInspect{}, nil, f.inspectErr
	}
	return types.ImageInspect{ID: "sha256:abc123"}, nil, nil
}

type tarTree map[string]string

func (t tarTree) Tar(w io.Writer) error {
	tw := tar.NewWriter(w)
	for name, body := range t {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			return err
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			return err
		}
	}
	return tw.Close()
}

func readTarEntries(t *testing.T, data []byte) map[string]string {
	t.Helper()
	out := map[string]string{}
	tr := tar.NewReader(bytes.NewReader(data))
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("read context tar: %v", err)
		}
		body, err := io.ReadAll(tr)
		if err != nil {
			t.Fatalf("read %s: %v", hdr.Name, err)
		}
		out[hdr.Name] = string(body)
	}
	return out
}

func TestBuildFromBase_StreamsTreeUnderWorldPrefixAndPassesDockerOptions(t *testing.T) {
	fake := &fakeImageBuilder{}
	var logs bytes.Buffer

	result, err := BuildFromBase(context.Background(), fake, BuildFromBaseRequest{
		BaseImage:       "spwn-test:latest",
		Tree:            tarTree{"agents/neo/AGENTS.md": "# neo"},
		TreeDestination: "/world",
		Tag:             "spwn-codex-pilot:test",
		Labels: map[string]string{
			"sh.spwn.project": "codex-pilot",
			"sh.spwn.runtime": "codex",
		},
		NoCache:   true,
		LogWriter: &logs,
	})
	if err != nil {
		t.Fatalf("BuildFromBase: %v", err)
	}
	if result.Tag != "spwn-codex-pilot:test" || result.ImageID != "sha256:abc123" {
		t.Fatalf("unexpected result: %+v", result)
	}

	if got := fake.options.Tags; len(got) != 1 || got[0] != "spwn-codex-pilot:test" {
		t.Fatalf("Tags = %v", got)
	}
	if fake.options.Dockerfile != "Dockerfile" {
		t.Fatalf("Dockerfile = %q", fake.options.Dockerfile)
	}
	if !fake.options.Remove || !fake.options.NoCache {
		t.Fatalf("Remove/NoCache not forwarded: %+v", fake.options)
	}
	if fake.options.Labels["sh.spwn.runtime"] != "codex" {
		t.Fatalf("Labels not forwarded: %+v", fake.options.Labels)
	}

	entries := readTarEntries(t, fake.contextTar)
	if entries["world/agents/neo/AGENTS.md"] != "# neo" {
		t.Fatalf("tree file missing from context: %#v", entries)
	}
	dockerfile := entries["Dockerfile"]
	for _, want := range []string{
		"FROM spwn-test:latest",
		"COPY world/ /world/",
		`LABEL sh.spwn.project="codex-pilot"`,
		`LABEL sh.spwn.runtime="codex"`,
	} {
		if !strings.Contains(dockerfile, want) {
			t.Fatalf("Dockerfile missing %q:\n%s", want, dockerfile)
		}
	}
	if !strings.Contains(logs.String(), "Building spwn-codex-pilot:test from spwn-test:latest") {
		t.Fatalf("logs missing build banner: %s", logs.String())
	}
	if !strings.Contains(logs.String(), "step ok") {
		t.Fatalf("logs missing docker stream: %s", logs.String())
	}
}

func TestBuildFromBase_ValidatesRequiredInputs(t *testing.T) {
	validTree := tarTree{"x": "y"}
	tests := []struct {
		name string
		req  BuildFromBaseRequest
		want string
	}{
		{
			name: "base image",
			req:  BuildFromBaseRequest{Tree: validTree, Tag: "tag"},
			want: "BaseImage is required",
		},
		{
			name: "tree",
			req:  BuildFromBaseRequest{BaseImage: "base", Tag: "tag"},
			want: "Tree is required",
		},
		{
			name: "tag",
			req:  BuildFromBaseRequest{BaseImage: "base", Tree: validTree},
			want: "Tag is required",
		},
		{
			name: "destination",
			req:  BuildFromBaseRequest{BaseImage: "base", Tree: validTree, Tag: "tag", TreeDestination: "relative"},
			want: "TreeDestination must be absolute",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BuildFromBase(context.Background(), &fakeImageBuilder{}, tt.req)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestBuildFromBase_ReturnsDockerErrorEnvelope(t *testing.T) {
	_, err := BuildFromBase(context.Background(), &fakeImageBuilder{buildBody: `{"error":"boom"}`}, BuildFromBaseRequest{
		BaseImage: "base",
		Tree:      tarTree{"x": "y"},
		Tag:       "tag",
	})
	if err == nil || !strings.Contains(err.Error(), "image build: boom") {
		t.Fatalf("err = %v", err)
	}
}
