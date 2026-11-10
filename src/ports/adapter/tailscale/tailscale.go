package tailscale

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/platform/local"
	"github.com/jterrazz/studio-station/src/ports"
)

const (
	queryBudget = 5 * time.Second
	loopback    = "127.0.0.1"
)

type Commands interface {
	Output(ctx context.Context, c local.Cmd) (local.Output, error)
	LookPath(name string) (string, bool)
}

type Files interface {
	Stat(path string) (fs.FileInfo, error)
}

type CLI struct {
	run   Commands
	fs    Files
	root  string
	track string
	self  *warmSelf
}

type warmSelf struct {
	mu   sync.Mutex
	name string
}

func New(run Commands, files Files, root string) *CLI {
	return &CLI{run: run, fs: files, root: root, self: &warmSelf{}}
}

func (t *CLI) Tracked(dir string) *CLI {
	c := *t
	c.track = dir
	return &c
}

func (t *CLI) variant() string {
	if _, err := t.fs.Stat(path.Join("/", t.root, catalog.TailscaleReceipt)); err == nil {
		return ports.VariantAppStore
	}
	if _, err := t.fs.Stat(path.Join("/", t.root, catalog.TailscaleApp)); err == nil {
		return ports.VariantStandalone
	}
	return ""
}

const (
	// value: the app a share rides, by the catalogue's name for it.
	App = "tailscale"
)

func (t *CLI) Present() bool {
	// value: the app is here: its CLI resolves, read from the paths alone.
	return t.client().Path != ""
}

func (t *CLI) AppStoreCLI() string {
	if t.variant() != ports.VariantAppStore {
		return ""
	}
	return path.Join("/", t.root, catalog.TailscaleBundleCLI)
}

func (t *CLI) client() ports.Client {
	// invariant: the CLI is resolved at each call, by absolute path: a launchd
	// agent's PATH holds neither Homebrew's bin nor ~/.local/bin, and the App
	// Store app has no CLI on any PATH, so the app's own paths come first and
	// PATH last.
	c := ports.Client{Variant: t.variant()}
	for _, p := range catalog.TailscaleCLIs {
		if found, ok := t.run.LookPath(path.Join("/", t.root, p)); ok {
			c.Path = found
			return c
		}
	}
	if found, ok := t.run.LookPath("tailscale"); ok {
		c.Path = found
	}
	return c
}

func (t *CLI) call(ctx context.Context, budget time.Duration, args ...string) (local.Output, error) {
	cli := t.client().Path
	if cli == "" {
		return local.Output{}, &ports.Absent{App: "tailscale"}
	}
	// invariant: the App Store bundle's executable answers as a CLI only when
	// TERM, SHLVL or TERM_PROGRAM is set, and launchd sets none: without them it
	// starts its GUI and fails (measured 2026-11-02), so every call carries SHLVL.
	out, err := t.run.Output(ctx, local.Cmd{Name: cli, Args: args, Env: []string{"SHLVL=1"}, Timeout: budget, Track: t.track})
	out.Stdout, out.Stderr = withoutWarnings(out.Stdout), withoutWarnings(out.Stderr)
	return out, err
}

// invariant: a CLI older or newer than its daemon prints "Warning: client
// version ..." before every answer, on either stream: the line says nothing
// about the answer, so nothing reads it.
func withoutWarnings(text string) string {
	if !strings.Contains(text, "Warning:") {
		return text
	}
	lines := strings.SplitAfter(text, "\n")
	kept := lines[:0]
	for _, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), "Warning:") {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "")
}

type rawNode struct {
	ID           string   `json:"ID"`
	UserID       int64    `json:"UserID"`
	MachineName  string   `json:"HostName"`
	DNSName      string   `json:"DNSName"`
	TailscaleIPs []string `json:"TailscaleIPs"`
	Online       bool     `json:"Online"`
	Tags         []string `json:"Tags"`
}

type rawStatus struct {
	BackendState string `json:"BackendState"`
	// value: who the node is logged in as, and in which tailnet: the Accounts
	// page's Tailscale row reads these and nothing else of the user.
	User map[string]struct {
		LoginName string `json:"LoginName"`
	} `json:"User"`
	CurrentTailnet *struct {
		Name string `json:"Name"`
	} `json:"CurrentTailnet"`
	CertDomains []string            `json:"CertDomains"`
	Self        *rawNode            `json:"Self"`
	Peer        map[string]*rawNode `json:"Peer"`
}

