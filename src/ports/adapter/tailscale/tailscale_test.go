package tailscale

import (
	"context"
	"encoding/json"
	"io/fs"
	"strings"
	"sync"
	"testing"

	"github.com/jterrazz/studio-station/src/platform/local"
	"github.com/jterrazz/studio-station/src/ports/contract"
	"github.com/jterrazz/studio-station/src/ports/contract/contracttest"
)

const status = `{"BackendState":"Running","CertDomains":["jterrazz-macbook-pro.tail1234.ts.net"],
"Self":{"ID":"n1","HostName":"jterrazz-macbook-pro","DNSName":"jterrazz-macbook-pro.tail1234.ts.net.","TailscaleIPs":["100.98.133.14"],"Online":true},
"Peer":{"k2":{"ID":"n2","HostName":"jterrazz-mac-mini","DNSName":"jterrazz-mac-mini.tail1234.ts.net.","TailscaleIPs":["100.98.133.15"],"Online":true}}}`

type handler struct {
	Proxy string `json:"Proxy,omitempty"`
}

type tcp struct {
	HTTPS      bool   `json:"HTTPS,omitempty"`
	TCPForward string `json:"TCPForward,omitempty"`
}

type web struct {
	Handlers map[string]handler `json:"Handlers"`
}

// value: the CLI as recorded: its status, and a serve config that `serve`
// changes and `serve status --json` prints in the CLI's own shape.
type cli struct {
	mu          sync.Mutex
	TCP         map[string]tcp  `json:"TCP,omitempty"`
	Web         map[string]web  `json:"Web,omitempty"`
	AllowFunnel map[string]bool `json:"AllowFunnel,omitempty"`
}

func (c *cli) LookPath(name string) (string, bool) {
	return "/opt/homebrew/bin/" + name, name == "tailscale"
}

func (c *cli) Output(_ context.Context, cmd local.Cmd) (local.Output, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	args := strings.Join(cmd.Args, " ")
	switch {
	case args == "status --json":
		return local.Output{Stdout: status}, nil
	case args == "serve status --json":
		data, _ := json.Marshal(c)
		return local.Output{Stdout: string(data)}, nil
	case strings.HasPrefix(args, "cert "):
		return local.Output{}, nil
	case strings.HasPrefix(args, "serve "):
		c.serve(cmd.Args[1:])
		return local.Output{}, nil
	case strings.HasPrefix(args, "funnel "):
		port := c.serve(cmd.Args[1:])
		if c.AllowFunnel == nil {
			c.AllowFunnel = map[string]bool{}
		}
		c.AllowFunnel["jterrazz-macbook-pro.tail1234.ts.net:"+port] = cmd.Args[len(cmd.Args)-1] != "off"
		return local.Output{}, nil
	}
	return local.Output{Stderr: "unexpected invocation: " + args, ExitCode: 97}, nil
}

func (c *cli) serve(args []string) (port string) {
	var path, target string
	raw := false
	for _, a := range args {
		switch {
		case strings.HasPrefix(a, "--https="):
			port = strings.TrimPrefix(a, "--https=")
		case strings.HasPrefix(a, "--tcp="):
			port, raw = strings.TrimPrefix(a, "--tcp="), true
		case strings.HasPrefix(a, "--set-path="):
			path = strings.TrimPrefix(a, "--set-path=")
		case a == "--bg":
		default:
			target = a
		}
	}
	host := "jterrazz-macbook-pro.tail1234.ts.net:" + port
	if c.TCP == nil {
		c.TCP, c.Web = map[string]tcp{}, map[string]web{}
	}
	if path == "" {
		path = "/"
	}
	switch {
	case target == "off" && raw:
		delete(c.TCP, port)
	case target == "off":
		delete(c.Web[host].Handlers, path)
		if len(c.Web[host].Handlers) == 0 {
			delete(c.Web, host)
			delete(c.TCP, port)
		}
	case raw:
		c.TCP[port] = tcp{TCPForward: strings.TrimPrefix(target, "tcp://")}
	default:
		c.TCP[port] = tcp{HTTPS: true}
		if c.Web[host].Handlers == nil {
			c.Web[host] = web{Handlers: map[string]handler{}}
		}
		c.Web[host].Handlers[path] = handler{Proxy: target}
	}
	return port
}

type noApp struct{}

func (noApp) Stat(string) (fs.FileInfo, error) { return nil, fs.ErrNotExist }

func TestTailscaleKeepsTheTailnetContract(t *testing.T) {
	contracttest.Tailnet(t, func(*testing.T) contract.Tailnet { return New(&cli{}, noApp{}, "") })
}

func TestTailscaleKeepsTheAudienceContract(t *testing.T) {
	contracttest.Audience(t, func(*testing.T) (contract.Audience, contract.Tailnet) {
		tn := New(&cli{}, noApp{}, "")
		return tn, tn
	})
}

// invariant: a path mounted on a funneled port reads as the private share
// station opened it as, its root entry as public: the owner's own Funnel
// on 443 never makes station's paths read as its Funnel shares.
func TestAMountedPathOnAFunneledPortReadsPrivate(t *testing.T) {
	shares, err := parseServe([]byte(`{"TCP":{"443":{"HTTPS":true}},"Web":{"mac.ts.net:443":{"Handlers":{"/":{"Proxy":"http://127.0.0.1:8080"},"/linux-1/5173":{"Proxy":"http://127.0.0.1:40001"}}}},"AllowFunnel":{"mac.ts.net:443":true}}`))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, s := range shares {
		got[s.Path] = string(s.To)
	}
	if got["/"] != "public" || got["/linux-1/5173"] != "" {
		t.Errorf("audiences by path = %v, want / public and /linux-1/5173 private", got)
	}
}
