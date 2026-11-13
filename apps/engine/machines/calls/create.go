package calls

import (
	"cmp"
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	model "github.com/jterrazz/studio-station/apps/engine/machines"

	imagescontract "github.com/jterrazz/studio-station/apps/engine/images/contract"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/engine"
)

var (
	ServerNeedsNameRule = &model.KindRuleError{Why: "a server needs a name", Fix: "station machines create <name> --server"}
	CheckForServersRule = &model.KindRuleError{Why: "a health check is for servers", Fix: "station machines create <name> --server --check <url>"}
	ServerBridgedRule   = &model.KindRuleError{Why: "a server stays behind its Mac: --network bridged is for macOS workspaces", Fix: "share its ports instead: station ports share"}
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

// value: the image of that name; else the image a Machinefile of that name is built into, set up
// by the first boot of the built-in image of its system.
func (s *Machinist) imageOf(ctx context.Context, name string) (catalog.Image, bool) {
	for _, img := range s.o.Images {
		if img.Name == name {
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
	img.Name, img.Machinefile = t.Name, t.Name
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
	if r, _, err = s.checkCreate(ctx, r); err != nil {
		return r, err
	}
	if err := s.built(ctx, r.Image); err != nil {
		return r, err
	}
	return r, s.engineUp(ctx, false)
}

// invariant: every refusal of a create that needs no built image, the
// memory's included, so a create the Mac has no room for is refused before
// its image is built; with the names the fleet holds, which the create's
// claim judges again with This Mac's records as they are then.
func (s *Machinist) checkCreate(ctx context.Context, r model.CreateRequest) (model.CreateRequest, claims, error) {
	server := r.Server
	switch {
	case server:
		r.Lifetime = model.AlwaysOn
	case r.Lifetime == "":
		r.Lifetime = model.Ephemeral
	case r.Lifetime != model.Ephemeral && r.Lifetime != model.Persistent:
		return r, claims{}, &model.BadLifetimeError{Value: string(r.Lifetime)}
	}
	switch {
	case server && r.Name == "":
		return r, claims{}, ServerNeedsNameRule
	case server && r.Network == model.Bridged:
		return r, claims{}, ServerBridgedRule
	case !server && r.Check != "":
		return r, claims{}, CheckForServersRule
	case r.Check != "" && !validCheck(r.Check):
		return r, claims{}, &model.BadCheckError{Value: r.Check}
	case r.Creator != "" && (!validName(r.Creator) || r.Creator == model.ByOwner):
		return r, claims{}, &model.BadCreatorError{Session: r.Creator}
	case !validPurpose(r.Purpose):
		return r, claims{}, &model.BadPurposeError{Purpose: r.Purpose}
	}
	img, ok := s.imageOf(ctx, r.Image)
	if !ok {
		return r, claims{}, &model.UnknownImageError{Image: r.Image}
	}
	r.Image = img.Name
	network, err := networkFor(img, r.Network)
	if err != nil {
		return r, claims{}, err
	}
	r.Network = network
	if r.Name != "" && !validName(r.Name) {
		return r, claims{}, &model.BadNameError{Name: r.Name}
	}
	if _, err = s.engine.Version(ctx); err != nil {
		return r, claims{}, &model.NoEngineError{Hypervisor: s.engine.Name(), Cause: err}
	}
	held, err := s.claims(ctx)
	if err != nil {
		return r, claims{}, err
	}
	switch {
	case r.Name == "":
		r.Name = allocateName(string(img.Kind), held)
	case !held.free(r.Name):
		return r, claims{}, held.taken(r.Name)
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
	if c, ok := s.capacity(ctx); ok && !model.Fits(r.Size, c) {
		return r, claims{}, &model.TooBigError{Mac: s.ThisMacName(), AskedGB: int(r.Size.MemoryBytes >> 30), LeftGB: int(c.Free >> 30)}
	}
	return r, held, nil
}

// value: refuses an image not built on this Mac, in its name.
func (s *Machinist) built(ctx context.Context, image string) error {
	_, _, err := s.img.Latest(ctx, image)
	if unbuilt := new(model.NoImageError); errors.As(err, &unbuilt) {
		return &model.NoImageError{Image: unbuilt.Image, Mac: s.ThisMacName(), Fetched: unbuilt.Fetched}
	}
	return err
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
		return &model.EngineDownError{Mac: s.ThisMacName(), Installed: s.h.Exists(catalog.EnginePlist(s.h.Home()))}
	}
	_, err := s.engineAPI.Ping(ctx)
	if !errors.Is(err, model.ErrEngineDown) {
		return err
	}
	installed := s.h.Exists(catalog.EnginePlist(s.h.Home()))
	if installed || s.o.StartEngine == nil || !s.engine.Installed() {
		return &model.EngineDownError{Mac: s.ThisMacName(), Installed: installed}
	}
	if !start {
		return nil
	}
	if err := s.o.StartEngine(ctx); err != nil {
		return err
	}
	if _, err := s.engineAPI.Ping(ctx); err != nil {
		if errors.Is(err, model.ErrEngineDown) {
			return &model.EngineDownError{Mac: s.ThisMacName(), Installed: true}
		}
		return err
	}
	return nil
}

// value: every name All machines holds, for a VM's name the Mac it is on,
// by the Name machines.list gives it, This Mac's included, and every VM's
// another Mac was last seen holding; and
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
		for _, vm := range h.VMs {
			c.held[vm.Name], c.on[vm.Name] = claimVM, h.Name
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
	for name, h := range cfg.Macs {
		c.held[name] = claimMac
		// invariant: a VM a silent Mac was last seen holding keeps its name
		// held, its kind told or not: the list leaves out one with no kind,
		// a create never takes its name.
		for _, vm := range h.VMs {
			if c.held[vm.Name] == unclaimed {
				c.held[vm.Name], c.on[vm.Name] = claimVM, name
			}
		}
	}
	return c, nil
}

// value: this Mac's capacity; unknown when the Mac does not say what it has.
func (s *Machinist) capacity(ctx context.Context) (model.Capacity, bool) {
	_, live := s.engineView(ctx)
	vms, _ := s.vms(ctx, "", live)
	return model.CapacityOf(model.MacView{Hardware: s.hardware(ctx), VMs: vms})
}

// invariant: a create's record lands by a claim of This Mac's records, so
// two creates of any processes of this Mac never both take the memory or
// the name one of them was judged for: the claim judges the name and the
// memory again from the records as they are then, the names the fleet
// held and what the Mac and its engine said read before it, so no other
// Mac nor the engine is waited for while the records are held. A name the
// create chose is chosen again there.
func (s *Machinist) claim(ctx context.Context, asked string, r *model.CreateRequest, img catalog.Image, others claims, rec *model.VMRecord) error {
	hardware, disks := s.hardware(ctx), s.disks(ctx)
	_, live := s.engineView(ctx)
	var refused error
	err := s.recs.Claim(ctx, func(records []model.Record) (model.Record, error) {
		held := others.with(s.ThisMacName(), records)
		switch {
		case asked == "":
			r.Name = allocateName(string(img.Kind), held)
		case !held.free(asked):
			refused = held.taken(asked)
			return model.Record{}, refused
		}
		vms := s.vmsOf("", records, disks, live)
		if c, ok := model.CapacityOf(model.MacView{Hardware: hardware, VMs: vms}); ok && !model.Fits(r.Size, c) {
			refused = &model.TooBigError{Mac: s.ThisMacName(), AskedGB: int(r.Size.MemoryBytes >> 30), LeftGB: int(c.Free >> 30)}
			return model.Record{}, refused
		}
		rec.Name = r.Name
		return storedOf(*rec), nil
	})
	switch {
	case refused != nil:
		return refused
	case err != nil && ctx.Err() == nil:
		return &model.CreateFailedError{Name: r.Name, Step: "the record", Cause: err}
	}
	return err
}

// value: the process this create runs in; nil when it cannot say, which
// no reader reads as running.
func (s *Machinist) self() *model.Process {
	if s.o.Processes == nil {
		return nil
	}
	p, err := s.o.Processes.Self()
	if err != nil {
		return nil
	}
	return &p
}

func (s *Machinist) runs(p *model.Process) bool {
	return p != nil && s.o.Processes != nil && s.o.Processes.Runs(*p)
}

func (s *Machinist) Create(ctx context.Context, r model.CreateRequest, build ImageBuild, progress func(model.Step)) (model.Created, error) {
	// invariant: a failure before the first run, the image's build
	// included, rolls every made thing back, its record too, and from the
	// first run on the VM is kept, with what failed; the ports its env
	// names are reserved by its Mac's engine at that run, and read back
	// once it answers.
	if err := s.CheckMac(); err != nil {
		return model.Created{}, err
	}
	r, t, err := s.fromTemplate(ctx, r)
	if err != nil {
		return model.Created{}, err
	}
	asked := r.Name
	r, others, err := s.checkCreate(ctx, r)
	if err != nil {
		return model.Created{}, err
	}
	switch unbuilt := s.built(ctx, r.Image); {
	case unbuilt == nil:
		build = nil
	case build == nil || !errors.As(unbuilt, new(*model.NoImageError)):
		return model.Created{}, unbuilt
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
	server := r.Lifetime == model.AlwaysOn
	owner := s.self()
	rec := model.VMRecord{
		Name: r.Name, Image: r.Image, Lifetime: r.Lifetime, Network: r.Network, Check: r.Check,
		Resources: r.Size, CreatedAt: s.h.Now().UTC().Truncate(time.Second), Template: r.From,
		Purpose: r.Purpose, Creator: r.Creator, Owner: owner,
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
	// frame, its image's build included, its memory reserved, wanting
	// nothing until its disk is made; each step moves its birth on, and a
	// record gone meanwhile (deleted) stops the create, never made again;
	// a failure before the clone takes it off, and the create's end, made
	// or kept failing, takes its birth off.
	born := func(step model.BirthStep, detail string) error {
		rec.Birth = model.BirthAt(step, build != nil, s.h.Now())
		rec.Birth.Detail = detail
		return s.writeBirth(r.Name, func(stored *model.Record) { stored.Birth = rec.Birth })
	}
	defer func() {
		if made.record && !made.undone {
			_ = s.writeBirth(r.Name, func(stored *model.Record) { stored.Birth, stored.Owner = nil, nil })
		}
	}()
	first := model.BornCloning
	if build != nil {
		first = model.BornBuilding
	}
	rec.Birth = model.BirthAt(first, build != nil, s.h.Now())
	if err = s.claim(ctx, asked, &r, img, others, &rec); err != nil {
		return failed(err)
	}
	made.record = true

	if build != nil {
		building, stop := context.WithCancel(ctx)
		var gone error
		err = build(building, func(label string) {
			say(label)
			if moved := born(model.BornBuilding, label); moved != nil && gone == nil {
				gone = moved
				stop()
			}
		})
		stop()
		if gone != nil {
			return fail("the record", gone)
		}
		if err != nil {
			s.undoCreate(ctx, r.Name, made)
			return failed(err)
		}
	}
	disk, version, err := s.img.Latest(ctx, r.Image)
	if err != nil {
		s.undoCreate(ctx, r.Name, made)
		return failed(err)
	}
	say("cloning " + disk)
	if err = born(model.BornCloning, ""); err != nil {
		return fail("the record", err)
	}
	if err = s.engine.Clone(ctx, imagescontract.ImageLocation{Hypervisor: s.engine.Name(), Ref: disk}, r.Name); err != nil {
		return fail("the clone", err)
	}
	made.vm = true
	if err = s.engine.Resize(ctx, r.Name, r.Size.Size()); err != nil {
		return fail("the resize", err)
	}
	if err = s.writeBirth(r.Name, func(stored *model.Record) { stored.Version, stored.Want = version, want }); err != nil {
		return fail("the record", err)
	}
	say(model.StepRunning)
	if err = born(model.BornBooting, ""); err != nil {
		return fail("the record", err)
	}
	if err = s.engineAPI.Do(ctx, engine.VMRun, r.Name); err != nil {
		return failed(s.unhealthy(ctx, r.Name, err))
	}
	say(model.StepAddress)
	if err = born(model.BornAddress, ""); err != nil {
		return failed(&model.UnknownMachineError{Name: r.Name})
	}
	addr, err := s.engine.Address(ctx, r.Name, ipWait)
	if err != nil {
		return failed(&model.UnhealthyError{Name: r.Name, Step: model.StepAddress, Detail: "none after 60 s"})
	}
	ip := addr.String()
	say(model.StepSSH)
	if err = born(model.BornSSH, ""); err != nil {
		return failed(&model.UnknownMachineError{Name: r.Name})
	}
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
	_ = s.engineAPI.Do(ctx, engine.VMReady, r.Name)
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
			State: model.Running, Resources: r.Size, Network: r.Network, IP: ip, Check: model.ShownCheck(r.Check), CreatedAt: rec.CreatedAt,
			Purpose: r.Purpose, Creator: model.CreatorOf(r.Creator),
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
	// anyone else keeps the one stored, read in the same Amend as its
	// write, so none outlives its create; a record gone (deleted) is never
	// made again, and its save reads the VM unknown.
	err := s.recs.Amend(context.Background(), r.Name, func(stored *model.Record) {
		birth, owner := stored.Birth, stored.Owner
		*stored = storedOf(r)
		stored.Birth, stored.Owner = birth, owner
	})
	if errors.Is(err, store.ErrNotFound) {
		return &model.UnknownMachineError{Name: r.Name}
	}
	return err
}

// invariant: a create writes back only what it owns of its record: its
// birth and its process, the version it cloned and what it wants, each
// read in the same Amend as its write, so an edit, a stop or the engine's
// env and mounts written meanwhile stay; a record gone is never made again.
func (s *Machinist) writeBirth(name string, change func(r *model.Record)) error {
	return s.recs.Amend(context.Background(), name, change)
}

func (s *Machinist) writeRecord(r model.VMRecord) error {
	return s.recs.Put(context.Background(), storedOf(r))
}