func (n *rawNode) name() string {
	if n.MachineName != "" && n.MachineName != "localhost" {
		return n.MachineName
	}
	label, _, _ := strings.Cut(strings.TrimSuffix(n.DNSName, "."), ".")
	return label
}

func (n *rawNode) ip() string {
	for _, ip := range n.TailscaleIPs {
		if !strings.Contains(ip, ":") {
			return ip
		}
	}
	if len(n.TailscaleIPs) == 0 {
		return ""
	}
	return n.TailscaleIPs[0]
}

func (t *CLI) status(ctx context.Context) (rawStatus, error) {
	var st rawStatus
	out, err := t.call(ctx, queryBudget, "status", "--json")
	if err != nil {
		return st, err
	}
	if err := json.Unmarshal([]byte(out.Stdout), &st); err != nil {
		if out.ExitCode != 0 {
			return st, fmt.Errorf("tailscale status exited %d", out.ExitCode)
		}
		return st, fmt.Errorf("tailscale status: %w", err)
	}
	return st, nil
}

func peersOf(st rawStatus) []ports.Node {
	peers := make([]ports.Node, 0, len(st.Peer))
	for _, n := range st.Peer {
		if n == nil {
			continue
		}
		name := n.name()
		if name == "" {
			name = "unknown"
		}
		peers = append(peers, ports.Node{ID: n.ID, Name: name, IP: n.ip(), Online: n.Online, Tags: n.Tags})
	}
	sort.Slice(peers, func(i, j int) bool {
		if peers[i].Online != peers[j].Online {
			return peers[i].Online
		}
		return peers[i].Name < peers[j].Name
	})
	return peers
}

func sharingOf(st rawStatus) ports.SharingState {
	s := ports.SharingState{Backend: st.BackendState, On: st.BackendState == ports.Running && len(st.CertDomains) > 0}
	if st.Self != nil {
		s.DNSName = strings.TrimSuffix(st.Self.DNSName, ".")
	}
	return s
}

func (t *CLI) Self(ctx context.Context) (ports.Self, error) {
	out := ports.Self{Client: t.client()}
	st, err := t.status(ctx)
	if err != nil {
		return out, err
	}
	out.Sharing = sharingOf(st)
	t.remember(out.Sharing)
	out.Online = st.BackendState == ports.Running
	if st.Self != nil {
		out.ID, out.Name, out.IP, out.Tags = st.Self.ID, st.Self.name(), st.Self.ip(), st.Self.Tags
		out.DNSName = out.Sharing.DNSName
		if u, ok := st.User[strconv.FormatInt(st.Self.UserID, 10)]; ok {
			out.Login = u.LoginName
		}
	}
	if st.CurrentTailnet != nil {
		out.Network = st.CurrentTailnet.Name
	}
	out.Peers = peersOf(st)
	return out, nil
}

func (t *CLI) Ready(ctx context.Context) (ports.SharingState, error) {
	st, err := t.status(ctx)
	if err != nil {
		return ports.SharingState{}, err
	}
	s := sharingOf(st)
	t.remember(s)
	return s, nil
}

func (t *CLI) remember(s ports.SharingState) {
	t.self.mu.Lock()
	t.self.name = s.DNSName
	t.self.mu.Unlock()
}

func target(s ports.Share) string {
	from := loopback + ":" + strconv.Itoa(s.Reserved)
	if s.Scheme == ports.SchemeTCP {
		return "tcp://" + from
	}
	return "http://" + from
}

func where(s ports.Share) []string {
	port := strconv.Itoa(s.Port)
	if s.Scheme == ports.SchemeTCP {
		return []string{"--tcp=" + port}
	}
	if s.Mounted() {
		return []string{"--https=" + port, "--set-path=" + s.Path}
	}
	return []string{"--https=" + port}
}

func (t *CLI) Share(ctx context.Context, s ports.Share) (string, error) {
	args := append(append([]string{"--bg"}, where(s)...), target(s))
	if err := t.serve(ctx, args...); err != nil {
		return "", err
	}
	t.self.mu.Lock()
	name := t.self.name
	t.self.mu.Unlock()
	switch {
	case name == "":
		return "", nil
	case s.Scheme == ports.SchemeTCP:
		return name + ":" + strconv.Itoa(s.Port), nil
	case s.Mounted():
		return "https://" + name + s.Path + "/", nil
	}
	return "https://" + name + ":" + strconv.Itoa(s.Port), nil
}

func (t *CLI) Unshare(ctx context.Context, s ports.Share) error {
	return t.serve(ctx, append(where(s), "off")...)
}

