package localtest

import (
	"context"
	"fmt"
	"time"

	"github.com/jterrazz/studio-station/src/platform/local"
)

type Env struct {
	Exe, HomeDir, Login, LoginShell, Mac, OS, Arch string
	UID                                            int
}

func (e Env) Executable() string { return e.Exe }
func (e Env) Home() string       { return e.HomeDir }
func (e Env) User() string       { return e.Login }
func (e Env) Shell() string      { return e.LoginShell }
func (e Env) Hostname() string   { return e.Mac }
func (e Env) Euid() int          { return e.UID }
func (e Env) GOOS() string       { return e.OS }
func (e Env) GOARCH() string     { return e.Arch }

type Response struct {
	Status int
	Header map[string]string
	Body   []byte
	Err    error
}

type Request struct {
	Method, URL string
	Body        []byte
}

type Net struct {
	Open      map[string]bool
	Responses map[string]Response
	Requests  []Request
	// value: each datagram broadcast, and the hardware address of each IP.
	Broadcasts []Datagram
	Hardware   map[string]string
}

type Datagram struct {
	Port    int
	Payload []byte
}

func NewNet() *Net { return &Net{Open: map[string]bool{}, Responses: map[string]Response{}} }

func (n *Net) Broadcast(_ context.Context, port int, payload []byte) error {
	n.Broadcasts = append(n.Broadcasts, Datagram{Port: port, Payload: append([]byte(nil), payload...)})
	return nil
}

func (n *Net) HardwareAddress(ip string) string { return n.Hardware[ip] }

func (n *Net) Dial(_ context.Context, mac string, port int, _ time.Duration) bool {
	return n.Open[fmt.Sprintf("%s:%d", mac, port)]
}

func (n *Net) answer(method, url string, payload []byte) (Response, error) {
	n.Requests = append(n.Requests, Request{Method: method, URL: url, Body: payload})
	r, ok := n.Responses[method+" "+url]
	if !ok {
		return Response{}, fmt.Errorf("systemtest: unscripted %s %s", method, url)
	}
	return r, r.Err
}

func (n *Net) Get(_ context.Context, url string, _ map[string]string) (status int, body []byte, err error) {
	r, err := n.answer("GET", url, nil)
	return r.Status, r.Body, err
}

func (n *Net) Fetch(_ context.Context, url string, _ map[string]string) (local.Reply, error) {
	r, err := n.answer("GET", url, nil)
	return local.Reply{Status: r.Status, Header: r.Header, Body: r.Body}, err
}

func (n *Net) Post(_ context.Context, url string, _ map[string]string, payload []byte) (status int, body []byte, err error) {
	r, err := n.answer("POST", url, payload)
	return r.Status, r.Body, err
}

func (n *Net) Delete(_ context.Context, url string, _ map[string]string) (status int, body []byte, err error) {
	r, err := n.answer("DELETE", url, nil)
	return r.Status, r.Body, err
}

type Clock struct{ At time.Time }

func (c *Clock) Now() time.Time { return c.At }

func (c *Clock) Sleep(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.At = c.At.Add(d)
	return nil
}

func Mac(r *Runner, f *FS, e Env, n *Net, c *Clock) local.System {
	return local.System{Run: r, FS: f, Env: e, Net: n, Clock: c}
}
