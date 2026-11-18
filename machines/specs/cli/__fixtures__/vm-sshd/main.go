package main

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// invariant: the VM a spec talks to: an sshd that accepts any key or password, offers an
// ed25519 and an ecdsa host key as OpenSSH does (the ed25519 one written to the
// document's vm/hostkey.ed25519), runs each exec under the document's vm/ folder, and answers every direct-tcpip itself,
// so no real listener exists on the runner; each exec and channel is recorded in
// the document's .calls.
var mu sync.Mutex

func workdir() string { return os.Getenv("TEST_WORKDIR") }

func record(line string) {
	mu.Lock()
	defer mu.Unlock()
	f, err := os.OpenFile(filepath.Join(workdir(), ".calls"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	_, _ = f.WriteString(line + "\n")
}

func main() {
	_, edKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	cfg := &ssh.ServerConfig{
		PublicKeyCallback: func(ssh.ConnMetadata, ssh.PublicKey) (*ssh.Permissions, error) { return nil, nil },
		PasswordCallback:  func(ssh.ConnMetadata, []byte) (*ssh.Permissions, error) { return nil, nil },
	}
	for _, key := range []any{edKey, ecKey} {
		signer, err := ssh.NewSignerFromKey(key)
		if err != nil {
			panic(err)
		}
		cfg.AddHostKey(signer)
		if signer.PublicKey().Type() == ssh.KeyAlgoED25519 {
			_ = os.MkdirAll(filepath.Join(workdir(), "vm"), 0o700)
			_ = os.WriteFile(filepath.Join(workdir(), "vm", "hostkey.ed25519"), ssh.MarshalAuthorizedKey(signer.PublicKey()), 0o600)
		}
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	_, _ = os.Stdout.WriteString("listening on port " + strconv.Itoa(l.Addr().(*net.TCPAddr).Port) + "\n")
	for {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		go serve(conn, cfg)
	}
}

func serve(conn net.Conn, cfg *ssh.ServerConfig) {
	if data, err := os.ReadFile(filepath.Join(workdir(), "vm", "boot-seconds")); err == nil {
		if n, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64); err == nil {
			time.Sleep(time.Duration(n * float64(time.Second)))
		}
	}
	_, chans, reqs, err := ssh.NewServerConn(conn, cfg)
	if err != nil {
		return
	}
	go ssh.DiscardRequests(reqs)
	for nc := range chans {
		switch nc.ChannelType() {
		case "session":
			go session(nc)
		case "direct-tcpip":
			go forward(nc)
		default:
			_ = nc.Reject(ssh.UnknownChannelType, "no")
		}
	}
}

func session(nc ssh.NewChannel) {
	ch, reqs, err := nc.Accept()
	if err != nil {
		return
	}
	defer func() { _ = ch.Close() }()
	for req := range reqs {
		if req.Type != "exec" {
			_ = req.Reply(false, nil)
			continue
		}
		_ = req.Reply(true, nil)
		command := string(req.Payload[4:])
		record("vm-sshd exec " + command)
		code := run(inVM(command), ch)
		status := make([]byte, 4)
		binary.BigEndian.PutUint32(status, uint32(code))
		_, _ = ch.SendRequest("exit-status", false, status)
		return
	}
}

// invariant: a command names the VM's own /tmp, which is the document's vm/tmp,
// so nothing a spec copies in reaches the runner's /tmp.
func inVM(command string) string {
	command = strings.Replace(command, "/usr/local/bin/station", "station", 1)
	tmp := filepath.Join(workdir(), "vm", "tmp")
	if strings.Contains(command, "'/tmp/") {
		_ = os.MkdirAll(tmp, 0o700)
	}
	return strings.ReplaceAll(command, "'/tmp/", "'"+tmp+"/")
}

func run(command string, ch ssh.Channel) int {
	vm := filepath.Join(workdir(), "vm")
	cmd := exec.Command("/bin/sh", "-c", command)
	cmd.Env = []string{"HOME=" + vm, "PATH=" + filepath.Join(vm, "bin"), "TEST_WORKDIR=" + workdir(), "STATION=" + os.Getenv("STATION_TEST_BIN")}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = ch, ch, ch.Stderr()
	err := cmd.Run()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode()
	}
	if err != nil {
		return 127
	}
	return 0
}

func bannerPorts() map[string]bool {
	out := map[string]bool{}
	data, err := os.ReadFile(filepath.Join(workdir(), "vm", "banner-ports"))
	if err != nil {
		return out
	}
	for _, f := range strings.Fields(string(data)) {
		out[f] = true
	}
	return out
}

func forward(nc ssh.NewChannel) {
	var target struct {
		Host       string
		Port       uint32
		OriginHost string
		OriginPort uint32
	}
	if err := ssh.Unmarshal(nc.ExtraData(), &target); err != nil {
		_ = nc.Reject(ssh.ConnectionFailed, "bad request")
		return
	}
	port := strconv.Itoa(int(target.Port))
	record("vm-sshd direct-tcpip " + net.JoinHostPort(target.Host, port))
	ch, reqs, err := nc.Accept()
	if err != nil {
		return
	}
	go ssh.DiscardRequests(reqs)
	defer func() { _ = ch.Close() }()
	if bannerPorts()[port] {
		_, _ = fmt.Fprintf(ch, "station-test %s\n", port)
		_, _ = io.Copy(io.Discard, ch)
		return
	}
	line, err := bufio.NewReader(ch).ReadString('\n')
	if err != nil && line == "" {
		return
	}
	if strings.HasPrefix(line, "HEAD ") || strings.HasPrefix(line, "GET ") {
		_, _ = fmt.Fprintf(ch, "HTTP/1.0 200 OK\r\n\r\nstation-test %s\n", port)
		return
	}
	_, _ = fmt.Fprintf(ch, "station-test %s\n", port)
}
