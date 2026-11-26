package tart

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	iofs "io/fs"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	imagescontract "github.com/jterrazz/studio-station/apps/engine/images/contract"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

const (
	name         = "tart"
	commandGrace = 15 * time.Second
)

type Kernel interface {
	// value: the pids of the processes whose name starts with prefix.
	Named(prefix string) ([]int, error)
	// value: the paths of the files a process holds open.
	Files(pid int) ([]string, error)
}

type hypervisor struct {
	run    local.Runner
	fs     local.FS
	clock  local.Clock
	kernel Kernel
	files  files
}

func New(run local.Runner, fs local.FS, clock local.Clock, kernel Kernel, stateDir, vmsDir string) contract.Hypervisor {
	// value: stateDir is station's state folder, where the adapter keeps its
	// runs and its ledger; vmsDir the folder tart keeps each VM's own folder
	// in, by the VM's name (<TART_HOME>/vms).
	return &hypervisor{run: run, fs: fs, clock: clock, kernel: kernel, files: files{state: stateDir, vms: vmsDir}}
}

// invariant: the tart path is resolved at each call, absolute, because a launchd
// agent has no PATH of the owner's and tart may land after this value is built.
func (h *hypervisor) bin() string {
	if path, ok := h.run.LookPath(name); ok {
		return path
	}
	return name
}

func (h *hypervisor) Name() string { return name }

func (h *hypervisor) Package() string { return name }

func (h *hypervisor) Installed() bool {
	_, ok := h.run.LookPath(name)
	return ok
}

type entry struct {
	Source string `json:"Source"`
	Name   string `json:"Name"`
	Size   int64  `json:"Size"`
	State  string `json:"State"`
}

func parseList(data string) ([]contract.Disk, error) {
	var entries []entry
	if err := json.Unmarshal([]byte(data), &entries); err != nil {
		return nil, fmt.Errorf("tart list: unreadable output: %w", err)
	}
	disks := []contract.Disk{}
	for _, e := range entries {
		if e.Source != "local" {
			continue
		}
		disks = append(disks, contract.Disk{VM: e.Name, Bytes: e.Size * 1e9, State: stateOf(e.State)})
	}
	return disks, nil
}

func stateOf(s string) contract.DiskState {
	switch strings.ToLower(s) {
	case "running":
		return contract.DiskRunning
	case "suspended":
		return contract.DiskSuspended
	case "stopped":
		return contract.DiskStopped
	default:
		return contract.DiskUnknown
	}
}

func (h *hypervisor) exec(ctx context.Context, timeout time.Duration, args ...string) (string, error) {
	out, err := h.run.Output(ctx, local.Cmd{Name: h.bin(), Args: args, Timeout: timeout})
	if err != nil {
		return "", fmt.Errorf("tart %s: %w", args[0], err)
	}
	if out.ExitCode != 0 {
		return out.Stdout, fmt.Errorf("tart %s: exit %d: %s", args[0], out.ExitCode, strings.TrimSpace(out.Stderr))
	}
	return out.Stdout, nil
}

func seconds(d time.Duration) string {
	return strconv.Itoa(int((d + time.Second - 1) / time.Second))
}

func (h *hypervisor) Version(ctx context.Context) (string, error) {
	out, err := h.exec(ctx, 0, "--version")
	return strings.TrimSpace(out), err
}

func (h *hypervisor) Disks(ctx context.Context) ([]contract.Disk, error) {
	out, err := h.exec(ctx, 0, "list", "--source", "local", "--format", "json")
	if err != nil {
		return nil, err
	}
	return parseList(out)
}

func (h *hypervisor) Fetch(ctx context.Context, img imagescontract.ImageLocation, p call.Progress) error {
	if p != nil {
		p.Step("fetching", img.Ref)
	}
	_, err := h.exec(ctx, 0, "pull", img.Ref)
	if p != nil {
		p.Done("fetching", err)
	}
	return err
}

func (h *hypervisor) Clone(ctx context.Context, from imagescontract.ImageLocation, vm string) error {
	if _, err := h.exec(ctx, 0, "clone", from.Ref, vm); err != nil {
		return err
	}
	return h.mark(vm)
}

// invariant: the ledger holds one file per VM this adapter cloned, by the
// VM's name, written once its clone is made and removed once its delete is
// done; the file is the identity of the VM's folder the clone made. A VM is
// the adapter's only while its folder still has that identity: one deleted
// and made again by hand under the name, or renamed by hand onto it, is not,
// and its entry is dropped when read.
//
// A clone whose folder says no identity keeps no entry: the VM is made, and
// never offered to free, the safe side of a ledger that cannot tell it.
func (h *hypervisor) mark(vm string) error {
	if info, err := h.fs.Stat(h.files.folder(vm)); err == nil {
		if id, ok := local.IdentityOf(info); ok {
			return h.keep(vm, id)
		}
	}
	return nil
}

