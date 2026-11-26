package architect

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"spwn.sh/packages/world/models"
)

type swapBeforeCreate struct {
	*imageStepBackend
	swap func()
}

func (b *swapBeforeCreate) EnsureImageWithContext(context.Context, string, string, []byte, map[string][]byte, io.Writer) (bool, error) {
	b.swap()
	return true, nil
}

type startsThenRemoves struct {
	*imageStepBackend
	removed chan string
}

func (b *startsThenRemoves) EnsureImageWithContext(context.Context, string, string, []byte, map[string][]byte, io.Writer) (bool, error) {
	return true, nil
}

func (b *startsThenRemoves) Start(context.Context, string) error { return nil }

func (b *startsThenRemoves) Remove(_ context.Context, id string) error {
	b.removed <- id
	return nil
}

func TestSpawn_ABindSourceSwappedWhileTheContainerStartsRemovesIt(t *testing.T) {
	isolateSpawn(t)
	b := &startsThenRemoves{imageStepBackend: newImageStepBackend(), removed: make(chan string, 1)}
	changed := errors.New("kn changed while starting")
	calls := 0
	recheck := func() error {
		calls++
		if calls == 2 {
			return changed
		}
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := New(b, nil).Spawn(ctx, SpawnOpts{ConfigName: "a", Manifest: models.Manifest{Deps: []string{"spwn:git"}}, Recheck: recheck})
	if !errors.Is(err, changed) {
		t.Fatalf("Spawn = %v, want the recheck's refusal", err)
	}
	created := <-b.created
	select {
	case id := <-b.removed:
		if id == "" {
			t.Fatalf("removed no container (made %+v)", created)
		}
	default:
		t.Fatalf("the container made for %+v was kept after its bind source changed", created)
	}
}

func TestSpawn_ABindSourceSwappedBeforeTheContainerIsMadeRefusesTheWorld(t *testing.T) {
	isolateSpawn(t)
	dir := t.TempDir()
	inside, outside := filepath.Join(dir, "inside"), filepath.Join(dir, "outside")
	for _, d := range []string{inside, outside} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(dir, "kn")
	if err := os.Symlink(inside, link); err != nil {
		t.Fatal(err)
	}
	approved, err := filepath.EvalSymlinks(link)
	if err != nil {
		t.Fatal(err)
	}
	b := &swapBeforeCreate{imageStepBackend: newImageStepBackend(), swap: func() {
		if err := os.Remove(link); err != nil {
			t.Error(err)
		}
		if err := os.Symlink(outside, link); err != nil {
			t.Error(err)
		}
	}}
	changed := errors.New("kn changed while starting")
	recheck := func() error {
		if now, _ := filepath.EvalSymlinks(link); now != approved {
			return changed
		}
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = New(b, nil).Spawn(ctx, SpawnOpts{ConfigName: "a", Manifest: models.Manifest{Deps: []string{"spwn:git"}}, Recheck: recheck})
	if !errors.Is(err, changed) {
		t.Fatalf("Spawn = %v, want the recheck's refusal", err)
	}
	select {
	case w := <-b.created:
		t.Fatalf("a container was made after its bind source changed: %+v", w)
	default:
	}
}
