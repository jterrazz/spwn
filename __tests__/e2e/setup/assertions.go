//go:build e2e

package setup

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jterrazz/universe/internal/config"
	"github.com/jterrazz/universe/internal/journal"
	"github.com/jterrazz/universe/internal/mind"
	"github.com/jterrazz/universe/internal/session"
)

// --- AssertionChain ---

// AssertionChain provides fluent assertions on a spawned universe.
type AssertionChain struct {
	tc       *TestContext
	universe *config.Universe
}

// Universe returns the underlying universe record.
func (a *AssertionChain) Universe() *config.Universe {
	return a.universe
}

// ExecInContainer runs a command inside the universe's container and returns stdout.
func (a *AssertionChain) ExecInContainer(cmd []string) string {
	a.tc.T.Helper()
	return a.tc.ExecInContainer(a.universe.ContainerID, cmd)
}

// ExpectState asserts against the state.json contents.
func (a *AssertionChain) ExpectState(fn func(s *StateAssertion)) *AssertionChain {
	a.tc.T.Helper()
	fn(&StateAssertion{tc: a.tc})
	return a
}

// ExpectContainer asserts against the Docker container.
func (a *AssertionChain) ExpectContainer(fn func(c *ContainerAssertion)) *AssertionChain {
	a.tc.T.Helper()
	fn(&ContainerAssertion{tc: a.tc, containerID: a.universe.ContainerID})
	return a
}

// ExpectMind asserts against the agent's Mind directory.
func (a *AssertionChain) ExpectMind(fn func(m *MindAssertion)) *AssertionChain {
	a.tc.T.Helper()
	if a.universe.Agent == "" {
		a.tc.T.Fatal("ExpectMind called but no agent is set on this universe")
	}
	fn(&MindAssertion{tc: a.tc, agentName: a.universe.Agent})
	return a
}

// ExpectMock asserts against the mock Claude output inside the container.
func (a *AssertionChain) ExpectMock(fn func(m *MockAssertion)) *AssertionChain {
	a.tc.T.Helper()
	mock := a.tc.ReadMockOutput(a.universe.ContainerID)
	fn(&MockAssertion{tc: a.tc, mock: mock})
	return a
}

// ExpectGate asserts against the gate bridges inside the container.
func (a *AssertionChain) ExpectGate(fn func(g *GateAssertion)) *AssertionChain {
	a.tc.T.Helper()
	fn(&GateAssertion{tc: a.tc, containerID: a.universe.ContainerID})
	return a
}

// ExpectSession asserts against the session persistence.
func (a *AssertionChain) ExpectSession(fn func(s *SessionAssertion)) *AssertionChain {
	a.tc.T.Helper()
	if a.universe.Agent == "" {
		a.tc.T.Fatal("ExpectSession called but no agent is set")
	}
	fn(&SessionAssertion{tc: a.tc, agentName: a.universe.Agent})
	return a
}

// ExpectJournal asserts against journal entries.
func (a *AssertionChain) ExpectJournal(fn func(j *JournalAssertion)) *AssertionChain {
	a.tc.T.Helper()
	if a.universe.Agent == "" {
		a.tc.T.Fatal("ExpectJournal called but no agent is set")
	}
	fn(&JournalAssertion{tc: a.tc, agentName: a.universe.Agent})
	return a
}

// Destroy destroys the universe and returns a new chain for post-destroy assertions.
func (a *AssertionChain) Destroy() *AssertionChain {
	a.tc.T.Helper()
	_, err := a.tc.Arc.Destroy(context.Background(), a.universe.ID)
	if err != nil {
		a.tc.T.Fatalf("Destroy failed: %v", err)
	}
	return a
}

// List returns the universe list for assertions.
func (a *AssertionChain) List() *ListAssertionChain {
	a.tc.T.Helper()
	universes, err := a.tc.Arc.List(context.Background())
	if err != nil {
		a.tc.T.Fatalf("List failed: %v", err)
	}
	return &ListAssertionChain{tc: a.tc, universes: universes}
}

