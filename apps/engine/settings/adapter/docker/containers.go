package docker

import (
	"context"
	"errors"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

type Containers struct{ run local.Runner }

func New(run local.Runner) *Containers { return &Containers{run: run} }

func (c *Containers) Containers(ctx context.Context, shortcut string) (settings.ContainersOutcome, error) {
	// value: the steps of the Docker shortcuts' functions, run by v2's run
	// docker for one version: each lists the ids, then removes them.
	var steps []func(context.Context) (settings.ContainersRemoval, error)
	switch shortcut {
	case "dclean":
		steps = append(steps, c.containers)
	case "dclean-images":
		steps = append(steps, c.images)
	case "dreset":
		steps = append(steps, c.containers, c.images)
	default:
		return settings.ContainersOutcome{}, &fail.Said{Message: "no Docker shortcut " + shortcut, Hint: "station settings docker-shortcuts"}
	}
	outcome := settings.ContainersOutcome{Shortcut: shortcut}
	for _, step := range steps {
		removal, err := step(ctx)
		if err != nil {
			return outcome, err
		}
		outcome.Removals = append(outcome.Removals, removal)
	}
	return outcome, nil
}

func (c *Containers) containers(ctx context.Context) (settings.ContainersRemoval, error) {
	return c.removeAll(ctx, "containers", []string{"ps", "-aq"}, []string{"rm", "-vf"})
}

func (c *Containers) images(ctx context.Context) (settings.ContainersRemoval, error) {
	return c.removeAll(ctx, "images", []string{"images", "-aq"}, []string{"rmi", "-f"})
}

func (c *Containers) removeAll(ctx context.Context, what string, list, remove []string) (settings.ContainersRemoval, error) {
	out, err := c.docker(ctx, list)
	if err != nil {
		return settings.ContainersRemoval{}, err
	}
	ids := strings.Fields(out)
	if len(ids) == 0 {
		return settings.ContainersRemoval{What: what}, nil
	}
	if _, err := c.docker(ctx, append(append([]string{}, remove...), ids...)); err != nil {
		return settings.ContainersRemoval{}, err
	}
	return settings.ContainersRemoval{What: what, Count: len(ids)}, nil
}

func (c *Containers) docker(ctx context.Context, args []string) (string, error) {
	out, err := c.run.Output(ctx, local.Cmd{Name: "docker", Args: args})
	if errors.Is(err, local.ErrNotFound) {
		return "", &fail.Said{Message: "docker is not installed", Hint: "station apps install docker"}
	}
	if err != nil {
		return "", err
	}
	if out.ExitCode != 0 {
		message := "docker " + args[0] + " failed"
		if line, _, _ := strings.Cut(strings.TrimSpace(out.Stderr), "\n"); line != "" {
			message += ": " + line
		}
		return "", &fail.Said{Message: message, Hint: "check that Docker runs, then run it again"}
	}
	return out.Stdout, nil
}
