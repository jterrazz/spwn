package calls

import (
	"context"
	"errors"

	"github.com/jterrazz/studio-station/apps/engine/machines/contract"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/machines"
)

type AddIn struct {
	Name string
	// value: the login and the key ssh uses there; empty takes the owner's.
	User, Key string
}

var Add = call.Call[Deps, AddIn, machines.Added]{
	Name: "machines.add", Where: call.InProcess, Applies: machines.AddsMacs,
	Errors: []fail.Code{BadName, NameTaken, call.Unreachable, call.NoStation},
	Run: func(ctx context.Context, d Deps, in AddIn, _ call.Progress) (machines.Added, error) {
		if err := d.Machines.CheckMac(); err != nil {
			return machines.Added{}, err
		}
		if err := d.Machines.AddMac(ctx, in.Name, machines.MacRecord{User: in.User, Key: in.Key}); err != nil {
			return machines.Added{}, err
		}
		added, err := d.Machines.Reach(ctx, in.Name)
		if err != nil {
			_ = d.Machines.ForgetMac(ctx, in.Name)
			return machines.Added{}, err
		}
		return added, nil
	},
}

type ForgetIn struct {
	Name    string
	Confirm string
}

var Forget = call.Call[Deps, ForgetIn, struct{}]{
	Name: "machines.forget", Where: call.InProcess, Applies: machine.Anywhere, Confirm: confirm.Forget,
	Errors: []fail.Code{confirm.YesNeeded},
	Run: func(ctx context.Context, d Deps, in ForgetIn, _ call.Progress) (struct{}, error) {
		if err := d.Machines.CheckMac(); err != nil {
			return struct{}{}, err
		}
		if _, err := d.Machines.CheckRemoteMac(in.Name, "forget"); err != nil {
			return struct{}{}, err
		}
		if err := confirm.Require(confirm.For(confirm.Forget, d.Kind), in.Name, in.Confirm); err != nil {
			return struct{}{}, err
		}
		return struct{}{}, d.Machines.ForgetMac(ctx, in.Name)
	},
}

type UnlockIn struct {
	Name string
	// value: the pre-boot session asks the password at a terminal.
	Interactive bool
}

var Unlock = call.Call[Deps, UnlockIn, machines.Unlocked]{
	Name: "machines.unlock", Where: call.InProcess, Applies: machine.Anywhere,
	Run: func(ctx context.Context, d Deps, in UnlockIn, _ call.Progress) (machines.Unlocked, error) {
		if err := d.Machines.CheckMac(); err != nil {
			return machines.Unlocked{}, err
		}
		mac, err := d.Machines.CheckRemoteMac(in.Name, "unlock")
		if err != nil {
			return machines.Unlocked{}, err
		}
		if d.Machines.Answers(ctx, mac) {
			return machines.Unlocked{Name: in.Name, Already: true}, nil
		}
		if err := d.Machines.Unlock(ctx, in.Name, in.Interactive); err != nil {
			return machines.Unlocked{}, err
		}
		return machines.Unlocked{Name: in.Name, Answers: d.Machines.AwaitAnswer(ctx, mac)}, nil
	},
}

type WakeIn struct {
	Name string
	// value: tell the owner once, when it answers again.
	Tell bool
}

var Wake = call.Call[Deps, WakeIn, machines.Woken]{
	Name: "machines.wake", Where: call.InProcess, Applies: machine.Anywhere,
	Run: func(ctx context.Context, d Deps, in WakeIn, _ call.Progress) (machines.Woken, error) {
		// invariant: SPEC B5.14: a Mac is woken by a Wake-on-LAN packet sent
		// from this Mac to the hardware address its last answer said; one that
		// answers already is left as it is, and one never heard has no
		// address to send to.
		if err := d.Machines.CheckMac(); err != nil {
			return machines.Woken{}, err
		}
		mac, err := d.Machines.CheckRemoteMac(in.Name, "wake")
		if err != nil {
			return machines.Woken{}, err
		}
		if d.Machines.Answers(ctx, mac) {
			return machines.Woken{Name: in.Name, Already: true}, nil
		}
		var none *machines.NoHardwareAddressError
		if err := d.Machines.Wake(ctx, mac, in.Tell); errors.As(err, &none) {
			return machines.Woken{}, &machines.NoHardwareAddressError{Name: in.Name}
		} else if err != nil {
			return machines.Woken{}, err
		}
		return machines.Woken{Name: in.Name, Sent: true, Tell: in.Tell}, nil
	},
}

