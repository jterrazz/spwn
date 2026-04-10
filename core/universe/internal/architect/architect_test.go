package architect

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"spwn.sh/core/universe/internal/backend"
	"spwn.sh/core/universe/internal/labels"
	"spwn.sh/core/universe/internal/models"
	"spwn.sh/core/universe/internal/runtimestate"
	"spwn.sh/core/universe/internal/state"
)

// mockBackend implements backend.Backend for unit testing without Docker.
type mockBackend struct {
	containers map[string]*mockContainer
	images     map[string]bool
	nextID     int
	// Error injection hooks
	createErr     error
	startErr      error
	stopErr       error
	removeErr     error
	execErr       error
	execOutput    string
	execOutputErr error
	copyToErr     error
	isRunningVal  bool
	isRunningErr  error
	imageExistsV  bool
	imageExistsE  error
	ensureImgErr  error
	logsReader    io.ReadCloser
	logsErr       error
	commitErr     error
	imageList     []backend.ImageInfo
	imageListErr  error
	imageRemoveE  error
	execDetachErr error

	// Call tracking
	createdConfigs []backend.ContainerConfig
	startedIDs     []string
	stoppedIDs     []string
	removedIDs     []string
	execCalls      []execCall
}

type mockContainer struct {
	id      string
	config  backend.ContainerConfig
	running bool
}

type execCall struct {
	containerID string
	cfg         backend.ExecConfig
}

func newMockBackend() *mockBackend {
	return &mockBackend{
		containers:   make(map[string]*mockContainer),
		images:       map[string]bool{"spwn/world:latest": true},
		isRunningVal: true,
		imageExistsV: true,
		execOutput:   "bash\nsh\ngit",
	}
}

func (m *mockBackend) Create(_ context.Context, cfg backend.ContainerConfig) (string, error) {
	if m.createErr != nil {
		return "", m.createErr
	}
	m.nextID++
	id := fmt.Sprintf("mock-%d", m.nextID)
	m.containers[id] = &mockContainer{id: id, config: cfg}
	m.createdConfigs = append(m.createdConfigs, cfg)
	return id, nil
}

func (m *mockBackend) Start(_ context.Context, containerID string) error {
	m.startedIDs = append(m.startedIDs, containerID)
	if m.startErr != nil {
		return m.startErr
	}
	if c, ok := m.containers[containerID]; ok {
		c.running = true
	}
	return nil
}

func (m *mockBackend) Stop(_ context.Context, containerID string) error {
	m.stoppedIDs = append(m.stoppedIDs, containerID)
	if m.stopErr != nil {
		return m.stopErr
	}
	if c, ok := m.containers[containerID]; ok {
		c.running = false
	}
	return nil
}

func (m *mockBackend) Remove(_ context.Context, containerID string) error {
	m.removedIDs = append(m.removedIDs, containerID)
	if m.removeErr != nil {
		return m.removeErr
	}
	delete(m.containers, containerID)
	return nil
}

func (m *mockBackend) Exec(_ context.Context, containerID string, cfg backend.ExecConfig) (int, error) {
	m.execCalls = append(m.execCalls, execCall{containerID, cfg})
	if m.execErr != nil {
		return 1, m.execErr
	}
	return 0, nil
}

func (m *mockBackend) ExecOutput(_ context.Context, _ string, _ []string) (string, error) {
	if m.execOutputErr != nil {
		return "", m.execOutputErr
	}
	return m.execOutput, nil
}

func (m *mockBackend) CopyTo(_ context.Context, _ string, _ string, _ []byte) error {
	return m.copyToErr
}

func (m *mockBackend) IsRunning(_ context.Context, _ string) (bool, error) {
	return m.isRunningVal, m.isRunningErr
}

func (m *mockBackend) ImageExists(_ context.Context, image string) (bool, error) {
	if m.imageExistsE != nil {
		return false, m.imageExistsE
	}
	return m.imageExistsV, nil
}

func (m *mockBackend) EnsureImage(_ context.Context, _ string, _ string, _ []byte, _ io.Writer) error {
	return m.ensureImgErr
}

func (m *mockBackend) EnsureImageWithContext(_ context.Context, _ string, _ string, _ []byte, _ map[string][]byte, _ io.Writer) error {
	return m.ensureImgErr
}

func (m *mockBackend) ImageVersion(_ context.Context, image string, label string) (string, error) {
	return "", nil
}

