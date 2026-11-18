package calls

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
)

const (
	firstbootWait = 5 * time.Minute
	ipWait        = 60 * time.Second
	sshWait       = 30 * time.Second
)

func renderFirstboot(script, name string, l model.Lifetime) string {
	return strings.NewReplacer("{{name}}", name, "{{lifetime}}", string(l)).Replace(script)
}

func (s *Machinist) firstBoot(ctx context.Context, img catalog.Image, name string, l model.Lifetime) error {
	if img.Firstboot == "" {
		// invariant: an image station only fetches carries its own setup:
		// no first boot runs inside it.
		return nil
	}
	script, err := catalog.Recipes.ReadFile(img.Firstboot)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, firstbootWait)
	defer cancel()
	out, err := s.engineAPI.GuestExec(ctx, name, catalog.AsRoot("bash -s"), renderFirstboot(string(script), name, l))
	switch {
	case err != nil:
		return err
	case out.ExitCode != 0:
		return errors.New("the first boot exited " + strconv.Itoa(out.ExitCode) + ": " + firstLine(out.Stderr))
	}
	return nil
}