type RemindIn struct{ Name string }

var Remind = call.Call[Deps, RemindIn, machines.Reminded]{
	Name: "machines.remind", Where: call.InProcess, Applies: machine.Anywhere,
	Run: func(ctx context.Context, d Deps, in RemindIn, _ call.Progress) (machines.Reminded, error) {
		// invariant: SPEC B5.14: "Tell me when it wakes" arms the one mac_back
		// notice and sends nothing, so it needs no hardware address; a Mac
		// that answers already is awake, and nothing is armed.
		if err := d.Machines.CheckMac(); err != nil {
			return machines.Reminded{}, err
		}
		mac, err := d.Machines.CheckRemoteMac(in.Name, "remind")
		if err != nil {
			return machines.Reminded{}, err
		}
		if d.Machines.Answers(ctx, mac) {
			return machines.Reminded{Name: in.Name, Already: true}, nil
		}
		if err := d.Machines.Remind(ctx, mac); err != nil {
			return machines.Reminded{}, err
		}
		return machines.Reminded{Name: in.Name}, nil
	},
}

type RestartMacIn struct {
	Name    string
	Confirm string
}

var RestartMac = call.Call[Deps, RestartMacIn, machines.Back]{
	Name: "machines.restart-mac", Where: call.InProcess, Applies: machine.Anywhere, Confirm: confirm.RestartMac,
	Errors: []fail.Code{confirm.YesNeeded},
	Run: func(ctx context.Context, d Deps, in RestartMacIn, _ call.Progress) (machines.Back, error) {
		// invariant: a Mac restarts by a planned restart that unlocks itself,
		// sent from this Mac; the answer waits for it to answer again, runs
		// again each VM it paused, and says how they are now.
		if err := d.Machines.CheckMac(); err != nil {
			return machines.Back{}, err
		}
		mac, err := d.Machines.CheckRemoteMac(in.Name, "restart")
		if err != nil {
			return machines.Back{}, err
		}
		if err = confirm.Require(confirm.For(confirm.RestartMac, d.Kind), in.Name, in.Confirm); err != nil {
			return machines.Back{}, err
		}
		paused, _ := d.Machines.VMsOf(ctx, mac)
		report, err := d.Machines.Restart(ctx, in.Name)
		if err != nil {
			return machines.Back{}, err
		}
		for _, vm := range paused {
			if vm.State == machines.Running {
				_ = d.Machines.StartOn(ctx, mac, vm.Name)
			}
		}
		now, _ := d.Machines.VMsOf(ctx, mac)
		return machines.Back{Name: in.Name, Waited: report.Waited, VMs: pausedNow(paused, now)}, nil
	},
}

func pausedNow(paused, now []machines.VM) []machines.VM {
	var out []machines.VM
	for _, before := range machines.Ordered(paused) {
		if before.State != machines.Running {
			continue
		}
		after := before
		after.State = machines.UnknownState
		for _, vm := range now {
			if vm.Name == before.Name {
				after = vm
			}
		}
		out = append(out, after)
	}
	return out
}

type SSHIn struct{ Name string }

var SSH = call.Call[Deps, SSHIn, machines.SSHTarget]{
	Name: "machines.ssh", Where: call.InProcess, Applies: machine.Anywhere,
	Run: func(ctx context.Context, d Deps, in SSHIn, _ call.Progress) (machines.SSHTarget, error) {
		return d.Machines.SSH(ctx, in.Name)
	},
}

func AdoptV1(hosts contract.MacsPrefs, vmUser contract.VMUserPrefs, macs map[string]machines.MacRecord, user string) error {
	// value: the Macs and the VM user a v1 config.json named, kept where
	// machines keeps them; setup's layout step converts the file once.
	if err := hosts.Set(macs); err != nil {
		return err
	}
	return vmUser.Set(user)
}
