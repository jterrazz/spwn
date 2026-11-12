package calls

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	model "github.com/jterrazz/studio-station/src/machines"

	imagescontract "github.com/jterrazz/studio-station/src/images/contract"
	"github.com/jterrazz/studio-station/src/machines/contract"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/machine"
)

var (
	errServerNeedsName = &model.KindRuleError{Why: "a server needs a name", Fix: "station machines create <name> --server"}
	errCheckForServers = &model.KindRuleError{Why: "a health check is for servers", Fix: "station machines create <name> --server --check <url>"}
	errServerBridged   = &model.KindRuleError{Why: "a server stays behind its Mac: --network bridged is for macOS workspaces", Fix: "share its ports instead: station ports share"}
)

func validCheck(check string) bool {
	// invariant: a check is an http(s) URL with a host, or host:port with a
	// port number (SPEC §7.2).
	if rest, ok := strings.CutPrefix(check, "https://"); ok {
		return rest != "" && !strings.HasPrefix(rest, "/")
	}
	if rest, ok := strings.CutPrefix(check, "http://"); ok {
		return rest != "" && !strings.HasPrefix(rest, "/")
	}
	mac, port, ok := strings.Cut(check, ":")
	n, err := strconv.Atoi(port)
	return ok && mac != "" && err == nil && n > 0 && n < 65536
}

// value: the image a name stands for, by its name or one it was renamed
// from; else the image a Machinefile of that name is built into, set up
// by the first boot of the built-in image of its system.
func (s *Machinist) imageOf(ctx context.Context, name string) (catalog.Image, bool) {
	for _, img := range s.o.Images {
		if img.Name == name || slices.Contains(img.RenamedFrom, name) {
			return img, true
		}
	}
	if s.o.Templates == nil {
		return catalog.Image{}, false
	}
	t, err := s.o.Templates.Template(ctx, name)
	if err != nil || t.Server {
		return catalog.Image{}, false
	}
	img, ok := catalog.Catalog{Images: s.o.Images}.RecipeFor(t.OS)
	if !ok {
		return catalog.Image{}, false
	}
	img.Name, img.Machinefile, img.RenamedFrom = t.Name, t.Name, nil
	return img, true
}

// invariant: a create from a Machinefile makes the image named after it,
// and takes what its vm says of how the workspace runs wherever the
// request says nothing; a server's Machinefile is refused, station
// setting nothing up in a server.
func (s *Machinist) fromTemplate(ctx context.Context, r model.CreateRequest) (model.CreateRequest, *model.Template, error) {
	if r.From == "" {
		return r, nil, nil
	}
	if s.o.Templates == nil {
		return r, nil, &model.UnknownImageError{Image: r.From}
	}
	t, err := s.o.Templates.Template(ctx, r.From)
	switch {
	case err != nil:
		return r, nil, err
	case t.Server || r.Server:
		return r, nil, &model.ServerTemplateError{Name: t.Name}
	}
	r.Image = t.Name
	r.Lifetime = cmp.Or(r.Lifetime, t.Lifetime)
	r.Network = cmp.Or(r.Network, t.Network)
	r.Size.CPUs = cmp.Or(r.Size.CPUs, t.Size.CPUs)
	r.Size.MemoryBytes = cmp.Or(r.Size.MemoryBytes, t.Size.MemoryBytes)
	r.Size.DiskBytes = cmp.Or(r.Size.DiskBytes, t.Size.DiskBytes)
	return r, &t, nil
}

func networkFor(img catalog.Image, asked string) (string, error) {
	switch asked {
	case "", model.NetworkNAT:
		return model.NetworkNAT, nil
	case model.Bridged:
		if img.Kind != catalog.MacOS {
			return asked, &model.BadNetworkError{Value: asked, Bridge: true}
		}
		return model.Bridged, nil
	default:
		return asked, &model.BadNetworkError{Value: asked}
	}
}

func (s *Machinist) CheckCreate(ctx context.Context, r model.CreateRequest) (model.CreateRequest, error) {
	if err := s.CheckMac(); err != nil {
		return r, err
	}
	r, _, err := s.fromTemplate(ctx, r)
	if err != nil {
		return r, err
	}
	return s.checkCreate(ctx, r)
}

