package agents

import (
	"errors"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

type Agent struct {
	Name string
	// value: its one line of purpose, as its agent.yaml says it.
	Description string
	// value: the team it belongs to, by slug; empty for none.
	Team string
	// value: the harness it runs in, one of the catalog's Harnesses; empty
	// for its project's default.
	Harness string
}

type Layer struct {
	Name string
	// value: the files the layer holds, by their path inside it.
	Files []string
}

type Entry struct {
	At      time.Time
	World   string
	Outcome string
	// value: the session's exit code: 0 completed, below 0 destroyed,
	// above 0 failed.
	Code int
	Took time.Duration
}

type Mind struct {
	Agent
	// value: the folder its Mind lives in, on its machine.
	Path   string
	Layers []Layer
	// value: its newest journal entries, newest first.
	Journal []Entry
}

type Edit struct {
	// value: the new value of each field asked, empty to clear it; nil
	// leaves it as it is.
	Team, Harness *string
}

type Forked struct {
	From, To string
	Layers   []string
}

type Exported struct {
	Agent string
	// value: the archive written, as a path on the agent's machine.
	Archive string
}

type Dreamed struct {
	Agent string
	// value: why nothing was analyzed; empty when the dream ran.
	Skipped   string
	Entries   int
	Completed int
	Failed    int
	Destroyed int
	// value: the completed among the completed and the failed, from 0 to 1.
	Rate float64
	// value: the playbook the dream wrote, as a path on the agent's machine.
	Written string
}

type Slept struct {
	Agent    string
	Archived int
	Pruned   int
}

var (
	ErrNotFound       = errors.New("no agent of that name")
	ErrExists         = errors.New("an agent of that name exists")
	ErrBadName        = errors.New("not a name an agent takes")
	ErrUnknownHarness = errors.New("not a harness")
	ErrBadArchive     = errors.New("not an agent's archive")
)

// value: what an agent's name may be, and what an agent's archive's name
// ends with.
const (
	MaxName       = 63
	ArchiveSuffix = ".tar.gz"
)

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

func Blank(name string) Agent {
	// value: an agent as it is made, before the edit its creation asks: the
	// one line of purpose Agents writes into every new agent.yaml, and the
	// default harness.
	return Agent{Name: name, Description: "A blank-slate agent; replace this one-liner with what " + name + " is for.", Harness: catalog.Harnesses()[0]}
}

func Verbs() []string {
	// value: the words under `agents` that are verbs, which no agent may be
	// named, since `agents <name>` is that agent's detail.
	return []string{"create", "edit", "delete", "fork", "export", "import", "dream", "sleep", "talk", "start", "stop", "log", "help"}
}

func ValidName(name string) bool {
	// invariant: a name is lowercase letters, digits and dashes, starts with
	// a letter, is at most MaxName long, and is no verb of the noun.
	return len(name) <= MaxName && namePattern.MatchString(name) && !slices.Contains(Verbs(), name)
}

func (e Edit) Check() error {
	// invariant: an edit names a harness of the catalog's, or clears it.
	if e.Harness != nil && *e.Harness != "" && !slices.Contains(catalog.Harnesses(), *e.Harness) {
		return ErrUnknownHarness
	}
	return nil
}

func (e Edit) Apply(a Agent) Agent {
	// invariant: the edit applied to an agent, its other fields kept.
	if e.Team != nil {
		a.Team = *e.Team
	}
	if e.Harness != nil {
		a.Harness = *e.Harness
	}
	return a
}

func ArchiveName(archive string) string {
	// value: the name an archive gives the agent it holds, when no other is
	// asked: its file's name without ArchiveSuffix.
	return strings.TrimSuffix(filepath.Base(archive), ArchiveSuffix)
}
