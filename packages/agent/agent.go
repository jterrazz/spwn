// Package agent provides the public API for the agent domain.
// It wraps mind, journal, and session operations.
package agent

import (
	"errors"
	"fmt"
	"os"
	"time"

	"spwn.sh/packages/agent/internal/evolution"
	"spwn.sh/packages/agent/internal/journal"
	"spwn.sh/packages/agent/internal/mind"
	"spwn.sh/packages/agent/internal/session"
	"spwn.sh/packages/activity"
)

// ErrNotFound is returned when an agent name has no matching on-disk
// tree. Use errors.Is(err, agent.ErrNotFound) instead of matching
// error strings. User-facing wrapping format: `agent %q not found`.
var ErrNotFound = errors.New("not found")

// Info describes an agent's Mind structure.
type Info = mind.AgentInfo

// JournalEntry represents a parsed journal entry.
type JournalEntry = journal.Entry

// Session tracks an agent's conversation state within a world.
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
		return fmt.Errorf("agent %q %w", name, ErrNotFound)
	}
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	activity.Log(activity.Event{
		Type:    activity.TypeAgentDeleted,
		Actor:   "user",
		Verb:    "deleted",
		Target:  name,
		Phrase:  activity.PhraseAgentDeleted(name),
		AgentID: name,
	})
	return nil
}

// --- Mind operations ---

// AgentDir returns the absolute path to the agent's Mind directory
// (~/.spwn/agents/{name}/).
func AgentDir(name string) string {
	return mind.AgentDir(name)
}

// InitMind scaffolds a new Mind for the named agent, creating all 5 layers
// (core, skills, knowledge, playbooks, journal) and returning the directory path.
func InitMind(name string) (string, error) {
	dir, err := mind.Init(name)
	if err != nil {
		return "", err
	}
	activity.Log(activity.Event{
		Type:    activity.TypeAgentCreated,
		Actor:   "user",
		Verb:    "created",
		Target:  name,
		Phrase:  activity.PhraseAgentCreated(name),
		AgentID: name,
	})
	return dir, nil
}

// ValidateMind checks that the named agent's Mind directory exists and contains
// at least the core layer.
func ValidateMind(name string) error {
	return mind.Validate(name)
}

// RepairMind re-creates missing Mind layer directories and the
// default profile for an already-existing agent. It is idempotent
// and safe to call on a valid Mind. Used by `agent create --force`
// to re-scaffold over a partially-deleted agent directory.
func RepairMind(name string) error {
	return mind.Repair(name)
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

// AppendJournal writes a timestamped journal entry for the given world
// session to the agent's journal directory.
func AppendJournal(mindPath, worldID string, exitCode int, duration time.Duration) error {
	return journal.Append(mindPath, worldID, exitCode, duration)
}

// ListJournal returns the last n journal entries from the agent's journal
// directory, ordered newest first.
func ListJournal(mindPath string, n int) ([]JournalEntry, error) {
	return journal.List(mindPath, n)
}

// --- Session operations ---

// DeterministicSessionID generates a stable session ID derived from the agent
// name and world ID, ensuring the same pair always maps to the same session.
func DeterministicSessionID(agentName, worldID string) string {
	return session.DeterministicID(agentName, worldID)
}

// LoadSession reads and parses the session file for the given world from the
// agent's sessions directory.
func LoadSession(mindPath, worldID string) (*Session, error) {
	return session.Load(mindPath, worldID)
}

// SaveSession persists the given session to the agent's sessions directory,
// creating the file if it does not exist.
func SaveSession(mindPath string, s *Session) error {
	return session.Save(mindPath, s)
}

// ListSessions returns all sessions from the agent's sessions directory.
func ListSessions(mindPath string) ([]Session, error) {
	return session.List(mindPath)
}

// --- Evolution operations ---

// Dream analyzes the named agent's recent journal entries and promotes
// successful strategies into playbooks/auto-reflexion.md.
func Dream(name string) (*ReflexionResult, error) {
	mindPath := AgentDir(name)
	return evolution.Dream(mindPath)
}

// Reflect is a backward-compatible alias for Dream.
func Reflect(name string) (*ReflexionResult, error) {
	return Dream(name)
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
