package calls

import (
	"errors"
	"strconv"

	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	imagescalls "github.com/jterrazz/studio-station/apps/engine/images/calls"
	"github.com/jterrazz/studio-station/apps/engine/machines"
)

const (
	NotEnoughMemory fail.Code = "machines.not-enough-memory"
	NoImage         fail.Code = "machines.no-image"
	EngineDown      fail.Code = "machines.engine-down"
	BadLifetime     fail.Code = "machines.bad-lifetime"
	ServerNeedsName fail.Code = "machines.server-needs-name"
	CheckForServers fail.Code = "machines.check-for-servers"
	ServerBridged   fail.Code = "machines.server-bridged"
	BadCheck        fail.Code = "machines.bad-check"
	BadCreator      fail.Code = "machines.bad-creator"
	UnknownImage    fail.Code = "machines.unknown-image"
	ServerTemplate  fail.Code = "machines.server-template"
	BadNetwork      fail.Code = "machines.bad-network"
	NoHypervisor    fail.Code = "machines.no-hypervisor"
	BadName         fail.Code = "machines.bad-name"
	NameTaken       fail.Code = "machines.name-taken"
	BeingCreated    fail.Code = "machines.being-created"
	BadPurpose      fail.Code = "machines.bad-purpose"
)

// value: the refusals both creates declare, each coded by Worded; each
// create adds those only it can end in, and a missing hypervisor is their
// need, refused before they run.
func createRefusals() []fail.Code {
	return []fail.Code{confirm.YesNeeded, NotEnoughMemory, NoImage, EngineDown, BadName, NameTaken, BadCreator, BadPurpose, UnknownImage, BadNetwork, imagescalls.NoLinuxBinary}
}

func workspaceRefusals() []fail.Code {
	return append(createRefusals(), BadLifetime, CheckForServers, ServerTemplate)
}

func serverRefusals() []fail.Code {
	return append(createRefusals(), ServerNeedsName, ServerBridged, BadCheck)
}

func Worded(err error) error {
	// invariant: each refusal of a create, an add or a delete is one code; its words
	// are built from its subject and its detail alone, the Mac it was
	// refused on among them, so a far Mac's refusal reads the same here.
	if err == nil {
		return nil
	}
	refused := func(t fail.Type, code fail.Code, subject string, detail map[string]string) error {
		e := fail.New(t, code, subject, detail)
		e.Cause = err
		return e
	}
	var (
		tooBig *machines.TooBigError
		noImg  *machines.NoImageError
		down   *machines.EngineDownError
		life   *machines.BadLifetimeError
		check  *machines.BadCheckError
		by     *machines.BadCreatorError
		image  *machines.UnknownImageError
		tmpl   *machines.ServerTemplateError
		net    *machines.BadNetworkError
		hyper  *machines.NoEngineError
		bad    *machines.BadNameError
		taken  *machines.NameTakenError
		born   *machines.BeingCreatedError
		why    *machines.BadPurposeError
	)
	switch {
	case errors.As(err, &tooBig):
		return refused(fail.Refused, NotEnoughMemory, tooBig.Mac, map[string]string{"asked_gb": strconv.Itoa(tooBig.AskedGB), "left_gb": strconv.Itoa(tooBig.LeftGB)})
	case errors.As(err, &noImg):
		return refused(fail.Refused, NoImage, noImg.Image, map[string]string{"mac": noImg.Mac, "fetched": yes(noImg.Fetched)})
	case errors.As(err, &down):
		return refused(fail.EngineStopped, EngineDown, down.Mac, map[string]string{"installed": yes(down.Installed)})
	case errors.As(err, &life):
		return refused(fail.Refused, BadLifetime, life.Value, nil)
	case errors.Is(err, ServerNeedsNameRule):
		return refused(fail.Refused, ServerNeedsName, "", nil)
	case errors.Is(err, CheckForServersRule):
		return refused(fail.Refused, CheckForServers, "", nil)
	case errors.Is(err, ServerBridgedRule):
		return refused(fail.Refused, ServerBridged, "", nil)
	case errors.As(err, &check):
		return refused(fail.Refused, BadCheck, check.Value, nil)
	case errors.As(err, &by):
		return refused(fail.Refused, BadCreator, by.Session, nil)
	case errors.As(err, &image):
		return refused(fail.NotFound, UnknownImage, image.Image, nil)
	case errors.As(err, &tmpl):
		return refused(fail.Refused, ServerTemplate, tmpl.Name, nil)
	case errors.As(err, &net):
		return refused(fail.Refused, BadNetwork, net.Value, map[string]string{"bridge": yes(net.Bridge)})
	case errors.As(err, &hyper):
		return refused(fail.Refused, NoHypervisor, hyper.Hypervisor, nil)
	case errors.As(err, &bad):
		return refused(fail.Refused, BadName, bad.Name, nil)
	case errors.As(err, &taken):
		return refused(fail.Refused, NameTaken, taken.Name, map[string]string{"by": takenBy(taken), "on": taken.On, "agent": taken.Agent})
	case errors.As(err, &why):
		return refused(fail.Refused, BadPurpose, why.Purpose, nil)
	case errors.As(err, &born):
		return refused(fail.Refused, BeingCreated, born.Name, map[string]string{"mac": born.Mac})
	}
	return err
}

func yes(b bool) string {
	if b {
		return "yes"
	}
	return ""
}

// value: what holds a taken name: a VM, a Mac, an agent, or a Mac this one
// already added.
func takenBy(e *machines.NameTakenError) string {
	switch {
	case e.Agent != "":
		return "agent"
	case e.Mac:
		return "mac"
	case e.VM:
		return "vm"
	}
	return "added"
}