func (s *Machinist) checkCreate(ctx context.Context, r model.CreateRequest) (model.CreateRequest, error) {
	server := r.Server
	switch {
	case server:
		r.Lifetime = model.AlwaysOn
	case r.Lifetime == "":
		r.Lifetime = model.Ephemeral
	case r.Lifetime != model.Ephemeral && r.Lifetime != model.Persistent:
		return r, &model.BadLifetimeError{Value: string(r.Lifetime)}
	}
	switch {
	case server && r.Name == "":
		return r, errServerNeedsName
	case server && r.Network == model.Bridged:
		return r, errServerBridged
	case !server && r.Check != "":
		return r, errCheckForServers
	case r.Check != "" && !validCheck(r.Check):
		return r, &model.BadCheckError{Value: r.Check}
	}
	img, ok := s.imageOf(ctx, r.Image)
	if !ok {
		return r, &model.UnknownImageError{Image: r.Image}
	}
	r.Image = img.Name
	network, err := networkFor(img, r.Network)
	if err != nil {
		return r, err
	}
	r.Network = network
	if r.Name != "" && !validName(r.Name) {
		return r, &model.BadNameError{Name: r.Name}
	}
	if _, err = s.engine.Version(ctx); err != nil {
		return r, &model.NoEngineError{Hypervisor: s.engine.Name(), Cause: err}
	}
	if _, _, err = s.img.Latest(ctx, r.Image); err != nil {
		return r, err
	}
	held, err := s.claims(ctx)
	if err != nil {
		return r, err
	}
	switch {
	case r.Name == "":
		r.Name = allocateName(string(img.Kind), held)
	case !held.free(r.Name):
		return r, held.taken(r.Name)
	}
	size := model.Resources{CPUs: img.Defaults.CPUs, MemoryBytes: img.Defaults.MemoryBytes, DiskBytes: img.Defaults.DiskBytes}
	if server {
		size = model.ServerSize()
	}
	if r.Size.CPUs == 0 {
		r.Size.CPUs = size.CPUs
	}
	if r.Size.MemoryBytes == 0 {
		r.Size.MemoryBytes = size.MemoryBytes
	}
	if r.Size.DiskBytes == 0 {
		r.Size.DiskBytes = size.DiskBytes
	}
	if left, ok := s.memoryLeft(ctx); ok && r.Size.MemoryBytes > left {
		return r, &model.TooBigError{Mac: "This Mac", AskedGB: int(r.Size.MemoryBytes >> 30), LeftGB: int(max(left, 0) >> 30)}
	}
	if err := s.engineUp(ctx, false); err != nil {
		return r, err
	}
	return r, nil
}

func (s *Machinist) EngineUp(ctx context.Context) error { return s.engineUp(ctx, true) }

func (s *Machinist) engineUp(ctx context.Context, start bool) error {
	// invariant: the engine comes to a Mac with its hypervisor the first
	// time a call needs it: an engine that does not answer and whose job was
	// never installed is installed and started here, then asked again; an
	// engine the owner stopped stays stopped, refused with its start. A
	// check that does not start it (create's, before its questions) lets
	// such an engine through: the create itself starts it.
	if s.engineAPI == nil {
		return &model.EngineDownError{Mac: "This Mac", Installed: s.h.Exists(catalog.EnginePlist(s.h.Home()))}
	}
	_, err := s.engineAPI.Ping(ctx)
	if !errors.Is(err, model.ErrEngineDown) {
		return err
	}
	installed := s.h.Exists(catalog.EnginePlist(s.h.Home()))
	if installed || s.o.StartEngine == nil || !s.engine.Installed() {
		return &model.EngineDownError{Mac: "This Mac", Installed: installed}
	}
	if !start {
		return nil
	}
	if err := s.o.StartEngine(ctx); err != nil {
		return err
	}
	if _, err := s.engineAPI.Ping(ctx); err != nil {
		if errors.Is(err, model.ErrEngineDown) {
			return &model.EngineDownError{Mac: "This Mac", Installed: true}
		}
		return err
	}
	return nil
}

// value: every name All machines holds, for a VM's name the Mac it is on,
// worded as the owner reads it: This Mac, or that Mac's short name; and
// every agent's of a running workspace of a Mac that answers, and those a
// workspace held when it last stopped, of one stopped or of a silent Mac.
func (s *Machinist) claims(ctx context.Context) (claims, error) {
	allMachines, err := s.AllMachines(ctx, false)
	if err != nil {
		return claims{}, err
	}
	c := claims{held: map[string]claim{}, on: map[string]string{}}
	for _, h := range allMachines.Macs {
		c.held[h.Name] = claimMac
		mac := "This Mac"
		if !h.ThisMac {
			mac = strings.TrimPrefix(h.Name, s.o.User+"-")
		}
		for _, vm := range h.VMs {
			c.held[vm.Name], c.on[vm.Name] = claimVM, mac
			if vm.Lifetime.Kind() != machine.Workspace {
				continue
			}
			names := vm.Agents
			if (h.ThisMac || h.Reachable) && vm.State == model.Running {
				names = nil
				if s.o.AgentsOf != nil {
					names = s.o.AgentsOf(ctx, vm.Name)
				}
			}
			for _, name := range names {
				c.agents = append(c.agents, heldAgent{Name: name, VM: vm.Name})
			}
		}
	}
	cfg, err := s.prefs()
	if err != nil {
		return claims{}, err
	}
	for name := range cfg.Macs {
		c.held[name] = claimMac
	}
	return c, nil
}