// Inspect returns inspection data for assertions.
func (a *AssertionChain) Inspect() *InspectAssertionChain {
	a.tc.T.Helper()
	u, err := a.tc.Arc.Inspect(context.Background(), a.universe.ID)
	if err != nil {
		a.tc.T.Fatalf("Inspect failed: %v", err)
	}
	return &InspectAssertionChain{tc: a.tc, universe: u}
}

// --- StateAssertion ---

type StateAssertion struct {
	tc *TestContext
}

func (s *StateAssertion) UniverseCount(expected int) {
	s.tc.T.Helper()
	universes := s.tc.LoadState()
	if len(universes) != expected {
		s.tc.T.Fatalf("Expected %d universe(s), got %d", expected, len(universes))
	}
}

func (s *StateAssertion) UniverseStatus(expected config.UniverseStatus) {
	s.tc.T.Helper()
	universes := s.tc.LoadState()
	if len(universes) == 0 {
		s.tc.T.Fatal("No universes in state")
	}
	if universes[0].Status != expected {
		s.tc.T.Fatalf("Expected status %q, got %q", expected, universes[0].Status)
	}
}

func (s *StateAssertion) HasAgent(name string) {
	s.tc.T.Helper()
	universes := s.tc.LoadState()
	for _, u := range universes {
		if u.Agent == name {
			return
		}
	}
	s.tc.T.Fatalf("Expected agent %q in state, not found", name)
}

func (s *StateAssertion) HasNoAgent() {
	s.tc.T.Helper()
	universes := s.tc.LoadState()
	for _, u := range universes {
		if u.Agent != "" {
			s.tc.T.Fatalf("Expected no agent in state, found %q", u.Agent)
		}
	}
}

// --- ContainerAssertion ---

type ContainerAssertion struct {
	tc          *TestContext
	containerID string
}

func (c *ContainerAssertion) IsRunning() {
	c.tc.T.Helper()
	running, err := c.tc.Backend.IsRunning(context.Background(), c.containerID)
	if err != nil {
		c.tc.T.Fatalf("Failed to check container: %v", err)
	}
	if !running {
		c.tc.T.Fatal("Expected container to be running")
	}
}

func (c *ContainerAssertion) NotExists() {
	c.tc.T.Helper()
	_, err := c.tc.Backend.IsRunning(context.Background(), c.containerID)
	if err == nil {
		c.tc.T.Fatal("Expected container to not exist, but it does")
	}
}

func (c *ContainerAssertion) HasMount(mountPath string) {
	c.tc.T.Helper()
	if !c.tc.DirExistsInContainer(c.containerID, mountPath) {
		c.tc.T.Fatalf("Expected mount at %s, not found", mountPath)
	}
}

func (c *ContainerAssertion) HasFile(path string) {
	c.tc.T.Helper()
	if !c.tc.FileExistsInContainer(c.containerID, path) {
		c.tc.T.Fatalf("Expected file %s, not found", path)
	}
}

func (c *ContainerAssertion) FileContains(path, substring string) {
	c.tc.T.Helper()
	content := c.tc.ReadFileInContainer(c.containerID, path)
	if !strings.Contains(content, substring) {
		c.tc.T.Fatalf("Expected %s to contain %q, got:\n%s", path, substring, content)
	}
}

// --- MindAssertion ---

type MindAssertion struct {
	tc        *TestContext
	agentName string
}

func (m *MindAssertion) HasLayer(layer string) {
	m.tc.T.Helper()
	info, err := mind.Inspect(m.agentName)
	if err != nil {
		m.tc.T.Fatalf("Failed to inspect agent: %v", err)
	}
	if _, ok := info.Layers[layer]; !ok {
		m.tc.T.Fatalf("Expected Mind layer %q, not found", layer)
	}
}

