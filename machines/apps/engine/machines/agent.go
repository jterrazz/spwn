package machines

import (
	"path"
	"strconv"
)

type AgentState string

const (
	AgentServed AgentState = "served"
	AgentHeld   AgentState = "held"
	AgentFailed AgentState = "failed"
)

type Agent struct {
	State      AgentState `json:"state"`
	PID        int        `json:"pid,omitempty"`
	Executable string     `json:"executable,omitempty"`
	Free       string     `json:"free,omitempty"`
	Detail     string     `json:"detail,omitempty"`
}

func (a Agent) Unserved() bool { return a.State == AgentHeld || a.State == AgentFailed }

func (a Agent) Holder() string {
	pid := "pid " + strconv.Itoa(a.PID)
	if a.Executable == "" {
		return pid
	}
	return path.Base(a.Executable) + " (" + pid + ")"
}

func (a Agent) Said(mac string) string {
	if a.State == AgentHeld {
		return mac + "'s ssh agent socket is held by " + a.Holder() + ", not its engine"
	}
	return mac + "'s engine serves no ssh agent: " + a.Detail
}

func (a Agent) Fix() string {
	if a.State == AgentHeld {
		return a.Free
	}
	return "station engine start"
}
