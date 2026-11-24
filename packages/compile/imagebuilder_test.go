package compile

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	cerrdefs "github.com/containerd/errdefs"

	"spwn.sh/packages/compile/baseimages"
	"spwn.sh/packages/container/backend"
	"spwn.sh/packages/dependency"
	"spwn.sh/packages/dependency/resolver"
)

type imageStore struct {
	backend.Backend
	mu       sync.Mutex
	tags     map[string]bool
	inUse    map[string]bool
	removals []string
	building chan string
	resume   chan struct{}
}

func newImageStore() *imageStore {
	return &imageStore{tags: map[string]bool{}, inUse: map[string]bool{}}
}

func (s *imageStore) has(tag string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tags[tag]
}

func (s *imageStore) build(tag string) {
	if s.building != nil {
		s.building <- tag
		<-s.resume
	}
	s.mu.Lock()
	s.tags[tag] = true
	s.mu.Unlock()
}

func (s *imageStore) EnsureImageWithContext(_ context.Context, tag, _ string, _ []byte, _ map[string][]byte, _ io.Writer) (bool, error) {
	if s.has(tag) {
		return false, nil
	}
	s.build(tag)
	return true, nil
}

func (s *imageStore) RebuildImage(_ context.Context, tag string, _ []byte, _ map[string][]byte, _ io.Writer) error {
	s.build(tag)
	return nil
}

func (s *imageStore) ImageRemove(_ context.Context, tag string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tags, tag)
	return nil
}

func (s *imageStore) ImageRemoveUnused(_ context.Context, tag string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removals = append(s.removals, tag)
	if s.inUse[tag] {
		return fmt.Errorf("image %s is used by a container: %w", tag, cerrdefs.ErrConflict)
	}
	delete(s.tags, tag)
	return nil
}

func (s *imageStore) ImageList(_ context.Context, prefix string) ([]backend.ImageInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []backend.ImageInfo
	for tag := range s.tags {
		if strings.HasPrefix(tag, prefix) {
			out = append(out, backend.ImageInfo{Tag: tag})
		}
	}
	return out, nil
}

func (s *imageStore) removed() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.removals...)
}

func newBuilder(t *testing.T, store *imageStore) *Builder {
	t.Helper()
	t.Setenv("TMPDIR", t.TempDir())
	reg := resolver.NewRegistry()
	if err := dependency.RegisterBuiltins(reg); err != nil {
		t.Fatalf("register tools: %v", err)
	}
	return New(reg, store)
}

func buildOf(t *testing.T, b *Builder, force bool, log io.Writer, tools ...string) *BuildResult {
	t.Helper()
	result, err := b.Build(context.Background(), BuildRequest{
		BaseDockerfile: baseimages.WorldDockerfile,
		Tools:          append([]string{"spwn:unix"}, tools...),
		Repository:     "spwn/world",
		ForceRebuild:   force,
		SkipVerify:     true,
		LogWriter:      log,
	})
	if err != nil {
		t.Fatalf("build %v: %v", tools, err)
	}
	return result
}

func TestBuild_AForcedRebuildKeepsTheTagForASiblingStillToCreate(t *testing.T) {
	store := newImageStore()
	b := newBuilder(t, store)
	sibling := buildOf(t, b, false, nil, "spwn:git")
	defer sibling.Release()

	store.building = make(chan string)
	store.resume = make(chan struct{})
	done := make(chan *BuildResult, 1)
	failed := make(chan error, 1)
	go func() {
		result, err := b.Build(context.Background(), BuildRequest{
			BaseDockerfile: baseimages.WorldDockerfile,
			Tools:          []string{"spwn:unix", "spwn:git"},
			Repository:     "spwn/world",
			ForceRebuild:   true,
			SkipVerify:     true,
		})
		if err != nil {
			failed <- err
			return
		}
		done <- result
	}()

	select {
	case tag := <-store.building:
		if tag != sibling.Tag {
			t.Fatalf("the forced rebuild built %q, not the sibling's %q", tag, sibling.Tag)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the forced rebuild never built")
	}
	if !store.has(sibling.Tag) {
		t.Fatalf("while the forced rebuild ran, %s was gone, so the sibling's create would find no image", sibling.Tag)
	}
	close(store.resume)
	var forced *BuildResult
	select {
	case forced = <-done:
	case err := <-failed:
		t.Fatalf("the forced rebuild failed: %v", err)
	}
	forced.Release()
	if forced.Cached || !store.has(sibling.Tag) {
		t.Fatalf("the forced rebuild left cached=%v, tag present=%v", forced.Cached, store.has(sibling.Tag))
	}
}

func TestBuild_ANewTagPrunesOnlyTheTagsNothingUses(t *testing.T) {
	store := newImageStore()
	b := newBuilder(t, store)

	inFlight := buildOf(t, b, false, nil, "spwn:node")
	used := buildOf(t, b, false, nil, "spwn:git")
	used.Release()
	store.inUse[used.Tag] = true
	unused := buildOf(t, b, false, nil, "spwn:docker-cli")
	unused.Release()

	var log bytes.Buffer
	fresh := buildOf(t, b, false, &log, "spwn:python")
	defer fresh.Release()
	if fresh.Cached || len(map[string]bool{used.Tag: true, unused.Tag: true, inFlight.Tag: true, fresh.Tag: true}) != 4 {
		t.Fatalf("the four builds are not four new tags: %s %s %s %s cached=%v", used.Tag, unused.Tag, inFlight.Tag, fresh.Tag, fresh.Cached)
	}

	if store.has(unused.Tag) {
		t.Fatalf("%s, used by nothing, was kept", unused.Tag)
	}
	if !store.has(used.Tag) {
		t.Fatalf("%s, used by a container, was removed", used.Tag)
	}
	if !store.has(inFlight.Tag) {
		t.Fatalf("%s, built by a spawn that has not created its container yet, was removed", inFlight.Tag)
	}
	for _, tag := range store.removed() {
		if tag == inFlight.Tag || tag == fresh.Tag {
			t.Fatalf("a removal was attempted on %s: %v", tag, store.removed())
		}
	}
	if !strings.Contains(log.String(), "Pruned "+unused.Tag+"\n") {
		t.Fatalf("the prune is not in the build log: %q", log.String())
	}

	inFlight.Release()
	again := buildOf(t, b, false, nil, "spwn:python", "spwn:docker-cli")
	again.Release()
	if store.has(inFlight.Tag) {
		t.Fatalf("%s, released by its spawn, survived the next new tag", inFlight.Tag)
	}
	if !store.has(used.Tag) {
		t.Fatalf("%s, used by a container, was removed", used.Tag)
	}
}

func TestBuild_ACachedTagPrunesNothing(t *testing.T) {
	store := newImageStore()
	b := newBuilder(t, store)
	other := buildOf(t, b, false, nil, "spwn:gh")
	other.Release()
	first := buildOf(t, b, false, nil, "spwn:git")
	first.Release()
	store.tags[other.Tag] = true

	cached := buildOf(t, b, false, nil, "spwn:git")
	cached.Release()
	if !cached.Cached || !store.has(other.Tag) {
		t.Fatalf("a cache hit pruned: cached=%v, %s present=%v", cached.Cached, other.Tag, store.has(other.Tag))
	}
}