func (m *MindAssertion) HasFile(relPath string) {
	m.tc.T.Helper()
	info, err := mind.Inspect(m.agentName)
	if err != nil {
		m.tc.T.Fatalf("Failed to inspect agent: %v", err)
	}

	// relPath is like "personas/default.md"
	parts := strings.SplitN(relPath, "/", 2)
	if len(parts) != 2 {
		m.tc.T.Fatalf("Invalid relPath %q, expected layer/file", relPath)
	}
	layer, file := parts[0], parts[1]

	files, ok := info.Layers[layer]
	if !ok {
		m.tc.T.Fatalf("Mind layer %q not found", layer)
	}
	for _, f := range files {
		if f == file {
			return
		}
	}
	m.tc.T.Fatalf("Expected file %q in Mind layer %q, not found. Files: %v", file, layer, files)
}

// --- MockAssertion ---

type MockAssertion struct {
	tc   *TestContext
	mock *MockOutput
}

func (m *MockAssertion) WasCalled() {
	m.tc.T.Helper()
	if m.mock.PID == 0 {
		m.tc.T.Fatal("Mock claude was not called (PID is 0)")
	}
}

func (m *MockAssertion) SawMind() {
	m.tc.T.Helper()
	if !m.mock.MindExists {
		m.tc.T.Fatal("Mock claude did not see /mind")
	}
}

func (m *MockAssertion) SawPhysics() {
	m.tc.T.Helper()
	if !m.mock.PhysicsExists {
		m.tc.T.Fatal("Mock claude did not see /universe/physics.md")
	}
}

func (m *MockAssertion) SawFaculties() {
	m.tc.T.Helper()
	if !m.mock.FacultiesExists {
		m.tc.T.Fatal("Mock claude did not see /universe/faculties.md")
	}
}

func (m *MockAssertion) SawWorkspace() {
	m.tc.T.Helper()
	if !m.mock.WorkspaceExists {
		m.tc.T.Fatal("Mock claude did not see /workspace")
	}
}

func (m *MockAssertion) PhysicsContains(substring string) {
	m.tc.T.Helper()
	if !strings.Contains(m.mock.PhysicsContent, substring) {
		m.tc.T.Fatalf("Expected physics to contain %q, got:\n%s", substring, m.mock.PhysicsContent)
	}
}

func (m *MockAssertion) FacultiesContains(substring string) {
	m.tc.T.Helper()
	if !strings.Contains(m.mock.FacultiesContent, substring) {
		m.tc.T.Fatalf("Expected faculties to contain %q, got:\n%s", substring, m.mock.FacultiesContent)
	}
}

func (m *MockAssertion) HasSessionID() {
	m.tc.T.Helper()
	if m.mock.SessionID == "" {
		m.tc.T.Fatal("Expected mock to receive --session-id, but it was empty")
	}
}

func (m *MockAssertion) SessionIDEquals(expected string) {
	m.tc.T.Helper()
	if m.mock.SessionID != expected {
		m.tc.T.Fatalf("Expected session ID %q, got %q", expected, m.mock.SessionID)
	}
}

func (m *MockAssertion) WasResumed() {
	m.tc.T.Helper()
	if !m.mock.Resume {
		m.tc.T.Fatal("Expected mock to be called with --resume, but it wasn't")
	}
}

func (m *MockAssertion) WasNotResumed() {
	m.tc.T.Helper()
	if m.mock.Resume {
		m.tc.T.Fatal("Expected mock NOT to be called with --resume, but it was")
	}
}

// --- SessionAssertion ---

type SessionAssertion struct {
	tc        *TestContext
	agentName string
}

func (s *SessionAssertion) HasSessionFile(universeID string) {
	s.tc.T.Helper()
	mindPath := mind.AgentDir(s.agentName)
	sess, err := session.Load(mindPath, universeID)
	if err != nil {
		s.tc.T.Fatalf("Failed to load session: %v", err)
	}
	if sess == nil {
		s.tc.T.Fatalf("Expected session file for universe %s, not found", universeID)
	}
}

func (s *SessionAssertion) SessionIDIsDeterministic(universeID string) {
	s.tc.T.Helper()
	id1 := session.DeterministicID(s.agentName, universeID)
	id2 := session.DeterministicID(s.agentName, universeID)
	if id1 != id2 {
		s.tc.T.Fatalf("Session IDs not deterministic: %q != %q", id1, id2)
	}
	if len(id1) != 16 {
		s.tc.T.Fatalf("Expected 16-char session ID, got %d: %q", len(id1), id1)
	}
}

