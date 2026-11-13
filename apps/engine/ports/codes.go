package ports

import (
	"strconv"
	"strings"

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
)

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