func (m *mockBackend) Logs(_ context.Context, _ string, _ backend.LogsConfig) (io.ReadCloser, error) {
	if m.logsErr != nil {
		return nil, m.logsErr
	}
	return m.logsReader, nil
}

func (m *mockBackend) ExecDetached(_ context.Context, _ string, _ backend.ExecConfig) error {
	return m.execDetachErr
}

func (m *mockBackend) Commit(_ context.Context, _ string, _ string) error {
	return m.commitErr
}

func (m *mockBackend) ImageList(_ context.Context, _ string) ([]backend.ImageInfo, error) {
	return m.imageList, m.imageListErr
}

func (m *mockBackend) ImageRemove(_ context.Context, _ string) error {
	return m.imageRemoveE
}

func (m *mockBackend) Inspect(_ context.Context, nameOrID string) (*backend.ContainerInfo, error) {
	return &backend.ContainerInfo{ID: nameOrID, Running: true}, nil
}

func (m *mockBackend) ListContainersByLabel(_ context.Context, key, value string) ([]backend.ContainerInfo, error) {
	out := []backend.ContainerInfo{}
	for id, c := range m.containers {
		if c.config.Labels[key] != value {
			continue
		}
		status := "exited"
		if c.running {
			status = "running"
		}
		out = append(out, backend.ContainerInfo{
			ID:      id,
			Name:    c.config.Name,
			Image:   c.config.Image,
			Status:  status,
			Running: c.running,
			Labels:  c.config.Labels,
		})
	}
	return out, nil
}

// --- Tests ---

func newTestArchitect(t *testing.T, b *mockBackend) (*Architect, *state.Store) {
	t.Helper()
	rs, err := runtimestate.NewStoreAt(t.TempDir())
	if err != nil {
		t.Fatalf("runtimestate: %v", err)
	}
	store := state.NewStoreWith(b, rs)
	arch := New(b, store)
	return arch, store
}

// seedWorld registers a world with the mock backend as if it had been
// created via Spawn. With the labels-as-truth architecture, the only
// way to make a world "exist" for the state Store is to put a labeled
// container in the backend. The container ID matches w.ContainerID
// when supplied so destroy/snapshot tests can assert against it.
func seedWorld(mb *mockBackend, w models.World) {
	id := w.ContainerID
	if id == "" {
		mb.nextID++
		id = fmt.Sprintf("mock-%d", mb.nextID)
	}
	cfg := backend.ContainerConfig{
		Name:   w.ID,
		Image:  "spwn/world:latest",
		Labels: labels.WorldLabels(w),
	}
	mb.containers[id] = &mockContainer{id: id, config: cfg, running: true}
}

func TestList_Empty(t *testing.T) {
	mb := newMockBackend()
	arch, _ := newTestArchitect(t, mb)

	worlds, err := arch.List(context.Background())
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(worlds) != 0 {
		t.Errorf("expected 0 worlds, got %d", len(worlds))
	}
}

func TestList_AfterSave(t *testing.T) {
	mb := newMockBackend()
	arch, store := newTestArchitect(t, mb)

	// Save a world directly to state
	w := models.World{
		ID:          "w-test-12345",
		Config:      "test",
		ContainerID: "mock-1",
		Status:      models.StatusIdle,
	}
	seedWorld(mb, w)
	_ = store // store kept in scope for later assertions if needed

	worlds, err := arch.List(context.Background())
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(worlds) != 1 {
		t.Fatalf("expected 1 world, got %d", len(worlds))
	}
	if worlds[0].ID != "w-test-12345" {
		t.Errorf("expected ID w-test-12345, got %s", worlds[0].ID)
	}
}

func TestInspect_Found(t *testing.T) {
	mb := newMockBackend()
	arch, _ := newTestArchitect(t, mb)

	w := models.World{
		ID:          "w-test-99999",
		Config:      "inspect-test",
		ContainerID: "mock-42",
		Status:      models.StatusRunning,
	}
	seedWorld(mb, w)

	got, err := arch.Inspect(context.Background(), "w-test-99999")
	if err != nil {
		t.Fatalf("Inspect() error: %v", err)
	}
	if got.Config != "inspect-test" {
		t.Errorf("expected config 'inspect-test', got %q", got.Config)
	}
}

