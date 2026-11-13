package architecture_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

const (
	rowModel        = "model"
	rowKernel       = "core"
	rowContract     = "contract"
	rowContracttest = "contracttest"
	rowCalls        = "calls"
	rowAdapter      = "adapter"
	rowEngineRun    = "engine/run"
	rowRemote       = "remote"
	rowPlatform     = "platform"
	rowSurface      = "surface"
	rowWorld        = "world"
	rowWire         = "wire"
	rowPresentation = "presentation"
)

type place struct {
	row    string
	family string
	tech   string
	sub    string
}

func hexagonal(top string) bool {
	return slices.Contains(families(), top) || top == "core" || top == "remote"
}

// invariant: which row of ARCHITECTURE 13.2 rules a package; ok is false for
// the packages today's import table of layers_test.go still rules (the
// tolerated top-level packages, catalog and cmd).
func placeOf(pkg string) (place, bool) {
	top, rest := topOf(pkg), restOf(pkg)
	if hexagonal(top) {
		p := place{family: top}
		switch {
		case rest == "contract":
			p.row = rowContract
		case rest == "contract/contracttest":
			p.row = rowContracttest
		case strings.HasPrefix(rest, "adapter/"):
			p.row, p.tech = rowAdapter, strings.TrimPrefix(rest, "adapter/")
		case top == "core":
			p.row = rowKernel
		case top == "remote":
			p.row = rowRemote
		case rest == "calls":
			p.row = rowCalls
		case top == "engine" && rest != "":
			p.row, p.sub = rowEngineRun, rest
		default:
			p.row = rowModel
		}
		return p, true
	}
	switch {
	case top == "platform":
		return place{row: rowPlatform}, true
	case top == "surface":
		return place{row: rowSurface}, true
	case top == "world":
		return place{row: rowWorld}, true
	case top == "wire":
		return place{row: rowWire}, true
	case slices.Contains(presentation(), top):
		return place{row: rowPresentation}, true
	}
	return place{}, false
}

func ioStd(ip string) bool {
	switch ip {
	case "net/netip", "net/url":
		return false
	case "os", "net", "syscall", "unsafe", "plugin", "C", "io/ioutil", "log", "flag", "runtime/debug", "runtime/cgo", "embed":
		return true
	}
	return strings.HasPrefix(ip, "os/") || strings.HasPrefix(ip, "net/") || strings.HasPrefix(ip, "log/") || strings.HasPrefix(ip, "testing")
}

func isStd(ip string) bool {
	first, _, _ := strings.Cut(ip, "/")
	return !strings.Contains(first, ".")
}

func surfaceLibs() []string {
	return []string{cobra, pflag, fang, lip, huh, tea, bubble, term, ansi, cprof}
}

func isModel(rel string) bool   { return slices.Contains(families(), rel) }
func isAdapter(rel string) bool { return strings.Contains(rel, "/adapter/") }
func isCalls(rel string) bool {
	return strings.HasSuffix(rel, "/calls") && slices.Contains(families(), topOf(rel))
}
func isContract(rel string) bool {
	return hexagonal(topOf(rel)) && restOf(rel) == "contract"
}
func isKernel(rel string) bool {
	return topOf(rel) == "core" && !isAdapter(rel) && !isContract(rel) && rel != "core/contract/contracttest"
}

