package calls

import (
	"context"
	"errors"
	"net/netip"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
)

// value: where the first login leaves station's binary for its script to
// install.
const firstLoginBin = "/tmp/station"

type FirstLogin struct {
	// value: the image's own account, over the VM's known hosts.
	Login contract.GuestTarget
	// value: station's binary, put at Bin, then Line run with its values
	// and its script on Stdin, never in Line.
	Bin, Line string
	Binary    []byte
	Stdin     string
}

func (f FirstLogin) Failed(code int, stderr, stdout string) error {
	// invariant: a first login that ran and exited non-zero fails with its
	// first line, as a recipe's run does.
	if code == 0 {
		return nil
	}
	return errors.New("its first login exited " + strconv.Itoa(code) + ": " + firstLine(cmpOr(stderr, stdout)))
}

func (s *Machinist) GuestTarget(vm string, at netip.AddrPort) contract.GuestTarget {
	// value: the target the engine dials a VM of this Mac by: its user and
	// the image key, its host key pinned in its own known hosts.
	return contract.GuestTarget{VM: vm, Address: at, User: s.VMUser(), Key: s.KeyPath(), KnownHosts: s.KnownHostsPath(vm)}
}

func (s *Machinist) FirstLoginOf(ctx context.Context, t contract.GuestTarget, refused error) (FirstLogin, bool, error) {
	// invariant: a VM of an image station only fetches that refuses the
	// image key, or met on a Mac that holds no image key yet, has never had
	// it: its first login makes the key when it is missing and puts it in as
	// catalog.UpstreamLogin, over the same known hosts. Any other refusal is
	// the dial's own.
	if !errors.Is(refused, contract.ErrLoginRefused) && s.h.Exists(t.Key) {
		return FirstLogin{}, false, nil
	}
	rec, err := s.VM(t.VM, "")
	if err != nil {
		return FirstLogin{}, false, err
	}
	img, _ := s.imageOf(ctx, rec.Image)
	if img.FirstLogin == "" {
		return FirstLogin{}, false, nil
	}
	script, err := catalog.Recipes.ReadFile(img.FirstLogin)
	if err != nil {
		return FirstLogin{}, false, err
	}
	if s.o.GuestBinary == nil {
		return FirstLogin{}, false, errors.New("the first login has no station binary to put in " + t.VM)
	}
	bin, err := s.o.GuestBinary()
	if err != nil {
		return FirstLogin{}, false, err
	}
	pub, err := s.img.ImageKey(ctx)
	if err != nil {
		return FirstLogin{}, false, err
	}
	login := t
	login.User, login.Key, login.Password = catalog.UpstreamLogin.User, "", catalog.UpstreamLogin.Password
	vars := "VM_USER=" + quote(t.User) + "\nIMAGE_KEY=" + quote(strings.TrimSpace(pub)) +
		"\nLOGIN_USER=" + quote(login.User) + "\nLOGIN_PASSWORD=" + quote(login.Password) + "\n"
	return FirstLogin{Login: login, Bin: firstLoginBin, Binary: bin, Line: catalog.AsRoot("-n /bin/bash -s"), Stdin: vars + string(script)}, true, nil
}
