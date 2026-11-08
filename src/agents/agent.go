package agents

import (
	"regexp"
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
)

type App string

type State string

const (
	Starting State = "starting"
	Working  State = "working"
	Waiting  State = "waiting"
	Finished State = "finished"
	// value: detected without a hook: the agent's process runs, and nothing
	// more is known of it.
	Running State = "running"
)

type Agent struct {
	VM      string `json:"vm,omitempty"`
	App     App    `json:"app"`
	Session string `json:"session"`
	State   State  `json:"state"`
	// value: the terminal it runs in on the VM's terminal host, "t7"; empty
	// for an agent no terminal of the host is known to hold.
	Terminal string `json:"terminal,omitempty"`
	// value: the folder the agent works in, inside the VM; a step's path is
	// said relative to it.
	Folder string `json:"folder,omitempty"`
	Task   string `json:"task,omitempty"`
	// value: the last steps, oldest first, at most catalog.AgentStepsKept.
	Steps   []Step    `json:"steps,omitempty"`
	Ask     *Ask      `json:"ask,omitempty"`
	PR      *PR       `json:"pr,omitempty"`
	Change  *Change   `json:"change,omitempty"`
	Since   time.Time `json:"since,omitzero"`
	Updated time.Time `json:"updated,omitzero"`
	// value: when the session ended and why, as its app said it.
	Ended  time.Time `json:"ended,omitzero"`
	Reason string    `json:"reason,omitempty"`
}

type Step struct {
	At     time.Time         `json:"at"`
	Verb   catalog.AgentVerb `json:"verb"`
	Object string            `json:"object,omitempty"`
	Diff   *Diff             `json:"diff,omitempty"`
	Tests  *Tests            `json:"tests,omitempty"`
}

type Diff struct {
	Plus  int `json:"plus"`
	Minus int `json:"minus"`
}

type Tests struct {
	Pass  int `json:"pass"`
	Total int `json:"total"`
}

type Ask struct {
	Title  string    `json:"title"`
	Detail string    `json:"detail,omitempty"`
	At     time.Time `json:"at"`
}

type PR struct {
	Number int    `json:"number"`
	URL    string `json:"url"`
}

type Change struct {
	Files int `json:"files"`
	Plus  int `json:"plus"`
	Minus int `json:"minus"`
}

func (a Agent) Expired(now time.Time) bool {
	// invariant: a session's journal is kept catalog.AgentKeptFor past its
	// end, and a session that has not ended is kept.
	return !a.Ended.IsZero() && now.Sub(a.Ended) > catalog.AgentKeptFor
}

var plainName = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

func PlainSession(session string) bool {
	// invariant: a session is named by letters, digits, - and _ alone, so
	// its name is a file's and never a path.
	return plainName.MatchString(session)
}