// --- JournalAssertion ---

type JournalAssertion struct {
	tc        *TestContext
	agentName string
}

func (j *JournalAssertion) HasEntries(minCount int) {
	j.tc.T.Helper()
	mindPath := mind.AgentDir(j.agentName)
	entries, err := journal.List(mindPath, 0)
	if err != nil {
		j.tc.T.Fatalf("Failed to list journal: %v", err)
	}
	if len(entries) < minCount {
		j.tc.T.Fatalf("Expected at least %d journal entries, got %d", minCount, len(entries))
	}
}

func (j *JournalAssertion) LatestOutcome(expected string) {
	j.tc.T.Helper()
	mindPath := mind.AgentDir(j.agentName)
	entries, err := journal.List(mindPath, 1)
	if err != nil || len(entries) == 0 {
		j.tc.T.Fatalf("No journal entries found")
	}
	if entries[0].Outcome != expected {
		j.tc.T.Fatalf("Expected latest journal outcome %q, got %q", expected, entries[0].Outcome)
	}
}

func (j *JournalAssertion) LatestUniverseID(expected string) {
	j.tc.T.Helper()
	mindPath := mind.AgentDir(j.agentName)
	entries, err := journal.List(mindPath, 1)
	if err != nil || len(entries) == 0 {
		j.tc.T.Fatalf("No journal entries found")
	}
	if entries[0].UniverseID != expected {
		j.tc.T.Fatalf("Expected latest journal universe ID %q, got %q", expected, entries[0].UniverseID)
	}
}

// --- ExportAssertion ---

type ExportAssertion struct {
	tc          *TestContext
	archivePath string
}

func (e *ExportAssertion) ArchiveExists() {
	e.tc.T.Helper()
	if _, err := os.Stat(e.archivePath); err != nil {
		e.tc.T.Fatalf("Expected archive at %s, not found", e.archivePath)
	}
}

func (e *ExportAssertion) ArchiveNonEmpty() {
	e.tc.T.Helper()
	info, err := os.Stat(e.archivePath)
	if err != nil {
		e.tc.T.Fatalf("Archive not found: %v", err)
	}
	if info.Size() == 0 {
		e.tc.T.Fatal("Expected non-empty archive")
	}
}

func (e *ExportAssertion) Path() string {
	return e.archivePath
}

// --- ListAssertionChain ---

type ListAssertionChain struct {
	tc        *TestContext
	universes []config.Universe
}

func (l *ListAssertionChain) ExpectCount(n int) *ListAssertionChain {
	l.tc.T.Helper()
	if len(l.universes) != n {
		l.tc.T.Fatalf("Expected %d universe(s) in list, got %d", n, len(l.universes))
	}
	return l
}

func (l *ListAssertionChain) ExpectUniverse(index int, fn func(e *ListEntryAssertion)) *ListAssertionChain {
	l.tc.T.Helper()
	if index >= len(l.universes) {
		l.tc.T.Fatalf("Index %d out of range (have %d universes)", index, len(l.universes))
	}
	fn(&ListEntryAssertion{tc: l.tc, universe: l.universes[index]})
	return l
}

// --- ListEntryAssertion ---

type ListEntryAssertion struct {
	tc       *TestContext
	universe config.Universe
}

func (e *ListEntryAssertion) StatusIs(status config.UniverseStatus) {
	e.tc.T.Helper()
	if e.universe.Status != status {
		e.tc.T.Fatalf("Expected status %q, got %q", status, e.universe.Status)
	}
}

// --- InspectAssertionChain ---

type InspectAssertionChain struct {
	tc       *TestContext
	universe *config.Universe
}

func (i *InspectAssertionChain) ExpectConfig(name string) *InspectAssertionChain {
	i.tc.T.Helper()
	if i.universe.Config != name {
		i.tc.T.Fatalf("Expected config %q, got %q", name, i.universe.Config)
	}
	return i
}

// --- AgentAssertionChain ---

type AgentAssertionChain struct {
	tc        *TestContext
	agentName string
}

