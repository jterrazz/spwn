package calls

import (
	"context"
	"encoding/base64"
	"errors"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
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
// each program the Machinefile grants is granted, as grants.add does, and
// each it signs in to that the workspace is not signed in to is left to
// the owner with its command, since a sign-in is done at the program's
// site; last, the image's drift from the Machinefile is read (hard case 17).
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
	for _, program := range t.Grants {
		g := model.FirstStartGrant{Program: program}
		if s.o.Grant == nil {
			g.Code, g.Subject = accounts.GrantCannotDeliver, name
		} else if err := s.o.Grant(ctx, program, name); err != nil {
			g.Code, g.Subject = s.refusal(err, name)
		}
		b.Grants = append(b.Grants, g)
	}
	b.SignIns = s.signInsLeft(ctx, name, t.SignIns)
	if s.o.Templates != nil {
		b.Drift, _ = s.o.Templates.Drift(ctx, t.Name, version)
	}
	return b
}

// invariant: a grant is refused by its code and subject; an error that
// carries no code is one the grant could not reach the VM through, and
// its own text is kept in the engine's log, so the cause is not lost.
func (s *Machinist) refusal(err error, vm string) (code fail.Code, subject string) {
	var refused *fail.Error
	if errors.As(err, &refused) && refused.Code != "" {
		return refused.Code, refused.Subject
	}
	line := s.h.Now().UTC().Format(time.RFC3339) + " grant to " + vm + ": " + err.Error() + "\n"
	_ = s.h.Append(path.Join(s.o.Logs, "engine.log"), line)
	return accounts.GrantCannotDeliver, vm
}

// invariant: a sign-in whose read fails is left to the owner too: a
// command run on a signed-in program says so.
func (s *Machinist) signInsLeft(ctx context.Context, vm string, programs []string) []model.FirstStartSignIn {
	if len(programs) == 0 {
		return nil
	}
	var signedIn map[string]bool
	if s.o.SignedIn != nil {
		signedIn, _ = s.o.SignedIn(ctx, vm)
	}
	var out []model.FirstStartSignIn
	for _, p := range programs {
		if !signedIn[p] {
			out = append(out, model.FirstStartSignIn{Program: p, Command: "station accounts login " + p + " --on " + vm})
		}
	}
	return out
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