func TestInspect_NotFound(t *testing.T) {
	mb := newMockBackend()
	arch, _ := newTestArchitect(t, mb)

	_, err := arch.Inspect(context.Background(), "w-nonexistent-00000")
	if err == nil {
		t.Fatal("expected error for nonexistent world")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}

func TestDestroy_RemovesWorld(t *testing.T) {
	mb := newMockBackend()
	arch, store := newTestArchitect(t, mb)

	w := models.World{
		ID:          "w-destroy-11111",
		Config:      "destroy-test",
		ContainerID: "mock-container-1",
		Status:      models.StatusIdle,
	}
	seedWorld(mb, w)

	destroyed, err := arch.Destroy(context.Background(), "w-destroy-11111")
	if err != nil {
		t.Fatalf("Destroy() error: %v", err)
	}
	if destroyed.ID != "w-destroy-11111" {
		t.Errorf("expected destroyed world ID w-destroy-11111, got %s", destroyed.ID)
	}

	// Verify container was stopped and removed
	if len(mb.stoppedIDs) != 1 || mb.stoppedIDs[0] != "mock-container-1" {
		t.Errorf("expected stop on mock-container-1, got %v", mb.stoppedIDs)
	}
	if len(mb.removedIDs) != 1 || mb.removedIDs[0] != "mock-container-1" {
		t.Errorf("expected remove on mock-container-1, got %v", mb.removedIDs)
	}

	// Verify world is gone from state
	worlds, _ := store.List()
	if len(worlds) != 0 {
		t.Errorf("expected 0 worlds after destroy, got %d", len(worlds))
	}
}

func TestDestroy_NotFound(t *testing.T) {
	mb := newMockBackend()
	arch, _ := newTestArchitect(t, mb)

	_, err := arch.Destroy(context.Background(), "w-ghost-00000")
	if err == nil {
		t.Fatal("expected error destroying nonexistent world")
	}
}

func TestDestroy_WritesJournal(t *testing.T) {
	mb := newMockBackend()
	arch, store := newTestArchitect(t, mb)

	// Create a temp agent Mind directory with journal layer
	// Note: journal.Append writes to mindPath/journal/ (not memory/journal)
	mindDir := t.TempDir()
	journalDir := mindDir + "/journal"
	if err := os.MkdirAll(journalDir, 0755); err != nil {
		t.Fatalf("create journal dir: %v", err)
	}

	w := models.World{
		ID:          "w-journal-11111",
		Config:      "journal-test",
		ContainerID: "mock-jrnl-1",
		Status:      models.StatusRunning,
		MindPath:    mindDir,
		CreatedAt:   time.Now().Add(-5 * time.Minute),
	}
	seedWorld(mb, w)
	_ = store

	_, err := arch.Destroy(context.Background(), "w-journal-11111")
	if err != nil {
		t.Fatalf("Destroy() error: %v", err)
	}

	// Check that a journal entry was written
	entries, err := os.ReadDir(journalDir)
	if err != nil {
		t.Fatalf("read journal dir: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected at least one journal entry after destroy")
	}
}

func TestDestroy_MultiAgentWritesAllJournals(t *testing.T) {
	mb := newMockBackend()
	arch, store := newTestArchitect(t, mb)

	// Create temp agent Mind directories
	agentsBase := t.TempDir()
	t.Setenv("SPWN_HOME", agentsBase)

	for _, name := range []string{"chief-a", "worker-b"} {
		journalDir := agentsBase + "/agents/" + name + "/journal"
		if err := os.MkdirAll(journalDir, 0755); err != nil {
			t.Fatalf("create journal dir for %s: %v", name, err)
		}
	}

	w := models.World{
		ID:          "w-multi-22222",
		Config:      "multi-test",
		ContainerID: "mock-multi-1",
		Status:      models.StatusRunning,
		CreatedAt:   time.Now().Add(-10 * time.Minute),
		Agents: []models.AgentRecord{
			{Name: "chief-a", AgentID: "a-chief-a-11111", Role: "chief", Status: models.StatusRunning},
			{Name: "worker-b", AgentID: "a-worker-b-22222", Role: "worker", Status: models.StatusRunning},
		},
	}
	seedWorld(mb, w)
	_ = store

	_, err := arch.Destroy(context.Background(), "w-multi-22222")
	if err != nil {
		t.Fatalf("Destroy() error: %v", err)
	}

	// Check journal entries for both agents
	for _, name := range []string{"chief-a", "worker-b"} {
		journalDir := agentsBase + "/agents/" + name + "/journal"
		entries, err := os.ReadDir(journalDir)
		if err != nil {
			t.Fatalf("read journal dir for %s: %v", name, err)
		}
		if len(entries) == 0 {
			t.Errorf("expected journal entry for agent %s after destroy", name)
		}
	}
}

func TestDestroyAll_RemovesAllWorlds(t *testing.T) {
	mb := newMockBackend()
	arch, store := newTestArchitect(t, mb)

	// Seed multiple worlds via the mock backend
	for _, id := range []string{"w-all-11111", "w-all-22222", "w-all-33333"} {
		seedWorld(mb, models.World{
			ID:          id,
			Config:      "test",
			ContainerID: "ctr-" + id,
			Status:      models.StatusRunning,
		})
	}
	_ = store

	destroyed, err := arch.DestroyAll(context.Background())
	if err != nil {
		t.Fatalf("DestroyAll() error: %v", err)
	}
	if len(destroyed) != 3 {
		t.Errorf("expected 3 destroyed worlds, got %d", len(destroyed))
	}

	// Verify all worlds are gone from state
	worlds, _ := store.List()
	if len(worlds) != 0 {
		t.Errorf("expected 0 worlds after DestroyAll, got %d", len(worlds))
	}
}

func TestDestroyAll_EmptyState(t *testing.T) {
	mb := newMockBackend()
	arch, _ := newTestArchitect(t, mb)

	destroyed, err := arch.DestroyAll(context.Background())
	if err != nil {
		t.Fatalf("DestroyAll() error: %v", err)
	}
	if len(destroyed) != 0 {
		t.Errorf("expected 0 destroyed worlds, got %d", len(destroyed))
	}
}

func TestSnapshot_Success(t *testing.T) {
	mb := newMockBackend()
	arch, store := newTestArchitect(t, mb)

	w := models.World{
		ID:          "w-snap-22222",
		ContainerID: "mock-snap-ctr",
		Status:      models.StatusIdle,
	}
	seedWorld(mb, w)
	_ = store

	tag, err := arch.Snapshot(context.Background(), "w-snap-22222", "mysnap")
	if err != nil {
		t.Fatalf("Snapshot() error: %v", err)
	}
	if !strings.Contains(tag, "spwn-snapshot:") {
		t.Errorf("expected snapshot tag, got %s", tag)
	}
	if !strings.Contains(tag, "mysnap") {
		t.Errorf("expected tag to contain 'mysnap', got %s", tag)
	}
}

func TestSnapshot_WorldNotFound(t *testing.T) {
	mb := newMockBackend()
	arch, _ := newTestArchitect(t, mb)

	_, err := arch.Snapshot(context.Background(), "w-nope-00000", "test")
	if err == nil {
		t.Fatal("expected error for nonexistent world")
	}
}

func TestSnapshot_CommitError(t *testing.T) {
	mb := newMockBackend()
	mb.commitErr = fmt.Errorf("disk full")
	arch, store := newTestArchitect(t, mb)

	seedWorld(mb, models.World{ID: "w-err-33333", ContainerID: "ctr-1", Status: models.StatusIdle})
	_ = store

	_, err := arch.Snapshot(context.Background(), "w-err-33333", "test")
	if err == nil {
		t.Fatal("expected error when commit fails")
	}
	if !strings.Contains(err.Error(), "disk full") {
		t.Errorf("error should propagate cause, got: %v", err)
	}
}

func TestMockBackendImplementsInterface(t *testing.T) {
	// Compile-time check that mockBackend satisfies backend.Backend
	var _ backend.Backend = (*mockBackend)(nil)
}

func TestDeleteSnapshot_Success(t *testing.T) {
	mb := newMockBackend()
	arch, _ := newTestArchitect(t, mb)

	err := arch.DeleteSnapshot(context.Background(), "spwn-snapshot:w-test--mysnap")
	if err != nil {
		t.Fatalf("DeleteSnapshot() error: %v", err)
	}
}

func TestDeleteSnapshot_Error(t *testing.T) {
	mb := newMockBackend()
	mb.imageRemoveE = fmt.Errorf("image in use")
	arch, _ := newTestArchitect(t, mb)

	err := arch.DeleteSnapshot(context.Background(), "spwn-snapshot:test")
	if err == nil {
		t.Fatal("expected error")
	}
}