func (h *hypervisor) keep(vm string, id local.FileID) error {
	if err := h.fs.MkdirAll(h.files.ledger(), 0o700); err != nil {
		return err
	}
	return h.fs.WriteFile(h.files.cloned(vm), []byte(id), 0o600)
}

func (h *hypervisor) entry(vm string) (local.FileID, bool) {
	data, err := h.fs.ReadFile(h.files.cloned(vm))
	return local.FileID(data), err == nil
}

func (h *hypervisor) forget(vm string) error {
	if err := h.fs.Remove(h.files.cloned(vm)); err != nil && !errors.Is(err, iofs.ErrNotExist) {
		return err
	}
	return nil
}

func (h *hypervisor) Cloned(context.Context) ([]string, error) {
	entries, err := h.fs.ReadDir(h.files.ledger())
	if errors.Is(err, iofs.ErrNotExist) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		switch h.still(e.Name()) {
		case held:
			out = append(out, e.Name())
		case gone:
			_ = h.forget(e.Name())
		case unknown:
		}
	}
	return out, nil
}

type standing int

const (
	held standing = iota
	gone
	unknown
)

// invariant: an entry stands while the VM's folder has the identity it
// keeps; a folder missing, or of another identity, is gone; one the file
// system cannot say is unknown, never the adapter's this read, and kept.
func (h *hypervisor) still(vm string) standing {
	kept, ok := h.entry(vm)
	if !ok {
		return unknown
	}
	info, err := h.fs.Stat(h.files.folder(vm))
	switch {
	case errors.Is(err, iofs.ErrNotExist):
		return gone
	case err != nil:
		return unknown
	}
	if id, ok := local.IdentityOf(info); ok && id == kept {
		return held
	}
	return gone
}

func (h *hypervisor) Forget(_ context.Context, vm string) error { return h.forget(vm) }

func (h *hypervisor) Rename(ctx context.Context, from, to string) error {
	// invariant: the entry moves with the VM, the identity it keeps unchanged,
	// as a rename keeps a folder's; a VM the adapter did not clone leaves no
	// entry under its new name, so one a delete left there is dropped. The
	// entry is read before the rename, so a read meanwhile cannot lose it.
	kept, ours := h.entry(from)
	if _, err := h.exec(ctx, 0, "rename", from, to); err != nil {
		return err
	}
	h.files.drop(h.fs, from)
	if !ours {
		return h.forget(to)
	}
	if err := h.keep(to, kept); err != nil {
		return err
	}
	return h.forget(from)
}

func (h *hypervisor) Resize(ctx context.Context, vm string, s machines.Size) error {
	args := []string{"set", vm}
	if s.CPUs > 0 {
		args = append(args, "--cpu", strconv.Itoa(s.CPUs))
	}
	if s.Memory > 0 {
		args = append(args, "--memory", strconv.FormatInt((s.Memory+1<<20-1)>>20, 10))
	}
	if s.Disk > 0 {
		args = append(args, "--disk-size", strconv.FormatInt((s.Disk+1e9-1)/1e9, 10))
	}
	_, err := h.exec(ctx, 0, args...)
	return err
}

func (h *hypervisor) Shutdown(ctx context.Context, vm string, grace time.Duration) error {
	_, err := h.exec(ctx, grace+commandGrace, "stop", "--timeout", seconds(grace), vm)
	return err
}

func (h *hypervisor) Delete(ctx context.Context, vm string) error {
	if _, err := h.exec(ctx, 0, "delete", vm); err != nil {
		return err
	}
	h.files.drop(h.fs, vm)
	return h.forget(vm)
}

func (h *hypervisor) Discard(ctx context.Context, img imagescontract.ImageLocation) error {
	if _, err := h.exec(ctx, 0, "delete", img.Ref); err != nil {
		return err
	}
	return h.forget(img.Ref)
}

func (h *hypervisor) Address(ctx context.Context, vm string, wait time.Duration) (netip.Addr, error) {
	out, err := h.exec(ctx, wait+commandGrace, "ip", vm, "--wait", seconds(wait))
	if err != nil {
		return netip.Addr{}, err
	}
	addr, err := netip.ParseAddr(strings.TrimSpace(out))
	if err != nil {
		return netip.Addr{}, fmt.Errorf("tart ip: unreadable address: %w", err)
	}
	return addr, nil
}