func (a *AgentAssertionChain) ExpectMind(fn func(m *MindAssertion)) *AgentAssertionChain {
	a.tc.T.Helper()
	fn(&MindAssertion{tc: a.tc, agentName: a.agentName})
	return a
}

func (a *AgentAssertionChain) Export(outputDir string, exclude []string) *ExportAssertion {
	a.tc.T.Helper()
	archivePath, err := mind.Export(a.agentName, outputDir, exclude)
	if err != nil {
		a.tc.T.Fatalf("Export failed: %v", err)
	}
	return &ExportAssertion{tc: a.tc, archivePath: archivePath}
}

func (a *AgentAssertionChain) ImportFrom(archivePath string) *AgentAssertionChain {
	a.tc.T.Helper()
	if err := mind.Import(a.agentName, archivePath); err != nil {
		a.tc.T.Fatalf("Import failed: %v", err)
	}
	return a
}

// HasSessionFile checks that a session file exists for this agent+universe combo.
func (m *MindAssertion) HasSessionFile(universeID string) {
	m.tc.T.Helper()
	mindPath := mind.AgentDir(m.agentName)
	sessionPath := filepath.Join(mindPath, "sessions", universeID+".json")
	if _, err := os.Stat(sessionPath); err != nil {
		m.tc.T.Fatalf("Expected session file at %s, not found", sessionPath)
	}
}

// HasJournalEntries checks that the journal directory has entries.
func (m *MindAssertion) HasJournalEntries(minCount int) {
	m.tc.T.Helper()
	mindPath := mind.AgentDir(m.agentName)
	entries, err := journal.List(mindPath, 0)
	if err != nil {
		m.tc.T.Fatalf("Failed to list journal: %v", err)
	}
	if len(entries) < minCount {
		m.tc.T.Fatalf("Expected at least %d journal entries, got %d", minCount, len(entries))
	}
}

// --- GateAssertion ---

type GateAssertion struct {
	tc          *TestContext
	containerID string
}

func (g *GateAssertion) HasBridge(name string) {
	g.tc.T.Helper()
	if !g.tc.FileExistsInContainer(g.containerID, "/gate/bin/"+name) {
		g.tc.T.Fatalf("Expected gate bridge %q at /gate/bin/%s, not found", name, name)
	}
}

func (g *GateAssertion) BridgeIsExecutable(name string) {
	g.tc.T.Helper()
	_, err := g.tc.Backend.ExecOutput(context.Background(), g.containerID, []string{
		"test", "-x", "/gate/bin/" + name,
	})
	if err != nil {
		g.tc.T.Fatalf("Expected bridge %q to be executable, but it's not", name)
	}
}

func (g *GateAssertion) HasSocket() {
	g.tc.T.Helper()
	// Use test -e (exists, any type) instead of test -f (regular file only) since sockets are special files.
	// On macOS Docker Desktop, sockets through bind mounts are not visible inside the container.
	_, err := g.tc.Backend.ExecOutput(context.Background(), g.containerID, []string{"test", "-e", "/gate/gate.sock"})
	if err != nil {
		if runtime.GOOS == "darwin" {
			g.tc.T.Skip("unix socket not visible through Docker bind mount on macOS")
		}
		g.tc.T.Fatal("Expected gate socket at /gate/gate.sock, not found")
	}
}

func (g *GateAssertion) NoSocket() {
	g.tc.T.Helper()
	if g.tc.DirExistsInContainer(g.containerID, "/gate") {
		g.tc.T.Fatal("Expected no /gate directory, but it exists")
	}
}

func (g *GateAssertion) NoBridge(name string) {
	g.tc.T.Helper()
	if g.tc.FileExistsInContainer(g.containerID, "/gate/bin/"+name) {
		g.tc.T.Fatalf("Expected NO gate bridge %q, but it exists", name)
	}
	// Also check /usr/local/bin
	if g.tc.FileExistsInContainer(g.containerID, "/usr/local/bin/"+name) {
		g.tc.T.Fatalf("Expected NO bridge symlink for %q in /usr/local/bin, but it exists", name)
	}
}
