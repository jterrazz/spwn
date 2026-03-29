// Package agent provides the public API for the agent domain.
// It wraps mind, journal, and session operations.
package agent

import (
	"fmt"
	"os"
	"time"

	"spwn.sh/core/agent/internal/evolution"
	"spwn.sh/core/agent/internal/journal"
	"spwn.sh/core/agent/internal/mind"
	"spwn.sh/core/agent/internal/session"
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

// DeleteAgent removes the agent's Mind directory entirely.
// Returns an error if the agent does not exist.
func DeleteAgent(name string) error {
	dir := AgentDir(name)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("agent %q not found", name)
	}
	return os.RemoveAll(dir)
}

// --- Mind operations ---

// AgentDir returns the absolute path to the agent's Mind directory
// (~/.spwn/agents/{name}/).
func AgentDir(name string) string {
	return mind.AgentDir(name)
}

// InitMind scaffolds a new Mind for the named agent, creating all 6 layers
// (personas, skills, knowledge, playbooks, journal, sessions) and returning
// the directory path.
func InitMind(name string) (string, error) {
	return mind.Init(name)
}

// ValidateMind checks that the named agent's Mind directory exists and contains
// at least the personas layer.
func ValidateMind(name string) error {
	return mind.Validate(name)
}

// ListAgents returns metadata for every agent found in ~/.spwn/agents/.
func ListAgents() ([]Info, error) {
	return mind.List()
}

// InspectAgent returns detailed information about the named agent's Mind,
// including layer contents and file counts.
func InspectAgent(name string) (*Info, error) {
	return mind.Inspect(name)
}

// LayerCount returns the number of Mind layers that contain at least one file.
func LayerCount(info *Info) int {
	return mind.LayerCount(info)
}

// ExportMind creates a tar.gz archive of the named agent's Mind directory,
// optionally excluding the specified layers, and returns the archive path.
func ExportMind(name string, outputPath string, excludeLayers []string) (string, error) {
	return mind.Export(name, outputPath, excludeLayers)
}

// ImportMind extracts a tar.gz archive into the named agent's Mind directory,
// overwriting existing files.
func ImportMind(name string, archivePath string) error {
	return mind.Import(name, archivePath)
}

// --- Journal operations ---

// AppendJournal writes a timestamped journal entry for the given universe
// session to the agent's journal directory.
func AppendJournal(mindPath, universeID string, exitCode int, duration time.Duration) error {
	return journal.Append(mindPath, universeID, exitCode, duration)
}

// ListJournal returns the last n journal entries from the agent's journal
// directory, ordered newest first.
func ListJournal(mindPath string, n int) ([]JournalEntry, error) {
	return journal.List(mindPath, n)
}

// --- Session operations ---

// DeterministicSessionID generates a stable session ID derived from the agent
// name and universe ID, ensuring the same pair always maps to the same session.
func DeterministicSessionID(agentName, universeID string) string {
	return session.DeterministicID(agentName, universeID)
}

// LoadSession reads and parses the session file for the given universe from the
// agent's sessions directory.
func LoadSession(mindPath, universeID string) (*Session, error) {
	return session.Load(mindPath, universeID)
}

// SaveSession persists the given session to the agent's sessions directory,
// creating the file if it does not exist.
func SaveSession(mindPath string, s *Session) error {
	return session.Save(mindPath, s)
}

// --- Evolution operations ---

// Reflect analyzes the named agent's recent journal entries and promotes
// successful strategies into playbooks/auto-reflexion.md.
func Reflect(name string) (*ReflexionResult, error) {
	mindPath := AgentDir(name)
	return evolution.Reflect(mindPath)
}

// Sleep consolidates the named agent's raw experience into durable knowledge,
// archiving stale files and pruning old sessions.
func Sleep(name string) (*SleepResult, error) {
	mindPath := AgentDir(name)
	return evolution.Sleep(mindPath)
}

// Fork clones the Mind from the source agent to the target agent, copying only
// the specified layers. Returns metadata about the cloned layers.
func Fork(source, target string, layers []string) (*ForkResult, error) {
	return evolution.Fork(source, target, layers)
}
