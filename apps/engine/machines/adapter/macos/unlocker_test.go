package macos

import (
	"io/fs"
	"slices"
	"sync"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machines/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

const knownHosts = "/state/known_hosts"

type state int

const (
	answering state = iota
	atPreBoot
	offline
	refusing
)

type dirs struct{}

func (dirs) MkdirAll(string, fs.FileMode) error { return nil }

// invariant: ssh as a Mac answers it: the key check passes on a running Mac
// and is denied at pre-boot, where only its LAN address answers, an offline Mac exits 255 to everything, the
// password session ends by the Mac dropping it, and its sudo may refuse.
type macs struct {
	t                  *testing.T
	mu                 sync.Mutex
	states             map[string]state
	lans               map[string]string
	refusal            string
	sessions, restarts map[string]int
}

func (m *macs) set(mac string, s state) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states[mac] = s
}

func (m *macs) count(counts map[string]int, mac string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return counts[mac]
}

func (m *macs) ssh(c local.Cmd) (local.Output, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	last := c.Args[len(c.Args)-1]
	mac := last
	if last == "true" || last == restartLine {
		mac = c.Args[len(c.Args)-2]
	}
	st := m.states[mac]
	if st == offline || st == atPreBoot && !slices.Contains(c.Args, "HostName="+m.lans[mac]) {
		return local.Output{ExitCode: 255}, nil
	}
	switch {
	case last == "true":
		if !slices.Contains(c.Args, "UserKnownHostsFile="+knownHosts+"/"+mac+".preboot") {
			m.t.Errorf("the key check of %s used another known-hosts file: %v", mac, c.Args)
		}
		if st == atPreBoot {
			return local.Output{ExitCode: 255, Stderr: mac + ": Permission denied (publickey,password)."}, nil
		}
		return local.Output{}, nil
	case slices.Contains(c.Args, "PreferredAuthentications=password"):
		m.sessions[mac]++
		return local.Output{ExitCode: 255}, nil
	case last == restartLine:
		if st == refusing {
			return local.Output{ExitCode: 1, Stderr: m.refusal + "\n"}, nil
		}
		m.restarts[mac]++
		return local.Output{ExitCode: 255}, nil
	}
	m.t.Errorf("unexpected ssh %v", c.Args)
	return local.Output{ExitCode: 97}, nil
}

func TestTheMacOSUnlockerKeepsTheContract(t *testing.T) {
	contracttest.Unlocker(t, func(t *testing.T) contracttest.UnlockerSubject {
		m := &macs{t: t, states: map[string]state{}, lans: map[string]string{}, sessions: map[string]int{}, restarts: map[string]int{}}
		return contracttest.UnlockerSubject{
			Unlocker: New(local.Played(m.ssh, "ssh"), dirs{}, knownHosts),
			AtPreBoot: func(mac, lan string) {
				m.set(mac, atPreBoot)
				m.mu.Lock()
				defer m.mu.Unlock()
				m.lans[mac] = lan
			},
			Offline:   func(mac string) { m.set(mac, offline) },
			Answering: func(mac string) { m.set(mac, answering) },
			Refusing: func(mac, detail string) {
				m.set(mac, refusing)
				m.refusal = detail
			},
			Sessions: func(mac string) int { return m.count(m.sessions, mac) },
			Restarts: func(mac string) int { return m.count(m.restarts, mac) },
		}
	})
}
