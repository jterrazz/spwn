package state

import (
	"context"
	"io"
	"testing"
	"time"

	"spwn.sh/packages/world/internal/backend"
	"spwn.sh/packages/world/internal/labels"
	"spwn.sh/packages/world/models"
	"spwn.sh/packages/world/internal/runtimestate"
)

// fakeBackend implements backend.Backend with stub methods for
// everything the state Store does not need. Its container set is
// mutable so tests can simulate "user ran docker rm" between calls.
type fakeBackend struct {
	containers []backend.ContainerInfo
}

func (f *fakeBackend) ListContainersByLabel(_ context.Context, key, value string) ([]backend.ContainerInfo, error) {
	out := []backend.ContainerInfo{}
	for _, c := range f.containers {
		if c.Labels[key] == value || (value == "" && c.Labels[key] != "") {
			out = append(out, c)
		}
	}
	return out, nil
}

// Stubs - none of these are called by Store, but we need them to
// satisfy the Backend interface.
func (f *fakeBackend) Create(context.Context, backend.ContainerConfig) (string, error) {
	return "", nil
}
func (f *fakeBackend) Start(context.Context, string) error  { return nil }
func (f *fakeBackend) Stop(context.Context, string) error   { return nil }
func (f *fakeBackend) Remove(context.Context, string) error { return nil }
func (f *fakeBackend) Exec(context.Context, string, backend.ExecConfig) (int, error) {
	return 0, nil
}
func (f *fakeBackend) ExecOutput(context.Context, string, []string) (string, error) {
	return "", nil
}
func (f *fakeBackend) CopyTo(context.Context, string, string, []byte) error  { return nil }
func (f *fakeBackend) CopyDirTo(context.Context, string, string, string) error {
	return nil
}
func (f *fakeBackend) CopyDirFrom(context.Context, string, string, string) error {
	return nil
}
func (f *fakeBackend) IsRunning(context.Context, string) (bool, error)       { return false, nil }
func (f *fakeBackend) ImageExists(context.Context, string) (bool, error)     { return false, nil }
func (f *fakeBackend) EnsureImage(context.Context, string, string, []byte, io.Writer) (bool, error) {
	return true, nil
}
func (f *fakeBackend) EnsureImageWithContext(context.Context, string, string, []byte, map[string][]byte, io.Writer) (bool, error) {
	return true, nil
}
func (f *fakeBackend) ImageVersion(context.Context, string, string) (string, error) {
	return "", nil
}
func (f *fakeBackend) ExecDetached(context.Context, string, backend.ExecConfig) error {
	return nil
}
func (f *fakeBackend) Commit(context.Context, string, string) error           { return nil }
func (f *fakeBackend) ImageList(context.Context, string) ([]backend.ImageInfo, error) {
	return nil, nil
}
func (f *fakeBackend) ImageRemove(context.Context, string) error { return nil }
func (f *fakeBackend) Inspect(context.Context, string) (*backend.ContainerInfo, error) {
	return nil, nil
}
func newWorldContainer(id, name, configName string, running bool) backend.ContainerInfo {
	w := models.World{
		ID:        id,
		Name:      name,
		Config:    configName,
		CreatedAt: time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
		Agents: []models.AgentRecord{
			{Name: "neo", AgentID: "a-neo-1", Role: "worker", Status: models.StatusIdle},
		},
	}
	return backend.ContainerInfo{
		ID:      "container-" + id,
		Name:    id,
		Image:   "spwn/world:latest",
		Status:  pickStatus(running),
		Running: running,
		Labels:  labels.WorldLabels(w),
	}
}

func pickStatus(running bool) string {
	if running {
		return "running"
	}
	return "exited"
}

func newStore(t *testing.T, fb *fakeBackend) *Store {
	t.Helper()
	rs, err := runtimestate.NewStoreAt(t.TempDir())
	if err != nil {
		t.Fatalf("runtimestate: %v", err)
	}
	return NewStoreWith(fb, rs)
}

