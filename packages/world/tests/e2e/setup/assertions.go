//go:build e2e

package setup

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"spwn.sh/packages/agent"
	"spwn.sh/packages/world"
)

// --- AssertionChain ---

// AssertionChain provides fluent assertions on a spawned world.
type AssertionChain struct {
	tc *TestContext
	w  *world.World
}

// World returns the underlying world record.
func (a *AssertionChain) World() *world.World {
	return a.w
}

// ExecInContainer runs a command inside the world container and returns stdout.
func (a *AssertionChain) ExecInContainer(cmd []string) string {
	a.tc.T.Helper()
	return a.tc.ExecInContainer(a.w.ContainerID, cmd)
}

// ExpectState asserts against the world-state surface (Docker labels
// + runtimestate, the replacement for the old state.json).
func (a *AssertionChain) ExpectState(fn func(s *StateAssertion)) *AssertionChain {
	a.tc.T.Helper()
	fn(&StateAssertion{tc: a.tc})
	return a
}

// ExpectContainer asserts against the Docker container.
func (a *AssertionChain) ExpectContainer(fn func(c *ContainerAssertion)) *AssertionChain {
	a.tc.T.Helper()
	fn(&ContainerAssertion{tc: a.tc, containerID: a.w.ContainerID})
	return a
}

// ExpectMind asserts against the agent's Mind directory.
func (a *AssertionChain) ExpectMind(fn func(m *MindAssertion)) *AssertionChain {
	a.tc.T.Helper()
	if a.w.Agent == "" {
		a.tc.T.Fatal("ExpectMind called but no agent is set on this world")
	}
	fn(&MindAssertion{tc: a.tc, agentName: a.w.Agent})
	return a
}

// ExpectMock asserts against the mock Claude output inside the container.
func (a *AssertionChain) ExpectMock(fn func(m *MockAssertion)) *AssertionChain {
	a.tc.T.Helper()
	mock := a.tc.ReadMockOutput(a.w.ContainerID)
	fn(&MockAssertion{tc: a.tc, mock: mock})
	return a
}

// ExpectSession asserts against the session persistence.
func (a *AssertionChain) ExpectSession(fn func(s *SessionAssertion)) *AssertionChain {
	a.tc.T.Helper()
	if a.w.Agent == "" {
		a.tc.T.Fatal("ExpectSession called but no agent is set")
	}
	fn(&SessionAssertion{tc: a.tc, agentName: a.w.Agent})
	return a
}

// ExpectJournal asserts against journal entries.
func (a *AssertionChain) ExpectJournal(fn func(j *JournalAssertion)) *AssertionChain {
	a.tc.T.Helper()
	if a.w.Agent == "" {
		a.tc.T.Fatal("ExpectJournal called but no agent is set")
	}
	fn(&JournalAssertion{tc: a.tc, agentName: a.w.Agent})
	return a
}

// Destroy destroys the world and returns a new chain for post-destroy assertions.
func (a *AssertionChain) Destroy() *AssertionChain {
	a.tc.T.Helper()
	_, err := a.tc.Arc.Destroy(context.Background(), a.w.ID)
	if err != nil {
		a.tc.T.Fatalf("Destroy failed: %v", err)
	}
	return a
}

// List returns the world list for assertions.
func (a *AssertionChain) List() *ListAssertionChain {
	a.tc.T.Helper()
	worlds, err := a.tc.Arc.List(context.Background())
	if err != nil {
		a.tc.T.Fatalf("List failed: %v", err)
	}
	return &ListAssertionChain{tc: a.tc, worlds: worlds}
}

// Inspect returns inspection data for assertions.
func (a *AssertionChain) Inspect() *InspectAssertionChain {
	a.tc.T.Helper()
	u, err := a.tc.Arc.Inspect(context.Background(), a.w.ID)
	if err != nil {
		a.tc.T.Fatalf("Inspect failed: %v", err)
	}
	return &InspectAssertionChain{tc: a.tc, w: u}
}

// --- StateAssertion ---

type StateAssertion struct {
	tc *TestContext
}

func (s *StateAssertion) WorldCount(expected int) {
	s.tc.T.Helper()
	worlds := s.tc.LoadState()
	if len(worlds) != expected {
		s.tc.T.Fatalf("Expected %d world(s), got %d", expected, len(worlds))
	}
}

func (s *StateAssertion) WorldStatus(expected world.Status) {
	s.tc.T.Helper()
	worlds := s.tc.LoadState()
	if len(worlds) == 0 {
		s.tc.T.Fatal("No worlds in state")
	}
	if worlds[0].Status != expected {
		s.tc.T.Fatalf("Expected status %q, got %q", expected, worlds[0].Status)
	}
}

