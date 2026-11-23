package guest_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/ports/adapter/guest"
)

func hello(t *testing.T, before string) string {
	t.Helper()
	top := t.TempDir()
	home := filepath.Join(top, "ci")
	file := filepath.Join(home, ".ssh", "authorized_keys")
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	answers, done := run(ctx, t, top, home, `{"v":1,"type":"hello","keys":["ssh-ed25519 AAAAmac host"]}`, `{"v":1,"type":"ping"}`)
	until(answers, `"pong"`)
	stop()
	<-done
	data, _ := os.ReadFile(file)
	return string(data)
}

func TestTheGuestKeepsADamagedBlockAsItIs(t *testing.T) {
	damaged := "ssh-ed25519 AAAAown laptop\n" + guest.KeysBegin + "\nssh-ed25519 AAAAx a\n" + guest.KeysBegin + "\nssh-ed25519 AAAAy b\n" + guest.KeysEnd + "\n"
	if got := hello(t, damaged); got != damaged {
		t.Fatalf("authorized_keys after a hello over a damaged block = %q; want it as it was", got)
	}
}

func TestTheGuestKeepsAFileLargerThanAMebibyteAsItIs(t *testing.T) {
	big := strings.Repeat("#"+strings.Repeat("x", 1023)+"\n", 1025)
	if got := hello(t, big); got != big {
		t.Fatalf("authorized_keys of %d bytes after a hello = %d bytes; want it as it was", len(big), len(got))
	}
}
