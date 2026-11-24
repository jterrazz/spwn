package architecture_test

import (
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// invariant: no spec reaches an ssh agent. ssh reads the passwd home's
// ~/.ssh/config whatever HOME says, so a spec's ssh that reaches the owner's
// agent loads its throwaway keys into it through AddKeysToAgent. The
// Makefile unexports SSH_AUTH_SOCK for every recipe; this is the gate that
// holds it, and it fails a run whose environment names any socket.
func agentsReachable(environ []string, isSocket func(string) bool) []string {
	var out []string
	for _, kv := range environ {
		name, value, _ := strings.Cut(kv, "=")
		if name == "SSH_AUTH_SOCK" && value != "" {
			out = append(out, kv)
			continue
		}
		if filepath.IsAbs(value) && isSocket(value) {
			out = append(out, kv)
		}
	}
	return out
}

func isSocketFile(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.Mode()&fs.ModeSocket != 0
}

func TestNoSpecReachesAnSSHAgent(t *testing.T) {
	if found := agentsReachable(os.Environ(), isSocketFile); len(found) > 0 {
		t.Errorf("the specs run with an ssh agent reachable, %q: run them through make, which unexports SSH_AUTH_SOCK", found)
	}
}

func TestTheAgentRuleBites(t *testing.T) {
	sock := "/tmp/arch-agent-" + strings.ReplaceAll(t.Name(), "/", "-")
	_ = os.Remove(sock)
	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close(); _ = os.Remove(sock) })
	for _, env := range [][]string{{"SSH_AUTH_SOCK=/private/tmp/com.apple.launchd.x/Listeners"}, {"STATION_AGENT=" + sock}} {
		if found := agentsReachable(env, isSocketFile); len(found) != 1 {
			t.Errorf("an environment of %q passed the agent rule", env)
		}
	}
	if found := agentsReachable([]string{"HOME=/tmp", "SSH_AUTH_SOCK="}, isSocketFile); len(found) != 0 {
		t.Errorf("an environment with no agent failed the agent rule: %q", found)
	}
}