func (s *StateAssertion) HasAgent(name string) {
	s.tc.T.Helper()
	worlds := s.tc.LoadState()
	for _, u := range worlds {
		if u.Agent == name {
			return
		}
	}
	s.tc.T.Fatalf("Expected agent %q in state, not found", name)
}

func (s *StateAssertion) HasNoAgent() {
	s.tc.T.Helper()
	worlds := s.tc.LoadState()
	for _, u := range worlds {
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

func (c *ContainerAssertion) FileNotContains(path, substring string) {
	c.tc.T.Helper()
	content := c.tc.ReadFileInContainer(c.containerID, path)
	if strings.Contains(content, substring) {
		c.tc.T.Fatalf("Expected %s NOT to contain %q, but it does", path, substring)
	}
}

// --- MindAssertion ---

type MindAssertion struct {
	tc        *TestContext
	agentName string
}

func (m *MindAssertion) HasLayer(layer string) {
	m.tc.T.Helper()
	info, err := agent.InspectAgent(m.agentName)
	if err != nil {
		m.tc.T.Fatalf("Failed to inspect agent: %v", err)
	}
	if _, ok := info.Layers[layer]; !ok {
		m.tc.T.Fatalf("Expected Mind layer %q, not found", layer)
	}
}

func (m *MindAssertion) HasFile(relPath string) {
	m.tc.T.Helper()
	info, err := agent.InspectAgent(m.agentName)
	if err != nil {
		m.tc.T.Fatalf("Failed to inspect agent: %v", err)
	}

	// Root-level files (SOUL.md, AGENTS.md, agent.yaml) live directly
	// under info.Path — no layer prefix. Fall back to a stat check.
	if !strings.Contains(relPath, "/") {
		p := filepath.Join(info.Path, relPath)
		if _, err := os.Stat(p); err != nil {
			m.tc.T.Fatalf("Expected %s at agent root, stat err=%v", relPath, err)
		}
		return
	}

	// relPath is like "skills/coding.md" or "journal/2025-01-01.md".
	// Try to match against known layers (longest prefix first).
	var matchedLayer, file string
	for layer := range info.Layers {
		prefix := layer + "/"
		if strings.HasPrefix(relPath, prefix) {
			// Pick the longest matching layer prefix
			if len(layer) > len(matchedLayer) {
				matchedLayer = layer
				file = relPath[len(prefix):]
			}
		}
	}

	if matchedLayer == "" {
		// Fallback: simple split for single-level layers
		parts := strings.SplitN(relPath, "/", 2)
		if len(parts) != 2 {
			m.tc.T.Fatalf("Invalid relPath %q, expected layer/file", relPath)
		}
		matchedLayer, file = parts[0], parts[1]
	}

	files, ok := info.Layers[matchedLayer]
	if !ok {
		m.tc.T.Fatalf("Mind layer %q not found", matchedLayer)
	}
	for _, f := range files {
		if f == file {
			return
		}
	}
	m.tc.T.Fatalf("Expected file %q in Mind layer %q, not found. Files: %v", file, matchedLayer, files)
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

// SawClaudeMD asserts the mock observed the per-agent CLAUDE.md —
// the single self-contained system prompt that replaces the old
// /world/AGENTS.md + /world/physics.md + /world/faculties.md +
// /world/skills/* split emission.
func (m *MockAssertion) SawClaudeMD() {
	m.tc.T.Helper()
	if !m.mock.ClaudeMDExists {
		m.tc.T.Fatal("Mock claude did not see /agents/<name>/CLAUDE.md")
	}
}

func (m *MockAssertion) SawWorkspace() {
	m.tc.T.Helper()
	if !m.mock.WorkspaceExists {
		m.tc.T.Fatal("Mock claude did not see /workspaces")
	}
}

// ClaudeMDContains asserts the CLAUDE.md the mock captured contains
// `substring`. Use to verify the inlined physics / faculties / roster
// content made it into the prompt.
func (m *MockAssertion) ClaudeMDContains(substring string) {
	m.tc.T.Helper()
	if !strings.Contains(m.mock.ClaudeMDContent, substring) {
		m.tc.T.Fatalf("Expected CLAUDE.md to contain %q, got:\n%s", substring, m.mock.ClaudeMDContent)
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

func (s *SessionAssertion) HasSessionFile(worldID string) {
	s.tc.T.Helper()
	mindPath := agent.AgentDir(s.agentName)
	sess, err := agent.LoadSession(mindPath, worldID)
	if err != nil {
		s.tc.T.Fatalf("Failed to load session: %v", err)
	}
	if sess == nil {
		s.tc.T.Fatalf("Expected session file for world %s, not found", worldID)
	}
}

func (s *SessionAssertion) SessionIDIsDeterministic(worldID string) {
	s.tc.T.Helper()
	id1 := agent.DeterministicSessionID(s.agentName, worldID)
	id2 := agent.DeterministicSessionID(s.agentName, worldID)
	if id1 != id2 {
		s.tc.T.Fatalf("Session IDs not deterministic: %q != %q", id1, id2)
	}
	if len(id1) != 36 {
		s.tc.T.Fatalf("Expected 36-char UUID session ID, got %d: %q", len(id1), id1)
	}
}

// --- JournalAssertion ---

type JournalAssertion struct {
	tc        *TestContext
	agentName string
}

func (j *JournalAssertion) HasEntries(minCount int) {
	j.tc.T.Helper()
	mindPath := agent.AgentDir(j.agentName)
	entries, err := agent.ListJournal(mindPath, 0)
	if err != nil {
		j.tc.T.Fatalf("Failed to list journal: %v", err)
	}
	if len(entries) < minCount {
		j.tc.T.Fatalf("Expected at least %d journal entries, got %d", minCount, len(entries))
	}
}

func (j *JournalAssertion) LatestOutcome(expected string) {
	j.tc.T.Helper()
	mindPath := agent.AgentDir(j.agentName)
	entries, err := agent.ListJournal(mindPath, 1)
	if err != nil || len(entries) == 0 {
		j.tc.T.Fatalf("No journal entries found")
	}
	if entries[0].Outcome != expected {
		j.tc.T.Fatalf("Expected latest journal outcome %q, got %q", expected, entries[0].Outcome)
	}
}

func (j *JournalAssertion) LatestWorldID(expected string) {
	j.tc.T.Helper()
	mindPath := agent.AgentDir(j.agentName)
	entries, err := agent.ListJournal(mindPath, 1)
	if err != nil || len(entries) == 0 {
		j.tc.T.Fatalf("No journal entries found")
	}
	if entries[0].WorldID != expected {
		j.tc.T.Fatalf("Expected latest journal world ID %q, got %q", expected, entries[0].WorldID)
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
	tc     *TestContext
	worlds []world.World
}

func (l *ListAssertionChain) ExpectCount(n int) *ListAssertionChain {
	l.tc.T.Helper()
	if len(l.worlds) != n {
		l.tc.T.Fatalf("Expected %d world(s) in list, got %d", n, len(l.worlds))
	}
	return l
}

func (l *ListAssertionChain) ExpectWorld(index int, fn func(e *ListEntryAssertion)) *ListAssertionChain {
	l.tc.T.Helper()
	if index >= len(l.worlds) {
		l.tc.T.Fatalf("Index %d out of range (have %d worlds)", index, len(l.worlds))
	}
	fn(&ListEntryAssertion{tc: l.tc, w: l.worlds[index]})
	return l
}

// --- ListEntryAssertion ---

type ListEntryAssertion struct {
	tc *TestContext
	w  world.World
}

func (e *ListEntryAssertion) StatusIs(status world.Status) {
	e.tc.T.Helper()
	if e.w.Status != status {
		e.tc.T.Fatalf("Expected status %q, got %q", status, e.w.Status)
	}
}

// --- InspectAssertionChain ---

type InspectAssertionChain struct {
	tc *TestContext
	w  *world.World
}

func (i *InspectAssertionChain) ExpectConfig(name string) *InspectAssertionChain {
	i.tc.T.Helper()
	if i.w.Config != name {
		i.tc.T.Fatalf("Expected config %q, got %q", name, i.w.Config)
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
	archivePath, err := agent.ExportMind(a.agentName, outputDir, exclude)
	if err != nil {
		a.tc.T.Fatalf("Export failed: %v", err)
	}
	return &ExportAssertion{tc: a.tc, archivePath: archivePath}
}

func (a *AgentAssertionChain) ImportFrom(archivePath string) *AgentAssertionChain {
	a.tc.T.Helper()
	if err := agent.ImportMind(a.agentName, archivePath); err != nil {
		a.tc.T.Fatalf("Import failed: %v", err)
	}
	return a
}

// HasSessionFile checks that a session file exists for this agent+world combo.
func (m *MindAssertion) HasSessionFile(worldID string) {
	m.tc.T.Helper()
	mindPath := agent.AgentDir(m.agentName)
	sessionPath := filepath.Join(mindPath, "journal", worldID+".json")
	if _, err := os.Stat(sessionPath); err != nil {
		m.tc.T.Fatalf("Expected session file at %s, not found", sessionPath)
	}
}

// HasJournalEntries checks that the journal directory has entries.
func (m *MindAssertion) HasJournalEntries(minCount int) {
	m.tc.T.Helper()
	mindPath := agent.AgentDir(m.agentName)
	entries, err := agent.ListJournal(mindPath, 0)
	if err != nil {
		m.tc.T.Fatalf("Failed to list journal: %v", err)
	}
	if len(entries) < minCount {
		m.tc.T.Fatalf("Expected at least %d journal entries, got %d", minCount, len(entries))
	}
}

