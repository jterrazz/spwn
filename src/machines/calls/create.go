package calls

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	model "github.com/jterrazz/studio-station/src/machines"

	imagescontract "github.com/jterrazz/studio-station/src/images/contract"
	"github.com/jterrazz/studio-station/src/machines/contract"

	"github.com/jterrazz/studio-station/src/catalog"
)

// value: what macOS keeps of a Mac's memory, never given to a VM (SPEC §7.1).
const keptForMacOS = 4 << 30

var (
	errServerNeedsName   = &model.KindRuleError{Why: "a server needs a name", Fix: "station machines create <name> --server"}
	errCheckForServers   = &model.KindRuleError{Why: "a health check is for servers", Fix: "station machines create <name> --server --check <url>"}
	errTaskForWorkspaces = &model.KindRuleError{Why: "a task is for a workspace's agent: a server runs none", Fix: "station machines create --task <task> --agent"}
	errServerBridged     = &model.KindRuleError{Why: "a server stays behind its Mac: --network bridged is for macOS workspaces", Fix: "share its ports instead: station ports share"}
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

func imageOf(name string) (catalog.Image, bool) {
	for _, img := range catalog.Images {
		if img.Name == name {
			return img, true
		}
	}
	return catalog.Image{}, false
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
	server := r.Server
	switch {
	case server:
		r.Lifetime = model.AlwaysOn
	case r.Lifetime == "":
		r.Lifetime = model.Throwaway
	case r.Lifetime != model.Throwaway && r.Lifetime != model.Permanent:
		return r, &model.BadLifetimeError{Value: string(r.Lifetime)}
	}
	switch {
	case server && r.Name == "":
		return r, errServerNeedsName
	case server && (r.Task != "" || r.Start):
		return r, errTaskForWorkspaces
	case server && r.Network == model.Bridged:
		return r, errServerBridged
	case !server && r.Check != "":
		return r, errCheckForServers
	case r.Check != "" && !validCheck(r.Check):
		return r, &model.BadCheckError{Value: r.Check}
	}
	img, ok := imageOf(r.Image)
	if !ok {
		return r, &model.UnknownImageError{Image: r.Image}
	}
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
	held, on, err := s.claims(ctx)
	if err != nil {
		return r, err
	}
	switch {
	case r.Name == "":
		r.Name = allocateName(r.Image, held)
	case !nameFree(r.Name, held):
		return r, &model.NameTakenError{Name: r.Name, VM: held[r.Name] == claimVM, Mac: held[r.Name] == claimMac, On: on[r.Name]}
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
	if err := s.EngineUp(ctx); err != nil {
		return r, err
	}
	return r, nil
}

func (s *Machinist) EngineUp(ctx context.Context) error {
	installed := s.h.Exists(catalog.EnginePlist(s.h.Home()))
	down := &model.EngineDownError{Mac: "This Mac", Installed: installed}
	if s.engineAPI == nil {
		return down
	}
	if _, err := s.engineAPI.Ping(ctx); err != nil {
		if errors.Is(err, model.ErrEngineDown) {
			return down
		}
		return err
	}
	return nil
}

// value: every name All machines holds, and for a VM's name the Mac it is
// on, worded as the owner reads it: This Mac, or that Mac's short name.
func (s *Machinist) claims(ctx context.Context) (held map[string]claim, on map[string]string, err error) {
	allMachines, err := s.AllMachines(ctx, false)
	if err != nil {
		return nil, nil, err
	}
	held, on = map[string]claim{}, map[string]string{}
	for _, h := range allMachines.Macs {
		held[h.Name] = claimMac
		mac := "This Mac"
		if !h.ThisMac {
			mac = strings.TrimPrefix(h.Name, s.o.User+"-")
		}
		for _, vm := range h.VMs {
			held[vm.Name], on[vm.Name] = claimVM, mac
		}
	}
	cfg, err := s.prefs()
	if err != nil {
		return nil, nil, err
	}
	for name := range cfg.Macs {
		held[name] = claimMac
	}
	return held, on, nil
}

// value: the memory a new VM may take on this Mac: what it has, less what
// macOS keeps and what its running VMs hold; unknown when the Mac does not
// say what it has.
func (s *Machinist) memoryLeft(ctx context.Context) (int64, bool) {
	total := int64(0)
	if s.o.Memory != nil {
		total = s.o.Memory(ctx)
	}
	if total <= 0 {
		return 0, false
	}
	left := total - keptForMacOS
	_, live := s.engineView(ctx)
	if vms, err := s.vms(ctx, "", live); err == nil {
		for _, vm := range vms {
			if vm.State == model.Running {
				left -= vm.Resources.MemoryBytes
			}
		}
	}
	return left, true
}

func (s *Machinist) Create(ctx context.Context, r model.CreateRequest, progress func(model.Step)) (model.Created, error) {
	// invariant: the profiles' ports are reserved before the VM first runs, and
	// their rendered variables go into the record, fixed for the VM's life; a
	// failure after the clone and before the first run rolls every made thing
	// back, and from the first run on the VM is kept, with what failed.
	r, err := s.CheckCreate(ctx, r)
	if err != nil {
		return model.Created{}, err
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
	img, _ := imageOf(r.Image)
	disk, version, err := s.img.Latest(ctx, r.Image)
	if err != nil {
		return model.Created{}, err
	}
	server := r.Lifetime == model.AlwaysOn
	rec := model.VMRecord{
		Name: r.Name, Image: r.Image, Version: version, Lifetime: r.Lifetime, Network: r.Network, Check: r.Check, Task: r.Task,
		Resources: r.Size, CreatedAt: s.h.Now().UTC().Truncate(time.Second),
	}
	if r.Lifetime == model.Permanent || server {
		rec.Want = model.WantRunning
	}
	made := &made{}
	fail := func(step string, cause error) (model.Created, error) {
		s.undoCreate(ctx, r.Name, made)
		return model.Created{}, &model.CreateFailedError{Name: r.Name, Step: step, Cause: cause}
	}

	say("cloning " + disk)
	if err = s.engine.Clone(ctx, imagescontract.ImageLocation{Hypervisor: s.engine.Name(), Ref: disk}, r.Name); err != nil {
		return fail("the clone", err)
	}
	made.vm = true
	if err = s.engine.Resize(ctx, r.Name, r.Size.Size()); err != nil {
		return fail("the resize", err)
	}
	if err = s.writeRecord(rec); err != nil {
		return fail("the record", err)
	}
	made.record = true
	var reserved []model.Reserved
	var env map[string]string
	if !server {
		say("reserving its ports")
		made.reservations = true
		if reserved, env, err = s.reserveProfiles(ctx, r.Name); err != nil {
			return fail("the reservations", err)
		}
		rec.Env = env
		if err = s.writeRecord(rec); err != nil {
			return fail("the record", err)
		}
	}
	say(model.StepRunning)
	if err = s.engineAPI.Do(ctx, "run", r.Name); err != nil {
		return model.Created{}, s.unhealthy(ctx, r.Name, err)
	}
	say(model.StepAddress)
	addr, err := s.engine.Address(ctx, r.Name, ipWait)
	if err != nil {
		return model.Created{}, &model.UnhealthyError{Name: r.Name, Step: model.StepAddress, Detail: "none after 60 s"}
	}
	ip := addr.String()
	say(model.StepSSH)
	if err := s.setUp(ctx, img, r); err != nil {
		return model.Created{}, &model.UnhealthyError{Name: r.Name, Step: model.StepSSH, Detail: strings.TrimSpace(err.Error())}
	}
	if allMachines, err := s.AllMachines(ctx, false); err == nil {
		_ = s.SSHBlocks(ctx, allMachines)
	}
	return model.Created{
		VM: model.VM{
			Name: r.Name, Image: r.Image, Version: version, Mac: s.thisMac(cfg), Kind: img.Kind, Lifetime: r.Lifetime,
			State: model.Running, Resources: r.Size, Network: r.Network, IP: ip, Check: r.Check, Task: r.Task, CreatedAt: rec.CreatedAt,
		},
		Ports: reserved,
		Env:   env,
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

func (s *Machinist) reserveProfiles(ctx context.Context, name string) (reserved []model.Reserved, env map[string]string, err error) {
	for _, p := range catalog.Profiles {
		got, err := s.engineAPI.Reserve(ctx, name, p.Port, p.Name)
		if err != nil {
			return nil, nil, err
		}
		reserved = append(reserved, got)
	}
	return reserved, model.ProfileEnv(reserved), nil
}

type made struct{ vm, record, reservations bool }

// invariant: the rollback is best effort and runs every step whatever fails, so a
// half-made VM leaves as little as it can; what it reports is the create's failure.
func (s *Machinist) undoCreate(ctx context.Context, name string, m *made) {
	if m.reservations && s.rel != nil {
		_ = s.rel.ReleaseVM(ctx, name)
	}
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

func (s *Machinist) Save(r model.VMRecord) error { return s.writeRecord(r) }

func (s *Machinist) writeRecord(r model.VMRecord) error {
	return s.recs.Put(context.Background(), storedOf(r))
}
