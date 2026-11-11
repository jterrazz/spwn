package kit

import (
	"errors"
	"strconv"

	"github.com/jterrazz/studio-station/src/core/fail"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/surface/cli/words"
)

func machinesFailure(err error) (*fail.Error, bool) {
	var unknown *model.UnknownMachineError
	var thisMac *model.ThisMachineError
	var notMac *model.NotAMacError
	var macOnly *model.OnlyOnMacError
	var taken *model.NameTakenError
	var bad *model.BadNameError
	var unreachable *model.UnreachableError
	var noEther *model.NoHardwareAddressError
	var timeout *model.RestartTimeoutError
	var refused *model.RestartRefusedError
	var terminal *model.NeedsTerminalError
	var sshConfig *model.SSHConfigError
	var record *model.RecordError
	if f, ok := vmFailure(err); ok {
		return f, true
	}
	switch {
	case errors.As(err, &unknown):
		return usage(unknown.Error(), "station machines"), true
	case errors.As(err, &thisMac) && thisMac.Verb == "forget":
		return usage(thisMac.Error(), "station machines"), true
	case errors.As(err, &thisMac):
		return usage(thisMac.Error(), "pick another machine: station machines lists them all"), true
	case errors.As(err, &notMac):
		return usage(notMac.Error(), notAMacHint(notMac)), true
	case errors.As(err, &macOnly):
		return usage(macOnly.Error(), "run it on a Mac"), true
	case errors.As(err, &taken):
		return usage(taken.Error(), takenHint(taken)), true
	case errors.As(err, &bad):
		return usage(bad.Error(), "use lowercase letters, digits and \"-\", 1 to 40 characters"), true
	case errors.As(err, &unreachable):
		return failed(unreachable.Error(), "station machines", err), true
	case errors.As(err, &noEther):
		return failed(noEther.Error(), "wake it by hand once; station keeps its address at its next answer", err), true
	case errors.As(err, &timeout):
		return failed(timeout.Error(), "station machines "+timeout.Name, err), true
	case errors.As(err, &refused):
		return failed(refused.Error(), words.PasswordlessRestart(refused.Name), err), true
	case errors.As(err, &terminal):
		return Translate(fail.New(fail.NeedsTerminal, words.NeedsTerminal, terminal.Verb, nil), ""), true
	case errors.As(err, &sshConfig):
		return failed(sshConfig.Error(), "check \""+sshConfig.Path+"\" and run it again", err), true
	case errors.As(err, &record):
		return failed(record.Error(), "repair or move away \""+record.Path+"\"", err), true
	}
	return nil, false
}

func notAMacHint(e *model.NotAMacError) string {
	if e.Verb == "forget" {
		return "station machines delete " + e.Name
	}
	return "run station machines for the hosts"
}

func takenHint(e *model.NameTakenError) string {
	if e.VM || e.Mac || e.Agent != "" {
		return "pick another name"
	}
	return "station machines forget " + e.Name + " first, to change it"
}

