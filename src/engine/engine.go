package engine

import (
	"encoding/json"
	"time"

	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/ports"
)

type Memory map[string]Remembered

type Remembered struct {
	// value: what an engine knows of a VM beyond its record and its run:
	// its state and why, and the runs it started again; a stepped engine
	// hands it from one command to the next, as a real one keeps it between
	// ticks.
	State   string      `json:"state"`
	Detail  string      `json:"detail,omitempty"`
	Reruns  []time.Time `json:"reruns,omitempty"`
	RerunAt time.Time   `json:"rerun_at,omitzero"`
	// value: a server's restarts since this engine started, when its run
	// began, and its health check as last run.
	Restarts model.Restarts    `json:"restarts,omitzero"`
	BootAt   time.Time         `json:"boot_at,omitzero"`
	Check    model.CheckStatus `json:"check,omitzero"`
	// value: its last check runs, the restart it walks, and whether its
	// uptime record holds it down.
	Latency []model.LatencyPoint `json:"latency,omitempty"`
	Restart *model.RestartWalk   `json:"restart,omitempty"`
	Down    bool                 `json:"down,omitempty"`
	// value: why it is down, and its console's last lines then (SPEC B5.5).
	Cause     string   `json:"cause,omitempty"`
	LastLines []string `json:"last_lines,omitempty"`
	// value: what it was last heard listening on, so a successor lists
	// the ports of a VM down or booting again.
	LastHeard []ports.Listener `json:"last_heard,omitempty"`
	// value: the last report of its agents' journal its guest sent, and
	// whether one came: a first report is what the agents were, told of no
	// change.
	Agents   json.RawMessage `json:"agents,omitempty"`
	Reported bool            `json:"reported,omitempty"`
}

type VMAccounts struct {
	Name     string          `json:"name"`
	Guest    string          `json:"guest"`
	Accounts json.RawMessage `json:"accounts,omitempty"`
}

type AccountsView struct {
	VMs []VMAccounts `json:"vms"`
}

type Stopped struct {
	Name     string `json:"name"`
	Lifetime string `json:"lifetime"`
	Result   string `json:"result"`
	Detail   string `json:"detail,omitempty"`
}

const (
	ResultDeleted = "deleted"
	ResultStopped = "stopped"
	ResultFailed  = "failed"
)
