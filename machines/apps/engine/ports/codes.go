package ports

import (
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
)

const (
	CodeRangeFull fail.Code = "port.range-full"
	CodeNotFound  fail.Code = "port.not-found"
	CodeSignedOut fail.Code = "port.signed-out"
	CodeNeedsVM   fail.Code = "port.needs-vm"
	CodeElsewhere fail.Code = "port.on-another-mac"
	// value: a declaration that says no route, or one station knows not.
	CodeNeedsRoute   fail.Code = "port.needs-route"
	CodeUnknownRoute fail.Code = "port.unknown-route"
	CodeNotDeclared  fail.Code = "port.not-declared"
	// value: a share's audience is a word of catalog.Audience, and only the
	// owner on a Mac chooses one: a guest's share takes its rights'.
	CodeUnknownAudience fail.Code = "port.unknown-audience"
	CodeAudienceInside  fail.Code = "port.audience-inside-vm"
	// value: an engine that does not read a reserve's audience, and a share
	// that came back with another audience than the one asked.
	CodeEngineTooOld    fail.Code = "port.engine-too-old"
	CodeAudienceNotKept fail.Code = "port.audience-not-kept"
)

func EngineTooOld(mac, version string) error {
	return fail.New(fail.FarTooOld, CodeEngineTooOld, mac, map[string]string{"version": version})
}

func AudienceNotKept(mac, vm string, port int, asked, got catalog.Audience, release error) error {
	detail := map[string]string{"mac": mac, "vm": vm, "port": strconv.Itoa(port), "asked": string(asked), "got": string(got)}
	if release == nil {
		detail["released"] = "yes"
	} else {
		detail["cause"] = release.Error()
	}
	return fail.New(fail.Failed, CodeAudienceNotKept, vm+":"+strconv.Itoa(port), detail)
}

func UnknownAudience(to string) error {
	return fail.New(fail.Refused, CodeUnknownAudience, to, map[string]string{"audiences": strings.Join(catalog.Audience("").Enum(), ", ")})
}

func AudienceInside(vm string, port int, to string) error {
	return fail.New(fail.Refused, CodeAudienceInside, strconv.Itoa(port), map[string]string{"vm": vm, "to": to})
}

func NotFound(vm string, port int) error {
	return fail.New(fail.Refused, CodeNotFound, vm+":"+strconv.Itoa(port), map[string]string{"vm": vm})
}

func SignedOut(mac string) error { return fail.New(fail.Failed, CodeSignedOut, mac, nil) }

func (e *VMFull) Refusal() error {
	return fail.New(fail.Failed, CodeRangeFull, e.VM, map[string]string{"max": strconv.Itoa(e.Max)})
}

func NeedsRoute(port int, routes []string) error {
	return fail.New(fail.Refused, CodeNeedsRoute, strconv.Itoa(port), map[string]string{"routes": strings.Join(routes, ", ")})
}

func UnknownRoute(via string, routes []string) error {
	return fail.New(fail.Refused, CodeUnknownRoute, via, map[string]string{"routes": strings.Join(routes, ", ")})
}

func NotDeclared(machine string, port int) error {
	return fail.New(fail.Refused, CodeNotDeclared, strconv.Itoa(port)+" on "+machine, map[string]string{"machine": machine})
}
