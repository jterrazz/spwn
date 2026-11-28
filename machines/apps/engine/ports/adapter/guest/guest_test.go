package guest_test

import (
	"bufio"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/ports/adapter/guest"
)

// value: the guest run on top with a home of home until ctx ends, told
// lines on its stdin, and the lines it answers.
func run(ctx context.Context, t *testing.T, top, home string, lines ...string) (answers *bufio.Scanner, done chan struct{}) {
	t.Helper()
	sys := local.New([]string{"HOME=" + home}, nil, nil, nil)
	agent := guest.New(sys, guest.Options{State: filepath.Join(top, "state"), PortsEnv: filepath.Join(top, "services.env"), Kind: catalog.Linux})
	in, told := io.Pipe()
	out, said := io.Pipe()
	done = make(chan struct{})
	go func() {
		_ = agent.Run(ctx, in, said)
		_ = said.Close()
		close(done)
	}()
	go func() {
		for _, l := range lines {
			if _, err := told.Write([]byte(l + "\n")); err != nil {
				return
			}
		}
	}()
	return bufio.NewScanner(out), done
}

// value: the answers read up to the first that holds word, or to their
// end when word is empty.
func until(answers *bufio.Scanner, word string) {
	for answers.Scan() {
		if word != "" && strings.Contains(answers.Text(), word) {
			return
		}
	}
}

func TestEachHelloRewritesTheMacTheGuestRunsOn(t *testing.T) {
	top := t.TempDir()
	home := filepath.Join(top, "ci")
	agent := guest.New(local.New([]string{"HOME=" + home}, nil, nil, nil), guest.Options{State: filepath.Join(top, "state")})
	if got := agent.Placement().Mac; got != "" {
		t.Fatalf("before any hello the guest runs on %q, want none", got)
	}
	for _, mac := range []string{"macbook-pro", "studio-laptop"} {
		ctx, cancel := context.WithCancel(context.Background())
		answers, done := run(ctx, t, top, home, `{"v":1,"type":"hello","host":"`+mac+`","vm":"web"}`)
		until(answers, `"ready"`)
		cancel()
		until(answers, "")
		<-done
		if got := agent.Placement(); got.Mac != mac || got.VM != "web" {
			t.Errorf("after a hello from %s the guest says %+v", mac, got)
		}
	}
}

func TestAHolderOfTheSSHFolderKeepsTheGuestOutOfAuthorizedKeys(t *testing.T) {
	top := t.TempDir()
	home := filepath.Join(top, "ci")
	ssh := filepath.Join(home, ".ssh")
	file := filepath.Join(ssh, "authorized_keys")
	if err := os.MkdirAll(ssh, 0o700); err != nil {
		t.Fatal(err)
	}
	own := "ssh-ed25519 AAAAown laptop\n"
	if err := os.WriteFile(file, []byte(own), 0o600); err != nil {
		t.Fatal(err)
	}
	fd, err := unix.Open(ssh, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	hello := `{"v":1,"type":"hello","keys":["ssh-ed25519 AAAAmac host"]}`
	held, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	answers, done := run(held, t, top, home, hello)
	until(answers, "")
	<-done
	if data, _ := os.ReadFile(file); string(data) != own {
		t.Fatalf("authorized_keys with ~/.ssh held = %q; want it as it was", data)
	}
	_ = unix.Close(fd)
	free, stop := context.WithCancel(context.Background())
	defer stop()
	answers, done = run(free, t, top, home, hello, `{"v":1,"type":"ping"}`)
	until(answers, `"pong"`)
	stop()
	<-done
	want := own + guest.KeysBegin + "\nssh-ed25519 AAAAmac host\n" + guest.KeysEnd + "\n"
	if data, _ := os.ReadFile(file); string(data) != want {
		t.Fatalf("authorized_keys once ~/.ssh is free = %q; want %q", data, want)
	}
}
