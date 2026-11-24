package architect

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"spwn.sh/packages/container/backend"
	"spwn.sh/packages/world/labels"
	"spwn.sh/packages/world/models"
)

var errStopAfterCreate = errors.New("the spawn stops at its container's start")

type imageStepBackend struct {
	backend.Backend
	mu       sync.Mutex
	held     bool
	entered  chan struct{}
	release  chan struct{}
	ensured  map[string]string
	created  chan createdWorld
	nextID   int
	tagByCfg map[string]string
}

type createdWorld struct {
	config string
	image  string
}

func newImageStepBackend() *imageStepBackend {
	return &imageStepBackend{
		entered:  make(chan struct{}),
		release:  make(chan struct{}),
		ensured:  make(map[string]string),
		created:  make(chan createdWorld, 16),
		tagByCfg: make(map[string]string),
	}
}

func (b *imageStepBackend) EnsureImageWithContext(ctx context.Context, tag, version string, _ []byte, _ map[string][]byte, _ io.Writer) (bool, error) {
	b.mu.Lock()
	b.ensured[tag] = version
	hold := !b.held
	b.held = true
	b.mu.Unlock()
	if hold {
		close(b.entered)
		select {
		case <-b.release:
		case <-ctx.Done():
			return false, ctx.Err()
		}
	}
	return true, nil
}

func (b *imageStepBackend) ImageRemove(context.Context, string) error { return nil }

func (b *imageStepBackend) ImageRemoveUnused(context.Context, string) error { return nil }

func (b *imageStepBackend) ImageList(context.Context, string) ([]backend.ImageInfo, error) {
	return nil, nil
}

func (b *imageStepBackend) Create(_ context.Context, cfg backend.ContainerConfig) (string, error) {
	b.mu.Lock()
	b.nextID++
	id := fmt.Sprintf("image-step-%d", b.nextID)
	b.mu.Unlock()
	b.created <- createdWorld{config: cfg.Labels[labels.WorldConfig], image: cfg.Image}
	return id, nil
}

func (b *imageStepBackend) Start(context.Context, string) error  { return errStopAfterCreate }
func (b *imageStepBackend) Stop(context.Context, string) error   { return nil }
func (b *imageStepBackend) Remove(context.Context, string) error { return nil }

func (b *imageStepBackend) ensuredVersion(tag string) (string, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	v, ok := b.ensured[tag]
	return v, ok
}

func isolateSpawn(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SPWN_HOME", t.TempDir())
	t.Setenv("TMPDIR", t.TempDir())
	t.Setenv("PATH", t.TempDir())
	t.Setenv("SPWN_BASE_IMAGE", "")
	t.Setenv("SPWN_SKIP_AUTH_VALIDATION", "1")
	t.Setenv("SPWN_SKIP_KEYCHAIN", "1")
}

func TestSpawn_AHeldImageBuildHoldsBackNoOtherToolSet(t *testing.T) {
	isolateSpawn(t)
	b := newImageStepBackend()
	arch := New(b, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	spawn := func(config string, deps ...string) chan error {
		done := make(chan error, 1)
		go func() {
			_, err := arch.Spawn(ctx, SpawnOpts{ConfigName: config, Manifest: models.Manifest{Deps: deps}})
			done <- err
		}()
		return done
	}

	slow := spawn("slow", "spwn:docker-cli")
	select {
	case <-b.entered:
	case <-time.After(10 * time.Second):
		t.Fatal("the slow spawn never reached its image build")
	}

	others := map[string][]string{
		"git-a": {"spwn:git"},
		"git-b": {"spwn:git"},
		"gh":    {"spwn:gh"},
	}
	var waits []chan error
	for config, deps := range others {
		waits = append(waits, spawn(config, deps...))
	}

	images := map[string]string{}
	deadline := time.After(10 * time.Second)
	for len(images) < len(others) {
		select {
		case w := <-b.created:
			if w.config == "slow" {
				t.Fatal("the slow spawn created its container while its image build was still held")
			}
			images[w.config] = w.image
		case <-deadline:
			t.Fatalf("while one spawn's image build was held, only %d of %d other spawns reached their container: %v", len(images), len(others), images)
		}
	}
	for _, done := range waits {
		if err := <-done; !errors.Is(err, errStopAfterCreate) {
			t.Fatalf("a spawn stopped before its container's start: %v", err)
		}
	}

	close(b.release)
	select {
	case w := <-b.created:
		images[w.config] = w.image
	case <-time.After(10 * time.Second):
		t.Fatal("the slow spawn never reached its container once its image build was released")
	}
	if err := <-slow; !errors.Is(err, errStopAfterCreate) {
		t.Fatalf("the slow spawn stopped before its container's start: %v", err)
	}

	for config, image := range images {
		repository, version, ok := strings.Cut(image, ":")
		if !ok || repository != "spwn/world" {
			t.Fatalf("%s created its world from %q, not a spwn/world tag", config, image)
		}
		if ensured, ok := b.ensuredVersion(image); !ok || ensured != version {
			t.Fatalf("%s created its world from %q, but that tag was ensured as version %q", config, image, ensured)
		}
	}
	if images["git-a"] != images["git-b"] {
		t.Fatalf("one tool set gave two images: %q and %q", images["git-a"], images["git-b"])
	}
	distinct := map[string]bool{images["slow"]: true, images["git-a"]: true, images["gh"]: true}
	if len(distinct) != 3 {
		t.Fatalf("three tool sets share an image: %v", images)
	}
}
