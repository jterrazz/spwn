package wire

import (
	"context"
	"errors"
	"math"
	"path/filepath"
	"slices"
	"time"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/src/core/call"
	coremachine "github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/engine/process"
	model "github.com/jterrazz/studio-station/src/images"
	"github.com/jterrazz/studio-station/src/images/adapter/oci"
	imagescalls "github.com/jterrazz/studio-station/src/images/calls"
	imagescontract "github.com/jterrazz/studio-station/src/images/contract"
	"github.com/jterrazz/studio-station/src/machinefile"
	machinefilecalls "github.com/jterrazz/studio-station/src/machinefile/calls"
	"github.com/jterrazz/studio-station/src/machines/adapter/openssh"
	"github.com/jterrazz/studio-station/src/machines/contract"
	"github.com/jterrazz/studio-station/src/platform/local"
	"github.com/jterrazz/studio-station/src/remote/each"
	imagescmd "github.com/jterrazz/studio-station/src/surface/cli/images"
	imagesbuild "github.com/jterrazz/studio-station/src/surface/cli/images/build"
	imagesdelete "github.com/jterrazz/studio-station/src/surface/cli/images/delete"
	imagesupdate "github.com/jterrazz/studio-station/src/surface/cli/images/update"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

func (m machine) images() *imagescalls.Builder {
	user := m.sys.Env.User()
	if current, err := m.stores.vmUser().Get(); err == nil && current != "" {
		user = current
	}
	hypervisor := m.hypervisor()
	return imagescalls.NewBuilder(imagesMachine{h: m.sys, keys: openssh.NewKeyGen(m.sys.Run, m.sys.FS)}, imagescalls.Contracts{
		Hypervisor: hypervisor, Builds: m.buildVMs(hypervisor), Store: m.stores.images,
		Source: m.imageSource(hypervisor.Name()), Machines: m.stores.machines, Templates: templates{m},
	}, m.cat, imagescalls.BuilderOptions{
		Cache: m.paths.Cache, Key: filepath.Join(m.paths.State, "images", "key"),
		Version: Version(),
		User:    user,
		Mac:     m.profile.Ref.Name,
	})
}

// value: where station's images are published: ghcr, the owner's packages.
func (m machine) imageSource(hypervisor string) imagescontract.ImageSource {
	if m.world != nil {
		return m.world.imageSource(hypervisor)
	}
	return oci.New(m.sys.Net, "https://ghcr.io", "jterrazz", hypervisor)
}

func imagesCommand(env kit.Env, m machine) *cobra.Command {
	return imagescmd.Command(imagescmd.Deps{ThisMachine: m.where, Images: m.cat.Images, Every: func(ctx context.Context) ([]model.Report, error) { return m.everyMacImages(ctx, env) }}, env,
		imagesbuild.Command(imagesbuild.Deps{ThisMachine: m.where, Images: m.cat.Images}, env),
		imagesupdate.Command(imagesupdate.Deps{ThisMachine: m.where, Images: m.cat.Images}, env),
		imagesdelete.Command(imagesdelete.Deps{ThisMachine: m.where, Images: m.cat.Images}, env),
	)
}

// value: the Machinefiles images are made from, read from every source
// as plan reads them.
type templates struct{ m machine }

func (t templates) Template(ctx context.Context, name string) (model.Template, error) {
	got, err := machinefilecalls.Template(ctx, t.m.machinefileDeps(nil), name)
	var missing *machinefile.NotFoundError
	var none *machinefile.NotTemplateError
	switch {
	case errors.As(err, &missing) && missing.By == "":
		return model.Template{}, &model.UnknownError{Image: name}
	case errors.As(err, &none):
		return model.Template{}, &model.NotTemplateError{Image: name}
	case err != nil:
		return model.Template{}, machinefilecalls.Worded(err)
	}
	out := model.Template{Name: got.Name, From: got.From, OS: got.OS, Bake: got.Bake, Lock: got.Lock, SHA: got.SHA}
	for _, f := range got.Files {
		out.Files = append(out.Files, model.TemplateFile{Path: f.Path, Body: f.Body})
	}
	return out, nil
}

func (t templates) Names(ctx context.Context) ([]string, error) {
	return machinefilecalls.Templates(ctx, t.m.machinefileDeps(nil))
}

type imageBuilder struct{ *imagescalls.Builder }

func (s imageBuilder) Build(ctx context.Context, names []string, binary string, step func(label string)) ([]model.Record, error) {
	return s.Builder.Build(ctx, model.BuildRequest{Images: names, Binary: binary}, func(st model.Step) { step(st.Label) })
}

func (s imageBuilder) Update(ctx context.Context, names []string, step func(label string)) ([]model.Update, error) {
	return s.Builder.Update(ctx, names, func(st model.Step) { step(st.Label) })
}

// invariant: the images of This Mac, then of every other Mac that answers,
// each Mac's own; a Mac that does not answer has no row.
func (m machine) everyMacImages(ctx context.Context, env kit.Env) ([]model.Report, error) {
	thisMac, err := call.Invoke(ctx, env.Calls, env.Self, imagescalls.List, imagescalls.ListIn{}, nil)
	if err != nil {
		return nil, err
	}
	others := m.machines(false).OtherMacs()
	slices.Sort(others)
	got := each.All(ctx, others, func(ctx context.Context, mac string) ([]model.Report, error) {
		return call.Invoke(ctx, env.Calls, coremachine.Ref{Name: mac}, imagescalls.List, imagescalls.ListIn{}, nil)
	})
	for _, mac := range others {
		if got[mac].Err == nil {
			thisMac = append(thisMac, got[mac].Value...)
		}
	}
	return thisMac, nil
}

type imagesMachine struct {
	h    local.System
	keys openssh.KeyGen
}

func (i imagesMachine) Now() time.Time { return i.h.Clock.Now() }
func (i imagesMachine) Sleep(ctx context.Context, d time.Duration) error {
	return i.h.Clock.Sleep(ctx, d)
}
func (i imagesMachine) Exists(p string) bool {
	_, err := i.h.FS.Stat(p)
	return err == nil
}
func (i imagesMachine) ReadFile(p string) ([]byte, error) { return i.h.FS.ReadFile(p) }
func (i imagesMachine) Keep(p string, data []byte) error {
	if err := i.h.FS.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	return i.h.FS.WriteFile(p, data, 0o755)
}
func (i imagesMachine) Get(ctx context.Context, url string) (status int, body []byte, err error) {
	return i.h.Net.Get(ctx, url, nil)
}
func (i imagesMachine) MakeKey(ctx context.Context, p string) error { return i.keys.MakeKey(ctx, p) }

// invariant: a build VM is the engine's: it boots the clone and holds the
// guest a build speaks to.
func (m machine) buildVMs(hv contract.Hypervisor) imagescalls.BuildVMs {
	port := m.guestPort()
	if port <= 0 || port > math.MaxUint16 {
		port = 22
	}
	return imageBuilds{process.NewBuildVMs(buildHypervisor{hv}, buildGuests{m.guestDialer()}, m.sys.Clock.Sleep, uint16(port))}
}

type imageBuilds struct{ *process.BuildVMs }

func (b imageBuilds) Open(ctx context.Context, vm string, wait time.Duration) (imagescalls.BuildVM, error) {
	v, err := b.BuildVMs.Open(ctx, vm, wait)
	if err != nil {
		return nil, err
	}
	return imageBuild{v}, nil
}

type imageBuild struct{ *process.BuildVM }

func (b imageBuild) Put(ctx context.Context, as imagescalls.Login, path string, data []byte, mode uint32) error {
	return b.BuildVM.Put(ctx, process.BuildLogin(as), path, data, mode)
}

func (b imageBuild) Run(ctx context.Context, as imagescalls.Login, line, stdin string) (imagescalls.Ran, error) {
	ran, err := b.BuildVM.Run(ctx, process.BuildLogin(as), line, stdin)
	return imagescalls.Ran(ran), err
}

// value: the image a create builds first, once the owner said yes.
type createImages struct{ b imageBuilder }

func (c createImages) Build(ctx context.Context, image string, step func(label string)) error {
	_, err := c.b.Build(ctx, []string{image}, "", step)
	return err
}
