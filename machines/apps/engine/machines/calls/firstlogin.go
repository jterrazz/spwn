package calls

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/netip"
	"regexp"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
)

var lockdownFolder = regexp.MustCompile(`^/tmp/station\.[A-Za-z0-9]+$`)

type Lockdown struct {
	// value: the image's own account, over the VM's known hosts: the login
	// a lockdown that never ran is run by.
	Login contract.GuestTarget
	// value: station's binary and the script, put in the folder
	// catalog.LockdownFolder makes, and the values the script reads on its
	// stdin.
	Binary, Script []byte
	Values         string
}

func (l Lockdown) Folder(made contract.GuestResult) (string, error) {
	dir := strings.TrimSpace(made.Stdout)
	if made.Code != 0 || !lockdownFolder.MatchString(dir) {
		return "", errors.New("its lockdown found no private folder: " + firstLine(cmpOr(made.Stderr, dir)))
	}
	return dir, nil
}

func (l Lockdown) Failed(code int, stderr, stdout string) error {
	if code == 0 {
		return nil
	}
	return errors.New("its lockdown exited " + strconv.Itoa(code) + ": " + firstLine(cmpOr(stderr, stdout)))
}

func (s *Machinist) GuestTarget(vm string, at netip.AddrPort) contract.GuestTarget {
	// value: the target the engine dials a VM of this Mac by: its user and
	// the image key, its host key pinned in its own known hosts.
	return contract.GuestTarget{VM: vm, Address: at, User: s.VMUser(), Key: s.KeyPath(), KnownHosts: s.KnownHostsPath(vm)}
}

func (s *Machinist) FirstLoginOf(ctx context.Context, t contract.GuestTarget, refused error) (Lockdown, bool, error) {
	// invariant: a VM of an image station only fetches that refuses the
	// image key, or met on a Mac that holds no image key yet, has never had
	// it: its lockdown runs as catalog.UpstreamLogin, over the same known
	// hosts. Any other refusal is the dial's own.
	if !errors.Is(refused, contract.ErrLoginRefused) && s.h.Exists(t.Key) {
		return Lockdown{}, false, nil
	}
	return s.LockdownOf(ctx, t)
}

func (s *Machinist) lockdownImage(ctx context.Context, vm string) (catalog.Image, bool, error) {
	rec, err := s.VM(vm, "")
	if err != nil {
		return catalog.Image{}, false, err
	}
	img, _ := s.imageOf(ctx, rec.Image)
	return img, img.FirstLogin != "", nil
}

func (s *Machinist) Locks(ctx context.Context, vm string) (bool, error) {
	_, ok, err := s.lockdownImage(ctx, vm)
	return ok, err
}

func (s *Machinist) LockdownOf(ctx context.Context, t contract.GuestTarget) (Lockdown, bool, error) {
	// invariant: the lockdown of a VM whose image carries one, the image
	// key made when it is missing; none for any other VM.
	img, ok, err := s.lockdownImage(ctx, t.VM)
	if err != nil || !ok {
		return Lockdown{}, false, err
	}
	script, err := catalog.Recipes.ReadFile(img.FirstLogin)
	if err != nil {
		return Lockdown{}, false, err
	}
	if s.o.GuestBinary == nil {
		return Lockdown{}, false, errors.New("the lockdown has no station binary to put in " + t.VM)
	}
	bin, err := s.o.GuestBinary()
	if err != nil {
		return Lockdown{}, false, err
	}
	pub, err := s.img.ImageKey(ctx)
	if err != nil {
		return Lockdown{}, false, err
	}
	sum := sha256.Sum256(bin)
	login := t
	login.User, login.Key, login.Password = catalog.UpstreamLogin.User, "", catalog.UpstreamLogin.Password
	values := strings.Join([]string{t.User, strings.TrimSpace(pub), login.User, login.Password, hex.EncodeToString(sum[:])}, "\n") + "\n"
	return Lockdown{Login: login, Binary: bin, Script: script, Values: values}, true, nil
}