// value: the memory a new VM may take on this Mac, its capacity's free
// figure; unknown when the Mac does not say what it has.
func (s *Machinist) memoryLeft(ctx context.Context) (int64, bool) {
	_, live := s.engineView(ctx)
	vms, _ := s.vms(ctx, "", live)
	c, ok := model.CapacityOf(model.MacView{Hardware: s.hardware(ctx), VMs: vms})
	return c.Free, ok
}

func (s *Machinist) Create(ctx context.Context, r model.CreateRequest, progress func(model.Step)) (model.Created, error) {
	// invariant: a failure after the clone and before the first run rolls
	// every made thing back, and from the first run on the VM is kept, with
	// what failed; the ports its env names are reserved by its Mac's engine
	// at that run, and read back once it answers.
	if err := s.CheckMac(); err != nil {
		return model.Created{}, err
	}
	r, t, err := s.fromTemplate(ctx, r)
	if err != nil {
		return model.Created{}, err
	}
	if r, err = s.checkCreate(ctx, r); err != nil {
		return model.Created{}, err
	}
	if up := s.EngineUp(ctx); up != nil {
		return model.Created{}, up
	}
	say := func(label string) {
		if progress != nil {
			progress(model.Step{Label: label})
		}
	}
	cfg, err := s.prefs()
	if err != nil {
		return model.Created{}, err
	}
	img, _ := s.imageOf(ctx, r.Image)
	disk, version, err := s.img.Latest(ctx, r.Image)
	if err != nil {
		return model.Created{}, err
	}
	server := r.Lifetime == model.AlwaysOn
	rec := model.VMRecord{
		Name: r.Name, Image: r.Image, Version: version, Lifetime: r.Lifetime, Network: r.Network, Check: r.Check,
		Resources: r.Size, CreatedAt: s.h.Now().UTC().Truncate(time.Second), Template: r.From,
	}
	want := ""
	if r.Lifetime == model.Persistent || server {
		want = model.WantRunning
	}
	made := &made{}
	// invariant: a create the owner interrupts ends as an interruption,
	// after the same rollback or keep as any failure at that step.
	failed := func(err error) (model.Created, error) {
		if ctx.Err() != nil {
			return model.Created{}, ctx.Err()
		}
		return model.Created{}, err
	}
	fail := func(step string, cause error) (model.Created, error) {
		s.undoCreate(ctx, r.Name, made)
		return failed(&model.CreateFailedError{Name: r.Name, Step: step, Cause: cause})
	}
	// invariant: SPEC B5.10: the record exists from the create's first
	// frame, its memory reserved, wanting nothing until its disk is made;
	// each step moves its birth on, and the create's end, made or kept
	// failing, takes it off.
	born := func(step string) error {
		rec.Birth = model.BirthAt(step, s.h.Now())
		return s.writeBirth(rec)
	}
	defer func() {
		if made.record && !made.undone {
			rec.Birth = nil
			_ = s.writeBirth(rec)
		}
	}()

	say("cloning " + disk)
	if err = born(model.BornCloning); err != nil {
		return fail("the record", err)
	}
	made.record = true
	if err = s.engine.Clone(ctx, imagescontract.ImageLocation{Hypervisor: s.engine.Name(), Ref: disk}, r.Name); err != nil {
		return fail("the clone", err)
	}
	made.vm = true
	if err = s.engine.Resize(ctx, r.Name, r.Size.Size()); err != nil {
		return fail("the resize", err)
	}
	rec.Want = want
	if err = s.writeBirth(rec); err != nil {
		return fail("the record", err)
	}
	say(model.StepRunning)
	_ = born(model.BornBooting)
	if err = s.engineAPI.Do(ctx, "run", r.Name); err != nil {
		return failed(s.unhealthy(ctx, r.Name, err))
	}
	say(model.StepAddress)
	_ = born(model.BornAddress)
	addr, err := s.engine.Address(ctx, r.Name, ipWait)
	if err != nil {
		return failed(&model.UnhealthyError{Name: r.Name, Step: model.StepAddress, Detail: "none after 60 s"})
	}
	ip := addr.String()
	say(model.StepSSH)
	_ = born(model.BornSSH)
	if err := s.setUp(ctx, img, r); err != nil {
		return failed(&model.UnhealthyError{Name: r.Name, Step: model.StepSSH, Detail: strings.TrimSpace(err.Error())})
	}
	var boot *model.FirstStart
	if t != nil {
		say(model.StepBoot)
		boot = s.bootPhase(ctx, r.Name, version, *t)
	}
	if allMachines, err := s.AllMachines(ctx, false); err == nil {
		_ = s.SSHBlocks(ctx, allMachines)
	}
	_ = s.engineAPI.Do(ctx, "ready", r.Name)
	var reserved []model.Reserved
	var env map[string]string
	if !server {
		reserved, _ = s.engineAPI.EnvPorts(ctx, r.Name)
		if kept, err := s.recs.Get(ctx, r.Name); err == nil {
			env = kept.Env
		}
	}
	// invariant: a workspace made is a machine this Mac's multiplexer
	// reaches, by its name, over the ssh block just written; a multiplexer that
	// cannot add it leaves the herdr-machine setting to say so.
	if r.Lifetime != model.AlwaysOn && s.o.Terminals != nil {
		_ = s.o.Terminals.Add(ctx, r.Name)
	}
	return model.Created{
		VM: model.VM{
			Name: r.Name, Image: r.Image, Version: version, Mac: s.thisMac(cfg), Kind: img.Kind, Lifetime: r.Lifetime,
			State: model.Running, Resources: r.Size, Network: r.Network, IP: ip, Check: r.Check, CreatedAt: rec.CreatedAt,
		},
		Ports: reserved,
		Env:   env,
		Boot:  boot,
	}, nil
}

