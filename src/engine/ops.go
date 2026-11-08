package engine

import (
	"path"
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
	model "github.com/jterrazz/studio-station/src/machines"
)

const Label = catalog.EngineLabel

type Hello struct {
	Version    string    `json:"version"`
	PID        int       `json:"pid"`
	StartedAt  time.Time `json:"started_at"`
	Executable string    `json:"executable,omitempty"`
	// value: the names of the calls this engine serves, as calls.json lists them.
	Calls []string `json:"calls,omitempty"`
	Page  int      `json:"page,omitempty"`
}

type VMView struct {
	Name   string `json:"name"`
	State  string `json:"state"`
	RunPID int    `json:"run_pid"`
	// value: the process that holds its guest on this Mac, 0 while none does.
	Holder int    `json:"holder,omitempty"`
	Guest  string `json:"guest"`
	IP     string `json:"ip"`
	Detail string `json:"detail"`
	// value: its ssh connection opened, whether station's guest answers on
	// it or not; a server's restarts and health check.
	SSH      bool              `json:"ssh,omitempty"`
	Restarts model.Restarts    `json:"restarts,omitzero"`
	Check    model.CheckStatus `json:"check,omitzero"`
	// value: when its run began, while it runs.
	BootAt time.Time `json:"boot_at,omitzero"`
	// value: a server's last check runs (SPEC B5.4), and the health step its
	// restart runs now, nil when none runs (B5.11).
	Latency []model.LatencyPoint `json:"latency,omitempty"`
	Restart *model.RestartWalk   `json:"restart,omitempty"`
	// value: why a server is down, and the last lines its console kept
	// then (SPEC B5.5); empty while it is up.
	Cause     string   `json:"cause,omitempty"`
	LastLines []string `json:"last_lines,omitempty"`
	// value: when the create that made it made it ready (SPEC B5.10).
	ReadyAt time.Time `json:"ready_at,omitzero"`
	// value: the agent its guest last reported it said by (SPEC B5.1).
	Agent *model.AgentView `json:"agent,omitempty"`
}

func SocketPath(state string) string { return path.Join(state, "engine.sock") }

func LogPath(state string) string { return path.Join(state, "logs", "engine.log") }

func ChildrenDir(state string) string { return path.Join(state, "children") }

type AwakeIn struct {
	Lid     bool `json:"lid,omitempty"`
	Battery bool `json:"battery,omitempty"`
	// value: seconds.
	For int    `json:"for,omitempty"`
	By  string `json:"by,omitempty"`
	// value: the end asked (until 17:00), which wins over For.
	Until *time.Time `json:"until,omitempty"`
}
