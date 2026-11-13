package calls

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/images"
	"github.com/jterrazz/studio-station/src/machines/contract"
)

const (
	addressWait = 60 * time.Second
	stopWait    = 120 * time.Second
	recipeWait  = 30 * time.Minute
	bakeWait    = catalog.PhaseBound
	pollEvery   = time.Second
	remoteBin   = "/tmp/station"
)

// value: the base's admin account, reached by its password until the recipe
// installs the image key.
var baseAdmin = Login{User: "admin", Password: "admin"}

func (s *Builder) recipeSHA(img catalog.Image) (string, error) {
	h := sha256.New()
	for _, file := range []string{img.Recipe, img.Bake, img.Cleanup, img.Firstboot} {
		data, err := catalog.Recipes.ReadFile(file)
		if err != nil {
			return "", err
		}
		h.Write(data)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// value: an image to build: the catalog's plumbing for its system and the
// template it is made from.
type wanted struct {
	img catalog.Image
	t   images.Template
}

func (s *Builder) wanted(ctx context.Context, name string) (wanted, error) {
	img, err := s.image(ctx, name)
	if err != nil {
		return wanted{}, err
	}
	if img.Recipe == "" {
		return wanted{}, &images.NoRecipeError{Image: name}
	}
	t, err := s.templates.Template(ctx, img.Machinefile)
	if err != nil {
		return wanted{}, err
	}
	return wanted{img: img, t: t}, nil
}

// invariant: a bare build builds every image missing here, or whose
// recipe, Machinefile chain or lock changed since its newest version.
func (s *Builder) selection(ctx context.Context, names []string) ([]wanted, error) {
	var out []wanted
	if len(names) > 0 {
		for _, name := range names {
			w, err := s.wanted(ctx, name)
			if err != nil {
				return nil, err
			}
			if !slices.ContainsFunc(out, func(o wanted) bool { return o.img.Name == w.img.Name }) {
				out = append(out, w)
			}
		}
		return out, nil
	}
	list, err := s.all(ctx)
	if err != nil {
		return nil, err
	}
	for _, img := range list {
		if img.Recipe == "" {
			continue
		}
		w, err := s.wanted(ctx, img.Name)
		if err != nil {
			return nil, err
		}
		sha, err := s.recipeSHA(img)
		if err != nil {
			return nil, err
		}
		held, err := s.records(img)
		switch {
		case err != nil:
			return nil, err
		case len(held) == 0, held[0].RecipeSHA != sha, held[0].MachinefileSHA != w.t.SHA:
			out = append(out, w)
		}
	}
	return out, nil
}

func (s *Builder) Build(ctx context.Context, r images.BuildRequest, progress func(images.Step)) ([]images.Record, error) {
	wanted, err := s.selection(ctx, r.Images)
	if err != nil || len(wanted) == 0 {
		return nil, err
	}
	if need := s.needsBinary(r.Binary); need != nil {
		return nil, need
	}
	if _, err = s.e.Version(ctx); err != nil {
		return nil, &images.NoEngineError{Hypervisor: s.e.Name(), Cause: err}
	}
	binary, err := s.binary(ctx, r.Binary)
	if err != nil {
		return nil, err
	}
	if progress == nil {
		progress = func(images.Step) {}
	}
	var built []images.Record
	for _, w := range wanted {
		rec, err := s.buildOne(ctx, w, binary, progress)
		if err != nil {
			return built, err
		}
		built = append(built, rec)
	}
	return built, nil
}

func (s *Builder) buildOne(ctx context.Context, w wanted, binary string, progress func(images.Step)) (images.Record, error) {
	img, t := w.img, w.t
	vm := buildPrefix + img.Name
	failed := func(step string, cause error) (images.Record, error) {
		// invariant: a failed build leaves no VM behind; the cleanup survives a cancelled ctx.
		detached := context.WithoutCancel(ctx)
		_ = s.e.Shutdown(detached, vm, 10*time.Second)
		_ = s.e.Delete(detached, vm)
		return images.Record{}, &images.BuildFailedError{Image: img.Name, Step: step, Cause: cause}
	}
	version, err := s.nextVersion(img)
	if err != nil {
		return images.Record{}, err
	}
	sha, err := s.recipeSHA(img)
	if err != nil {
		return images.Record{}, err
	}
	recipe, err := catalog.Recipes.ReadFile(img.Recipe)
	if err != nil {
		return images.Record{}, err
	}
	cleanup, err := catalog.Recipes.ReadFile(img.Cleanup)
	if err != nil {
		return images.Record{}, err
	}
	key, pub, err := s.imageKey(ctx)
	if err != nil {
		return images.Record{}, &images.BuildFailedError{Image: img.Name, Step: "making the image key", Cause: err}
	}
	if err = s.dropLeftover(ctx, vm); err != nil {
		return images.Record{}, &images.BuildFailedError{Image: img.Name, Step: "removing the earlier build VM", Cause: err}
	}

	progress(images.Step{Label: "pulling " + t.From})
	if err = s.e.Fetch(ctx, s.location(t.From), nil); err != nil {
		return failed("pulling the base", err)
	}
	progress(images.Step{Label: "cloning " + vm})
	if err = s.e.Clone(ctx, s.location(t.From), vm); err != nil {
		return failed("cloning the base", err)
	}
	progress(images.Step{Label: "booting " + vm})
	if err = s.builds.Boot(ctx, vm); err != nil {
		return failed("booting the build VM", err)
	}
	b, err := s.builds.Open(ctx, vm, addressWait)
	if err != nil {
		return failed("waiting for the build VM's address", err)
	}
	defer func() { _ = b.Close() }()

	progress(images.Step{Label: "copying station into " + vm})
	if err = s.copy(ctx, b, binary); err != nil {
		return failed("copying the station binary", err)
	}
	progress(images.Step{Label: "running the " + img.Name + " recipe"})
	env := "VM_USER=" + quote(s.o.User) + " IMAGE_KEY=" + quote(strings.TrimSpace(pub))
	if err = recipeRun(ctx, b, string(recipe), env); err != nil {
		return failed("the recipe", err)
	}
	progress(images.Step{Label: "baking " + t.Name + "'s image phase"})
	lock, err := s.bakeRun(ctx, b, img, t, Login{User: s.o.User, Key: key})
	if err != nil {
		return failed("the image phase", err)
	}
	progress(images.Step{Label: "removing the base's admin user"})
	if err := cleanupRun(ctx, b, string(cleanup), Login{User: s.o.User, Key: key}); err != nil {
		return failed("the cleanup", err)
	}
	progress(images.Step{Label: "waiting for " + vm + " to power off"})
	if err := s.waitStopped(ctx, vm); err != nil {
		return failed("waiting for the build VM to power off", err)
	}
	_ = s.e.Shutdown(ctx, vm, 10*time.Second)
	progress(images.Step{Label: "recording " + img.Name + " " + version})
	final := DiskName(img.Name, version)
	if err := s.e.Rename(ctx, vm, final); err != nil {
		return failed("renaming the build VM", err)
	}
	rec := images.Record{
		Name: img.Name, Version: version, Base: t.From, Station: s.o.Version, RecipeSHA: sha, BuiltAt: s.m.Now().UTC(),
		Machinefile: t.Name, MachinefileSHA: t.SHA, Lock: lock,
	}
	if disks, err := s.e.Disks(ctx); err == nil {
		for _, d := range disks {
			if d.VM == final {
				rec.SizeGB = int(d.Bytes / 1e9)
			}
		}
	}
	if err := s.writeRecord(rec); err != nil {
		return images.Record{}, &images.BuildFailedError{Image: img.Name, Step: "writing the image record", Cause: err}
	}
	return rec, nil
}

func (s *Builder) dropLeftover(ctx context.Context, vm string) error {
	disks, err := s.e.Disks(ctx)
	if err != nil {
		return err
	}
	if slices.ContainsFunc(disks, func(d contract.Disk) bool { return d.VM == vm }) {
		_ = s.e.Shutdown(ctx, vm, 10*time.Second)
		return s.e.Delete(ctx, vm)
	}
	return nil
}

func (s *Builder) waitStopped(ctx context.Context, vm string) error {
	for range int(stopWait / pollEvery) {
		disks, err := s.e.Disks(ctx)
		if err != nil {
			return err
		}
		for _, d := range disks {
			if d.VM == vm && d.State == contract.DiskStopped {
				return nil
			}
		}
		if err := s.m.Sleep(ctx, pollEvery); err != nil {
			return err
		}
	}
	return fmt.Errorf("still running after %d s", int(stopWait/time.Second))
}

func (s *Builder) KeyPath() string { return s.o.Key }

func (s *Builder) imageKey(ctx context.Context) (key, pub string, err error) {
	key = s.KeyPath()
	if !s.m.Exists(key) {
		if made := s.m.MakeKey(ctx, key); made != nil {
			return "", "", made
		}
	}
	data, err := s.m.ReadFile(key + ".pub")
	if err != nil {
		return "", "", err
	}
	return key, string(data), nil
}

func (s *Builder) copy(ctx context.Context, b BuildVM, binary string) error {
	data, err := s.m.ReadFile(binary)
	if err != nil {
		return err
	}
	return b.Put(ctx, baseAdmin, remoteBin, data, 0o755)
}

func recipeRun(ctx context.Context, b BuildVM, script, env string) error {
	ctx, cancel := context.WithTimeout(ctx, recipeWait)
	defer cancel()
	out, err := b.Run(ctx, baseAdmin, catalog.AsRoot("env "+env+" bash -s"), script)
	if err != nil {
		return err
	}
	if out.Code != 0 {
		return outputError(out)
	}
	return nil
}

// value: the line that parts the image phase's own output from the lock
// it wrote, which the bake prints after it.
const lockMark = "--- station lock"

func (s *Builder) bakeRun(ctx context.Context, b BuildVM, img catalog.Image, t images.Template, as Login) (string, error) {
	// invariant: the image phase runs as the VM's user, its sudo
	// passwordless, over the chain's files and the lock put into its
	// owner's folder; what it hands back is the lock that run wrote.
	script, err := catalog.Recipes.ReadFile(img.Bake)
	if err != nil {
		return "", err
	}
	folder := "/home/" + s.o.User + "/.config/station/"
	files := slices.Clone(t.Files)
	if t.Lock != nil {
		files = append(files, images.TemplateFile{Path: "machinefiles/" + t.Name + catalog.LockSuffix, Body: t.Lock})
	}
	for _, f := range files {
		if put := b.Put(ctx, as, folder+f.Path, f.Body, 0o600); put != nil {
			return "", put
		}
	}
	ctx, cancel := context.WithTimeout(ctx, bakeWait)
	defer cancel()
	out, err := b.Run(ctx, as, "env MACHINEFILE="+quote(t.Name)+" bash -s", string(script))
	if err != nil {
		return "", err
	}
	if out.Code != 0 {
		return "", outputError(out)
	}
	_, lock, _ := strings.Cut(out.Stdout, lockMark+"\n")
	return lock, nil
}

func cleanupRun(ctx context.Context, b BuildVM, script string, as Login) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	out, err := b.Run(ctx, as, catalog.AsRoot("bash -s"), script)
	// invariant: the cleanup powers the VM off under its own session, so the
	// session ends with 0 or is cut by the poweroff; any other code is a
	// failed cleanup.
	if err != nil && ctx.Err() != nil {
		return err
	}
	if err == nil && out.Code != 0 {
		return outputError(out)
	}
	return nil
}

func outputError(out Ran) error {
	tail := strings.TrimSpace(out.Stderr)
	if tail == "" {
		tail = strings.TrimSpace(out.Stdout)
	}
	lines := strings.Split(tail, "\n")
	tail = strings.Join(lines[max(0, len(lines)-5):], "\n")
	return fmt.Errorf("exited %d: %s", out.Code, tail)
}

func quote(v string) string { return "'" + strings.ReplaceAll(v, "'", `'\''`) + "'" }