func vmFailure(err error) (*fail.Error, bool) {
	var notVM *model.NotAVMError
	var badLifetime *model.BadLifetimeError
	var unknownImage *model.UnknownImageError
	var noImage *model.NoImageError
	var noEngine *model.NoEngineError
	var tooBig *model.TooBigError
	var badSize *model.BadSizeError
	var noUnit *model.SizeNeedsUnitError
	var created *model.CreateFailedError
	var noBoot *model.DidNotBootError
	var unhealthy *model.UnhealthyError
	var down *model.EngineDownError
	var network *model.BadNetworkError
	var nothing *model.NothingToChangeError
	var shrinks *model.DiskShrinksError
	var kindRule *model.KindRuleError
	var badCheck *model.BadCheckError
	var serverLifetime *model.ServerLifetimeError
	var agent *model.AgentNotStartedError
	if f, ok := pipeFailure(err); ok {
		return f, true
	}
	switch {
	case errors.As(err, &down):
		return failed(down.Error(), "station engine start", err), true
	case errors.As(err, &agent):
		return failed(agent.Name+" is ready, but its agent did not start: "+Translate(agent.Cause, "").Message, "station agents start \"<task>\" --on "+agent.Name, err), true
	case errors.As(err, &serverLifetime):
		return usage(serverLifetime.Error(), "station machines create "+serverLifetime.Name+" --server"), true
	case errors.As(err, &kindRule):
		return usage(kindRule.Error(), kindRule.Fix), true
	case errors.As(err, &badCheck):
		return usage(badCheck.Error(), "--check https://<name>:6443/readyz, or --check <name>:6443"), true
	case errors.As(err, &nothing):
		return usage(nothing.Error(), "station machines edit "+nothing.Name+" --memory 8G"), true
	case errors.As(err, &shrinks):
		return usage(shrinks.Error(), shrinks.Name+" has "+strconv.FormatFloat(float64(shrinks.Has)/1e9, 'f', 1, 64)+" GB: ask for more"), true
	case errors.As(err, &network):
		if network.Bridge {
			return usage(network.Error(), "share its ports instead: station ports share"), true
		}
		return usage(network.Error(), "station machines create --network nat"), true
	case errors.As(err, &noBoot):
		return failed(noBoot.Error(), keptFix(noBoot.Name), err), true
	case errors.As(err, &unhealthy):
		return failed(unhealthy.Error(), keptFix(unhealthy.Name), err), true
	case errors.As(err, &created):
		return failed(created.Error(), "fix the cause, then run station machines create again", err), true
	case errors.As(err, &notVM):
		return usage(notVM.Error(), notAVMHint(notVM)), true
	case errors.As(err, &badLifetime):
		return usage(badLifetime.Error(), "station machines create --persistent"), true
	case errors.As(err, &unknownImage):
		return usage(unknownImage.Error(), "station images for the images"), true
	case errors.As(err, &noImage) && noImage.Fetched:
		return usage("image "+noImage.Image+" is not on This Mac", "station images update "+noImage.Image), true
	case errors.As(err, &noImage):
		return usage(noImage.Error(), "station images build "+noImage.Image), true
	case errors.As(err, &noEngine):
		return usage(noEngine.Error(), "station apps install "+noEngine.Hypervisor), true
	case errors.As(err, &tooBig):
		if tooBig.LeftGB < 1 {
			return usage(tooBig.Error(), "stop a VM first: station machines"), true
		}
		return usage(tooBig.Error(), "ask for less: --memory "+strconv.Itoa(tooBig.LeftGB)+"G"), true
	case errors.As(err, &noUnit):
		return usage(noUnit.Error(), "--"+noUnit.Flag+" "+noUnit.Value+"G"), true
	case errors.As(err, &badSize):
		return usage(badSize.Error(), "write a size with its unit: 512M, 8G, 1T"), true
	}
	return nil, false
}

// value: the two ways out for a VM a create kept: start it again, or delete it.
func keptFix(vm string) string {
	return "station machines start " + vm + ", or station machines delete " + vm
}

func notAVMHint(e *model.NotAVMError) string {
	switch e.Verb {
	case "delete", "stop":
		return "station machines restart " + e.Name
	}
	return "run station machines for the VMs"
}

func pipeFailure(err error) (*fail.Error, bool) {
	var notThisMac *model.NotThisMacError
	var stopped *model.StoppedError
	var closed *model.SSHDClosedError
	var running *model.RunningUnderAgentError
	switch {
	case errors.As(err, &notThisMac):
		return usage(notThisMac.Error(), words.ReachedThrough(notThisMac.Name, notThisMac.Mac)), true
	case errors.As(err, &stopped):
		return failed(stopped.Error(), "station machines start "+stopped.Name, err), true
	case errors.As(err, &closed):
		return failed(closed.Error(), "station machines "+closed.Name, err), true
	case errors.As(err, &running):
		return failed(running.Error(), "stop the persistent VMs and delete the ephemeral ones (station machines), then station engine start", err), true
	}
	return nil, false
}
