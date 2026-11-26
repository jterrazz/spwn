package calls

import (
	"context"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/project"
)

// value: every call runs in the process that asks it, on the folder a
// command runs in, an absolute path of this machine: a project is the
// files of a folder, read where they are.

type ShowIn struct {
	Folder string
}

var Show = call.Call[Deps, ShowIn, project.Project]{
	Name: "project.show", Where: call.InProcess, Applies: machine.Anywhere,
	Errors: []fail.Code{NotFound, LinkInTheWay, NotAFile},
	Run: func(ctx context.Context, d Deps, in ShowIn, _ call.Progress) (project.Project, error) {
		root, err := d.Projects.Find(ctx, in.Folder)
		if err != nil {
			return project.Project{}, coded(err, in.Folder)
		}
		p, err := d.Projects.Show(ctx, root)
		return p, coded(err, root)
	},
}

type CreateIn struct {
	Folder string
	Name   string
	// value: the harness its starter agent runs in; empty, the catalog's
	// default.
	Harness string `json:",omitempty"`
	// value: one of the catalog's examples, whose files the project is made
	// of; empty, a blank project. The name then defaults to the example's.
	From string `json:",omitempty"`
}

var Create = call.Call[Deps, CreateIn, project.Project]{
	Name: "project.create", Where: call.InProcess, Applies: machine.Anywhere,
	Errors: []fail.Code{BadName, UnknownHarness, UnknownExample, ExampleDeclaresHarness, Exists},
	Run: func(ctx context.Context, d Deps, in CreateIn, _ call.Progress) (project.Project, error) {
		// invariant: a project is made as a new folder, never over one that
		// exists, and its harness is written in its file, never left to be
		// chosen later.
		if in.From != "" && !slices.Contains(catalog.Examples(), in.From) {
			return project.Project{}, fail.New(fail.Refused, UnknownExample, in.From, map[string]string{"known": strings.Join(catalog.Examples(), ", ")})
		}
		name := in.Name
		if name == "" {
			name = in.From
		}
		if !project.ValidName(name) {
			return project.Project{}, fail.New(fail.Refused, BadName, name, nil)
		}
		if in.From != "" {
			if in.Harness != "" {
				return project.Project{}, fail.New(fail.Refused, ExampleDeclaresHarness, in.From, nil)
			}
			p, err := d.Projects.Create(ctx, in.Folder, name, "", in.From)
			return p, coded(err, filepath.Join(in.Folder, name))
		}
		harness := in.Harness
		if harness == "" {
			harness = catalog.Harnesses()[0]
		}
		if !slices.Contains(catalog.Harnesses(), harness) {
			return project.Project{}, fail.New(fail.Refused, UnknownHarness, harness, map[string]string{"known": strings.Join(catalog.Harnesses(), ", ")})
		}
		p, err := d.Projects.Create(ctx, in.Folder, name, harness, "")
		return p, coded(err, filepath.Join(in.Folder, name))
	},
}

type PlanIn struct {
	Folder string
}

var Plan = call.Call[Deps, PlanIn, project.Plan]{
	// value: what the project's rules find in what it declares, the errors
	// first; it changes nothing, and a plan with an error is refused by the
	// surface that prints it.
	Name: "project.plan", Where: call.InProcess, Applies: machine.Anywhere,
	Errors: []fail.Code{NotFound, Invalid, LinkInTheWay, NotAFile, NotAFolder},
	Run: func(ctx context.Context, d Deps, in PlanIn, _ call.Progress) (project.Plan, error) {
		root, err := d.Projects.Find(ctx, in.Folder)
		if err != nil {
			return project.Plan{}, coded(err, in.Folder)
		}
		p, err := d.Projects.Plan(ctx, root)
		if err != nil || !p.Valid {
			return p, coded(err, root)
		}
		p.Changes, _, _, err = pending(ctx, d, root)
		return p, coded(err, root)
	},
}

type DriftIn struct {
	Folder string
}

var Drift = call.Call[Deps, DriftIn, project.Drift]{
	// value: how what was installed for the project differs from what it
	// declares; a row refuses nothing.
	Name: "project.drift", Where: call.InProcess, Applies: machine.Anywhere,
	Errors: []fail.Code{NotFound, LinkInTheWay, NotAFile},
	Run: func(ctx context.Context, d Deps, in DriftIn, _ call.Progress) (project.Drift, error) {
		root, err := d.Projects.Find(ctx, in.Folder)
		if err != nil {
			return project.Drift{}, coded(err, in.Folder)
		}
		p, err := d.Projects.Drift(ctx, root)
		return p, coded(err, root)
	},
}

func Refused(p project.Plan) error {
	// invariant: a plan with an error is refused once printed, exit 2.
	if p.Valid {
		return nil
	}
	return fail.New(fail.Refused, Invalid, p.Root, map[string]string{"errors": strconv.Itoa(p.Summary.Errors)})
}