// invariant: the import table of ARCHITECTURE 13.2, owned HERE for the
// families and the new plumbing; a module import is judged by its path, a
// standard one by ioStd, a third-party one by the row's own libraries.
// remote's adapters speak remote/api, its frames, as a family's adapter
// speaks its model; their tests run remote/serve as the far side, and any
// test may use its own family's fake; a platform package's fake (local/localtest)
// imports the package whose types it fakes.
func allowedModule(p place, pkg, rel string, test bool) bool {
	own := topOf(rel) == p.family && p.family != ""
	if test && (rel == pkg || (own && rel == p.family+"/contract/contracttest") || (own && rel == p.family+"/adapter/fake") || rel == "world") {
		return true
	}
	switch p.row {
	case rowModel:
		return (isKernel(rel) && rel != "core/call" && !strings.HasPrefix(rel, "core/call/")) || isModel(rel) || rel == "catalog"
	case rowKernel:
		return isKernel(rel) || rel == "catalog"
	case rowContract:
		return isModel(rel) || isKernel(rel) || isContract(rel) || rel == "catalog"
	case rowContracttest:
		return isModel(rel) || isKernel(rel) || isContract(rel) || rel == "catalog" || rel == "world" || (own && isAdapter(rel) && strings.HasSuffix(rel, "/adapter/fake"))
	case rowCalls:
		return isModel(rel) || isContract(rel) || isCalls(rel) || isKernel(rel) || rel == "catalog"
	case rowAdapter:
		if isAdapter(rel) {
			return false
		}
		return isContract(rel) || (own && isModel(rel)) || isKernel(rel) || rel == "catalog" || topOf(rel) == "platform" || (p.tech == "fake" && rel == "world") ||
			(p.family == "remote" && (rel == "remote/api" || (test && rel == "remote/serve")))
	case rowEngineRun:
		if p.sub == "feeds" && (rel == "page" || rel == "pages" || strings.HasPrefix(rel, "pages/")) {
			return true
		}
		return isCalls(rel) || isModel(rel) || isContract(rel) || isKernel(rel) || rel == "catalog" ||
			(topOf(rel) == "remote" && !isAdapter(rel)) || (topOf(rel) == "engine" && slices.Contains([]string{"process", "jobs", "feeds"}, restOf(rel)))
	case rowRemote:
		return isKernel(rel) || isModel(rel) || (topOf(rel) == "remote" && !isAdapter(rel))
	case rowPlatform:
		return (topOf(rel) == "platform" && test) || pkg == rel+"/"+rel[strings.LastIndex(rel, "/")+1:]+"test"
	case rowSurface:
		return isCalls(rel) || isKernel(rel) || (isModel(rel) && rel != "engine") || rel == "page" || rel == "pages" || strings.HasPrefix(rel, "pages/") ||
			rel == "output" || strings.HasPrefix(rel, "ui/render") || rel == "catalog" || topOf(rel) == "surface"
	case rowWorld:
		return isModel(rel) || isKernel(rel) || rel == "catalog"
	case rowWire:
		return true
	case rowPresentation:
		return !isContract(rel) && !isAdapter(rel)
	}
	return false
}

func allowedOutside(p place, ip string, test bool) bool {
	if test && (ip == "testing" || strings.HasPrefix(ip, "testing/") || ip == "os" || ip == "flag") {
		return true
	}
	if isStd(ip) {
		switch p.row {
		case rowPlatform, rowWire, rowPresentation:
			return true
		case rowContract, rowContracttest, rowAdapter:
			return !ioStd(ip) || (p.row == rowContracttest && ip == "testing")
		case rowModel, rowKernel, rowWorld, rowCalls, rowEngineRun:
			return !ioStd(ip) && ip != "io" && ip != "io/fs"
		}
		return !ioStd(ip)
	}
	switch p.row {
	case rowAdapter, rowWire, rowPresentation:
		return true
	case rowPlatform:
		return strings.HasPrefix(ip, "golang.org/x/sys/")
	case rowSurface:
		return slices.ContainsFunc(surfaceLibs(), func(l string) bool { return ip == l || strings.HasPrefix(ip, l+"/") })
	}
	return false
}

// invariant: rule imports: each family's model pure, adapters isolated,
// surfaces through calls. The tolerated top-level packages stay under the
// import table of layers_test.go; a package this table rules imports one
// only where its row allows it, so no family reaches today's I/O.
func ruleFamilyImports(w *world) []finding {
	var out []finding
	for _, f := range w.files {
		p, ok := placeOf(f.pkg)
		if !ok {
			continue
		}
		for i, ip := range f.imports {
			allowed := allowedOutside(p, ip, f.test)
			if rel, mod := strings.CutPrefix(ip, module+"/"); mod {
				allowed = allowedModule(p, f.pkg, rel, f.test)
			}
			if !allowed {
				out = append(out, finding{"imports", f.at(f.syntax.Imports[i].Pos()), fmt.Sprintf("a %s package (%q) may not import %q", p.row, f.pkg, ip)})
			}
		}
	}
	return out
}

func TestFamilyImports(t *testing.T)                 { judge(t, "FamilyImports") }
func TestAFamilyModelReachesNoPlatform(t *testing.T) { judge(t, "FamilyImportsPlatform") }
func TestASurfaceReachesTheEngineOnlyThroughItsCalls(t *testing.T) {
	judge(t, "FamilyImportsSurfaceEngine")
}