// invariant: a workspace is set up by its image's first boot; a server is
// set up by its owner's own tooling, so station only waits for its ssh.
func (s *Machinist) setUp(ctx context.Context, img catalog.Image, r model.CreateRequest) error {
	if r.Lifetime != model.AlwaysOn {
		return s.firstBoot(ctx, img, r.Name, r.Lifetime)
	}
	ctx, cancel := context.WithTimeout(ctx, sshWait)
	defer cancel()
	out, err := s.engineAPI.GuestExec(ctx, r.Name, "true", "")
	switch {
	case err != nil:
		return err
	case out.ExitCode != 0:
		return errors.New("none after 30 s")
	}
	return nil
}

// invariant: once its first run is asked, a VM is kept whatever fails
// (CR-15, CR-16): a run the hypervisor never began did not boot; one that
// runs failed the first health step that did not pass (SPEC §7.2).
func (s *Machinist) unhealthy(ctx context.Context, name string, cause error) error {
	if s.engineState(ctx, name) != contract.DiskRunning {
		return &model.DidNotBootError{Name: name, Cause: cause}
	}
	if _, err := s.engine.Address(ctx, name, 0); err != nil {
		return &model.UnhealthyError{Name: name, Step: model.StepAddress, Detail: "none after 60 s"}
	}
	return &model.UnhealthyError{Name: name, Step: model.StepSSH, Detail: "none after 30 s"}
}

type made struct{ vm, record, undone bool }

// invariant: the rollback is best effort and runs every step whatever fails, so a
// half-made VM leaves as little as it can; what it reports is the create's failure.
func (s *Machinist) undoCreate(ctx context.Context, name string, m *made) {
	m.undone = true
	if m.vm {
		if s.engineState(ctx, name) == contract.DiskRunning {
			_ = s.engine.Shutdown(ctx, name, hardStopWait)
		}
		_ = s.engine.Delete(ctx, name)
	}
	s.removeFiles(name, m.record)
}

func (s *Machinist) removeFiles(name string, record bool) {
	for _, f := range []string{s.LogPath(name), s.KnownHostsPath(name)} {
		_ = s.h.Remove(f)
	}
	if record {
		_ = s.recs.Delete(context.Background(), name)
	}
}

func (s *Machinist) Save(r model.VMRecord) error {
	// invariant: a birth is the create's alone to write: a record saved by
	// anyone else keeps the one stored, read in the same Update as its
	// write, so none outlives its create.
	return s.recs.Update(context.Background(), r.Name, func(stored *model.Record) {
		birth := stored.Birth
		*stored = storedOf(r)
		stored.Birth = birth
	})
}

// invariant: the env is the engine's to write, at the VM's run: each write
// of the create keeps the one stored, read in the same Update as its
// write, so the create never writes back an env the engine replaced.
func (s *Machinist) writeBirth(r model.VMRecord) error {
	return s.recs.Update(context.Background(), r.Name, func(stored *model.Record) {
		env := stored.Env
		*stored = storedOf(r)
		stored.Env = env
	})
}

func (s *Machinist) writeRecord(r model.VMRecord) error {
	return s.recs.Put(context.Background(), storedOf(r))
}
