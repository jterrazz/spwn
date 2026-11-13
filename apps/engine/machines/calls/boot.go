package calls

import (
	"context"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
)

// value: the longest a boot phase's apply runs inside a workspace.
const bootWait = catalog.PhaseBound

// value: the folder of the VM user's station files, under its home, and
// the station binary the image carries.
const (
	guestStation    = ".config/station/"
	guestStationBin = "/usr/local/bin/station"
)

// invariant: a workspace made from a Machinefile runs its boot phase at
// its first start, once it answers ssh: the folders it shares with its
// Mac mounted first, then its Machinefile's chain and lock,
// as this Mac reads them now, replace the image's copies in the VM user's
// folder, then `station apply --phase boot` runs as that user, a
// secret named only by its `vault:` reference, which the guest resolves;
// each account the Machinefile shares is shared, as accounts share does;
// last, the image's drift from the Machinefile is read (hard case 17).
// Nothing that fails here fails the create: the workspace is made, and
// its page says what is left to do.
func (s *Machinist) bootPhase(ctx context.Context, name, version string, t model.Template) *model.FirstStart {
	b := &model.FirstStart{Template: t.Name}
	if err := s.mount(ctx, name, t.Mounts); err != nil {
		b.Failed = firstLine(err.Error())
	} else if err := s.putTemplate(ctx, name, t.Files); err != nil {
		b.Failed = firstLine(err.Error())
	} else if err := s.applyBoot(ctx, name, t); err != nil {
		b.Failed = firstLine(err.Error())
	}
	for _, sh := range t.Shares {
		share := model.FirstStartShare{Account: sh.Account, For: sh.For}
		if s.o.Share == nil {
			share.Failed = "this Mac shares no account"
		} else if err := s.o.Share(ctx, sh.Account, name, sh.For); err != nil {
			share.Failed = firstLine(err.Error())
		}
		b.Shares = append(b.Shares, share)
	}
	if s.o.Templates != nil {
		b.Drift, _ = s.o.Templates.Drift(ctx, t.Name, version)
	}
	return b
}

// invariant: each folder the VM shares with its Mac is mounted at its
// guest folder before the boot phase runs, so its lines may read it; the
// engine brings the mounts to the Machinefile's again at every later
// start (model.MountLine).
func (s *Machinist) mount(ctx context.Context, name string, mounts []model.Mount) error {
	if len(mounts) == 0 {
		return nil
	}
	line, script := model.MountLine(mounts)
	out, err := s.engineAPI.GuestExec(ctx, name, line, script)
	switch {
	case err != nil:
		return err
	case out.ExitCode != 0:
		return errors.New("its folders could not be mounted: " + firstLine(cmpOr(out.Stderr, out.Stdout)))
	}
	return nil
}

func (s *Machinist) putTemplate(ctx context.Context, name string, files []model.TemplateFile) error {
	var script strings.Builder
	script.WriteString("set -e\n")
	for _, f := range files {
		at := "$HOME/" + guestStation + f.Path
		script.WriteString("mkdir -p \"$(dirname \"" + at + "\")\"\n")
		script.WriteString("printf %s " + base64.StdEncoding.EncodeToString(f.Body) + " | base64 -d > \"" + at + "\"\n")
		script.WriteString("chmod 600 \"" + at + "\"\n")
	}
	out, err := s.engineAPI.GuestExec(ctx, name, "bash -s", script.String())
	switch {
	case err != nil:
		return err
	case out.ExitCode != 0:
		return errors.New("its Machinefile could not be written: " + firstLine(out.Stderr))
	}
	return nil
}

func (s *Machinist) applyBoot(ctx context.Context, name string, t model.Template) error {
	ctx, cancel := context.WithTimeout(ctx, bootWait)
	defer cancel()
	out, err := s.engineAPI.GuestExec(ctx, name, bootLine(t), "")
	switch {
	case err != nil:
		return err
	case out.ExitCode != 0:
		return errors.New("the boot phase exited " + strconv.Itoa(out.ExitCode) + ": " + firstLine(cmpOr(out.Stderr, out.Stdout)))
	}
	return nil
}

// value: the line a workspace's boot phase runs as its user: station's
// apply of its Machinefile, its boot phase alone, the owner's run.boot
// lines last.
func bootLine(t model.Template) string {
	return guestStationBin + " apply --machinefile " + quote(t.Name) + " --phase boot </dev/null"
}

func quote(v string) string { return "'" + strings.ReplaceAll(v, "'", `'\''`) + "'" }

func cmpOr(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