func TestList_ReadsFromContainerLabels(t *testing.T) {
	fb := &fakeBackend{containers: []backend.ContainerInfo{
		newWorldContainer("w-default-11111", "", "default", true),
		newWorldContainer("w-acme-22222", "Acme", "acme", false),
	}}
	s := newStore(t, fb)

	worlds, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(worlds) != 2 {
		t.Fatalf("expected 2 worlds, got %d", len(worlds))
	}

	byID := map[string]models.World{}
	for _, w := range worlds {
		byID[w.ID] = w
	}
	if byID["w-default-11111"].Status != models.StatusRunning {
		t.Errorf("running world should map to StatusRunning")
	}
	if byID["w-acme-22222"].Status != models.StatusStopped {
		t.Errorf("exited world should map to StatusStopped")
	}
	if byID["w-acme-22222"].Name != "Acme" {
		t.Errorf("name should round-trip from labels")
	}
}

func TestList_GhostContainersDisappearWhenRemoved(t *testing.T) {
	// This is the bug we are explicitly fixing: if Docker no longer
	// has the container, the world must not be visible - full stop.
	fb := &fakeBackend{containers: []backend.ContainerInfo{
		newWorldContainer("w-doomed-99999", "", "default", true),
	}}
	s := newStore(t, fb)

	if worlds, _ := s.List(); len(worlds) != 1 {
		t.Fatalf("expected 1 world before removal, got %d", len(worlds))
	}

	// Simulate `docker rm`.
	fb.containers = nil

	worlds, err := s.List()
	if err != nil {
		t.Fatalf("List after removal: %v", err)
	}
	if len(worlds) != 0 {
		t.Fatalf("expected 0 worlds after removal, got %d", len(worlds))
	}
}

func TestGet_NotFound(t *testing.T) {
	s := newStore(t, &fakeBackend{})
	if _, err := s.Get("w-missing"); err == nil {
		t.Fatal("expected error for missing world")
	}
}

func TestSessionID_HydratedIntoListResults(t *testing.T) {
	fb := &fakeBackend{containers: []backend.ContainerInfo{
		newWorldContainer("w-1", "", "default", true),
	}}
	s := newStore(t, fb)

	if err := s.SetSessionID("w-1", "neo", "sess-abc"); err != nil {
		t.Fatalf("SetSessionID: %v", err)
	}

	worlds, _ := s.List()
	if len(worlds) != 1 {
		t.Fatalf("expected 1 world, got %d", len(worlds))
	}
	if got := worlds[0].SessionIDs["neo"]; got != "sess-abc" {
		t.Errorf("session id not hydrated into List() result: %q", got)
	}
}

func TestList_GCsOrphanedRuntimeFiles(t *testing.T) {
	fb := &fakeBackend{containers: []backend.ContainerInfo{
		newWorldContainer("w-live", "", "default", true),
	}}
	s := newStore(t, fb)

	// Pretend a previous spawn left runtime state for a world that's
	// since been removed.
	if err := s.rstate.SetSessionID("w-orphan", "neo", "old"); err != nil {
		t.Fatalf("seed orphan: %v", err)
	}
	if got := s.rstate.GetSessionID("w-orphan", "neo"); got != "old" {
		t.Fatalf("orphan seed broken: %q", got)
	}

	if _, err := s.List(); err != nil {
		t.Fatalf("List: %v", err)
	}

	// GC should have wiped the orphaned file.
	if got := s.rstate.GetSessionID("w-orphan", "neo"); got != "" {
		t.Errorf("orphaned runtime file not GC'd: %q", got)
	}
}

func TestAddAgent_RequiresLiveWorld(t *testing.T) {
	s := newStore(t, &fakeBackend{})
	err := s.AddAgent("w-missing", models.AgentRecord{Name: "neo", AgentID: "a-1"})
	if err == nil {
		t.Fatal("AddAgent should reject missing worlds")
	}
}

func TestSaveDeleteUpdateStatus_AreNoOps(t *testing.T) {
	s := newStore(t, &fakeBackend{})
	// All three should return nil for any input - they exist only for
	// API stability.
	if err := s.Save(models.World{}); err != nil {
		t.Errorf("Save should be a no-op: %v", err)
	}
	if err := s.UpdateStatus("anything", models.StatusRunning); err != nil {
		t.Errorf("UpdateStatus should be a no-op: %v", err)
	}
	if err := s.Delete("nope"); err != nil {
		t.Errorf("Delete on missing world should be a no-op: %v", err)
	}
}
