// Package agent provides the public API for the agent domain.
// It wraps mind, journal, and session operations.
package agent

import (
	"time"

	"github.com/jterrazz/spwn/core/agent/internal/evolution"
	"github.com/jterrazz/spwn/core/agent/internal/journal"
	"github.com/jterrazz/spwn/core/agent/internal/mind"
	"github.com/jterrazz/spwn/core/agent/internal/session"
)

// Info describes an agent's Mind structure.
type Info = mind.AgentInfo

// JournalEntry represents a parsed journal entry.
type JournalEntry = journal.Entry

// Session tracks an agent's conversation state within a universe.
type Session = session.Session

// ReflexionResult holds the outcome of a reflexion analysis.
type ReflexionResult = evolution.ReflexionResult

// SleepResult holds the outcome of a sleep cycle.
type SleepResult = evolution.SleepResult

// ForkResult holds the outcome of a fork operation.
type ForkResult = evolution.ForkResult

// --- Mind operations ---

// AgentDir returns the path to ~/.spwn/agents/{name}/.
func AgentDir(name string) string {
	return mind.AgentDir(name)
}

// InitMind scaffolds a new Mind with all 6 layers.
func InitMind(name string) (string, error) {
	return mind.Init(name)
}

// ValidateMind checks that a Mind directory exists and has the personas layer.
func ValidateMind(name string) error {
	return mind.Validate(name)
}

// ListAgents returns all agents in ~/.spwn/agents/.
func ListAgents() ([]Info, error) {
	return mind.List()
}

// InspectAgent returns detailed information about an agent's Mind.
func InspectAgent(name string) (*Info, error) {
	return mind.Inspect(name)
}

// LayerCount returns how many layers have at least one file.
func LayerCount(info *Info) int {
	return mind.LayerCount(info)
}

// ExportMind creates a tar.gz archive of an agent's Mind directory.
func ExportMind(name string, outputPath string, excludeLayers []string) (string, error) {
	return mind.Export(name, outputPath, excludeLayers)
}

// ImportMind extracts a tar.gz archive into an agent's Mind directory.
func ImportMind(name string, archivePath string) error {
	return mind.Import(name, archivePath)
}

// --- Journal operations ---

// AppendJournal writes a new journal entry to the Mind's journal directory.
func AppendJournal(mindPath, universeID string, exitCode int, duration time.Duration) error {
	return journal.Append(mindPath, universeID, exitCode, duration)
}

// ListJournal returns the last n journal entries, newest first.
func ListJournal(mindPath string, n int) ([]JournalEntry, error) {
	return journal.List(mindPath, n)
}

// --- Session operations ---

// DeterministicSessionID generates a session ID from agent name and universe ID.
func DeterministicSessionID(agentName, universeID string) string {
	return session.DeterministicID(agentName, universeID)
}

// LoadSession reads a session file from the Mind's sessions directory.
func LoadSession(mindPath, universeID string) (*Session, error) {
	return session.Load(mindPath, universeID)
}

// SaveSession writes a session file to the Mind's sessions directory.
func SaveSession(mindPath string, s *Session) error {
	return session.Save(mindPath, s)
}

// --- Evolution operations ---

// Reflect analyzes recent journal entries and promotes successful patterns to playbooks.
func Reflect(name string) (*ReflexionResult, error) {
	mindPath := AgentDir(name)
	return evolution.Reflect(mindPath)
}

// Sleep consolidates experience into durable knowledge.
func Sleep(name string) (*SleepResult, error) {
	mindPath := AgentDir(name)
	return evolution.Sleep(mindPath)
}

// Fork clones a Mind from source agent to target agent.
func Fork(source, target string, layers []string) (*ForkResult, error) {
	return evolution.Fork(source, target, layers)
}
