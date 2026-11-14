package machines

import (
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	StepCheck = "Check passes"
	// value: SPEC §7.2: a check runs after every boot, then every minute,
	// and answers in 5 s or fails.
	CheckEvery  = 60 * time.Second
	CheckBudget = 5 * time.Second
)

type HealthCheck struct {
	// value: one of the two: an http(s) URL that answers 2xx or 3xx, or a
	// host:port that accepts a connection.
	URL *url.URL
	TCP string
}

func ParseCheck(check string) (HealthCheck, bool) {
	// invariant: a check is an http(s) URL with a host, or host:port with a
	// port number (SPEC §7.2).
	if strings.HasPrefix(check, "https://") || strings.HasPrefix(check, "http://") {
		u, err := url.Parse(check)
		if err != nil || u.Hostname() == "" {
			return HealthCheck{}, false
		}
		return HealthCheck{URL: u}, true
	}
	name, port, ok := splitAddress(check)
	n, perr := strconv.Atoi(port)
	if !ok || name == "" || perr != nil || n < 1 || n > 65535 {
		return HealthCheck{}, false
	}
	return HealthCheck{TCP: check}, true
}

type CheckTarget struct {
	// value: where the probe connects (host:port), the URL it asks when
	// the check is one, the name a TLS certificate must hold, and whether
	// it is verified.
	Dial   string
	URL    string
	Name   string
	Verify bool
}

func (c HealthCheck) Target(vm string, addr netip.Addr) CheckTarget {
	// invariant: a check whose hostname is the VM's own name reaches the
	// VM's address and accepts the VM's own certificate (a cluster's); any
	// other name is resolved by the probe and its certificate verified.
	name, port := "", ""
	if c.URL != nil {
		name, port = c.URL.Hostname(), c.URL.Port()
		if port == "" {
			port = "80"
			if c.URL.Scheme == "https" {
				port = "443"
			}
		}
	} else {
		name, port, _ = splitAddress(c.TCP)
	}
	own := strings.EqualFold(name, vm) && addr.IsValid()
	dial := name
	if own {
		dial = addr.String()
	}
	t := CheckTarget{Dial: joinAddress(dial, port), Name: name, Verify: !own}
	if c.URL != nil {
		t.URL = c.URL.String()
	}
	return t
}

func splitAddress(s string) (name, port string, ok bool) {
	i := strings.LastIndex(s, ":")
	if i < 0 {
		return "", "", false
	}
	name, port = s[:i], s[i+1:]
	if strings.HasPrefix(name, "[") && strings.HasSuffix(name, "]") {
		name = name[1 : len(name)-1]
	} else if strings.Contains(name, ":") {
		return "", "", false
	}
	return name, port, true
}

func joinAddress(name, port string) string {
	if strings.Contains(name, ":") {
		return "[" + name + "]:" + port
	}
	return name + ":" + port
}

type CheckAnswer struct {
	// value: what it answered: a URL's status (`200`), empty for a
	// connection; and how long the answer took.
	Status string
	Took   time.Duration
}

type CheckState string

const (
	CheckWaiting CheckState = "waiting"
	CheckPassing CheckState = "passing"
	CheckFailing CheckState = "failing"
)

func (CheckState) Enum() []string {
	return []string{string(CheckWaiting), string(CheckPassing), string(CheckFailing)}
}

type CheckStatus struct {
	State CheckState `json:"state,omitempty"`
	// value: when the failing began; zero otherwise.
	Since time.Time `json:"since,omitzero"`
	// value: when the check last ran.
	At time.Time `json:"at,omitzero"`
	// value: what it answered when it last passed: a URL's status (`200`);
	// empty for a connection.
	Answer string `json:"answer,omitempty"`
}

type CheckChange int

const (
	CheckSame CheckChange = iota
	// value: a check that passed now fails: a finding, never a restart.
	CheckFails
	// value: a failing check passes again: the server is back.
	CheckBack
)

func Health(was CheckStatus, passed bool, now time.Time) (CheckStatus, CheckChange) {
	// invariant: SPEC §7.2: a check that never passed since boot waits, which
	// is not a failure; one that passed then fails is failing since that
	// run, and passing again is the server back. A failing check never
	// restarts a server.
	switch {
	case passed && was.State == CheckFailing:
		return CheckStatus{State: CheckPassing, At: now}, CheckBack
	case passed:
		return CheckStatus{State: CheckPassing, At: now}, CheckSame
	case was.State == CheckPassing:
		return CheckStatus{State: CheckFailing, Since: now, At: now}, CheckFails
	case was.State == CheckFailing:
		return CheckStatus{State: CheckFailing, Since: was.Since, At: now}, CheckSame
	}
	return CheckStatus{State: CheckWaiting, At: now}, CheckSame
}

func (s CheckStatus) Due(now time.Time) bool {
	return s.At.IsZero() || now.Sub(s.At) >= CheckEvery
}