func (t *CLI) serve(ctx context.Context, args ...string) error {
	// invariant: a serve that never returns (the app waits for the admin to
	// enable Serve) is refused with the line it printed, not with the timeout.
	out, err := t.call(ctx, queryBudget, append([]string{"serve"}, args...)...)
	if errors.Is(err, local.ErrTimeout) {
		if detail := firstLine(out.Stderr + "\n" + out.Stdout); detail != "" {
			return refused(detail)
		}
	}
	if err != nil {
		return err
	}
	if out.ExitCode != 0 {
		detail := firstLine(out.Stderr + "\n" + out.Stdout)
		if detail == "" {
			detail = "exit " + strconv.Itoa(out.ExitCode)
		}
		return refused(detail)
	}
	return nil
}

func refused(detail string) error { return &ports.Refused{By: "tailscale serve", Detail: detail} }

func (t *CLI) Warm(ctx context.Context) error {
	t.self.mu.Lock()
	name := t.self.name
	t.self.mu.Unlock()
	if name == "" {
		return nil
	}
	// invariant: both outputs are stdout, thrown away: the certificate stays in
	// tailscaled's own store, so the first https request after it is answered
	// at once. /dev/null is refused, as an existing file that is not a regular
	// one (measured 2026-11-02).
	out, err := t.call(ctx, 0, "cert", "--cert-file=-", "--key-file=-", name)
	if err != nil {
		return err
	}
	if out.ExitCode != 0 {
		detail := firstLine(out.Stderr + "\n" + out.Stdout)
		if detail == "" {
			detail = "exit " + strconv.Itoa(out.ExitCode)
		}
		return errors.New(detail)
	}
	return nil
}

type rawServe struct {
	TCP map[string]struct {
		HTTPS      bool   `json:"HTTPS"`
		TCPForward string `json:"TCPForward"`
	} `json:"TCP"`
	Web map[string]struct {
		Handlers map[string]struct {
			Proxy string `json:"Proxy"`
		} `json:"Handlers"`
	} `json:"Web"`
	AllowFunnel map[string]bool `json:"AllowFunnel"`
}

func (t *CLI) Shared(ctx context.Context) ([]ports.Share, error) {
	out, err := t.call(ctx, queryBudget, "serve", "status", "--json")
	if err != nil {
		return nil, err
	}
	if out.ExitCode != 0 {
		return nil, refused(firstLine(out.Stderr))
	}
	return parseServe([]byte(out.Stdout))
}

func (t *CLI) Funneled(ctx context.Context) ([]int, error) {
	out, err := t.call(ctx, queryBudget, "serve", "status", "--json")
	if err != nil {
		return nil, err
	}
	if out.ExitCode != 0 {
		return nil, refused(firstLine(out.Stderr))
	}
	return parseFunnel([]byte(out.Stdout))
}

// invariant: a port is funneled when the config allows Funnel on it for
// any of the node's names ("host:443": true).
func parseFunnel(data []byte) ([]int, error) {
	var raw rawServe
	if strings.TrimSpace(string(data)) == "" {
		return nil, nil
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	var open []int
	for addr, on := range raw.AllowFunnel {
		_, portText, _ := strings.Cut(addr, ":")
		if port, err := strconv.Atoi(portText); err == nil && on && !slices.Contains(open, port) {
			open = append(open, port)
		}
	}
	slices.Sort(open)
	return open, nil
}

func parseServe(data []byte) ([]ports.Share, error) {
	var raw rawServe
	if strings.TrimSpace(string(data)) == "" {
		return nil, nil
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	var shares []ports.Share
	seen := map[int]bool{}
	for addr, web := range raw.Web {
		_, portText, _ := strings.Cut(addr, ":")
		port, _ := strconv.Atoi(portText)
		for p, h := range web.Handlers {
			shares = append(shares, ports.Share{Port: port, Path: p, Scheme: ports.SchemeHTTPS, Target: h.Proxy})
			seen[port] = true
		}
	}
	for portText, tcp := range raw.TCP {
		port, err := strconv.Atoi(portText)
		if err != nil || seen[port] {
			continue
		}
		s := ports.Share{Port: port, Scheme: ports.SchemeHTTPS, Target: tcp.TCPForward}
		if tcp.TCPForward != "" {
			s.Scheme = ports.SchemeTCP
		}
		shares = append(shares, s)
	}
	sort.Slice(shares, func(i, j int) bool {
		if shares[i].Port != shares[j].Port {
			return shares[i].Port < shares[j].Port
		}
		return shares[i].Path < shares[j].Path
	})
	return shares, nil
}

func firstLine(text string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(text), "\n")
	return strings.TrimSpace(line)
}
