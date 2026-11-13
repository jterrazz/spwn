package architecture_test

import (
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

// value: the machines every call is judged against (ARCHITECTURE 14.3): the
// Macs and VMs of the worlds, by the name their outcome file takes.
type reference struct {
	name    string
	profile machine.Profile
}

func references() []reference {
	mac := machine.Capabilities{Jobs: true, Engine: true}
	laptop := mac
	laptop.Battery, laptop.Lid, laptop.Thermal = true, true, true
	return []reference{
		{"laptop", machine.Profile{Kind: machine.Computer, System: machine.System{OS: machine.MacOS}, Caps: laptop}},
		{"desktop", machine.Profile{Kind: machine.Computer, System: machine.System{OS: machine.MacOS}, Caps: mac}},
		{"workspace", machine.Profile{Kind: machine.Workspace, System: machine.System{OS: machine.Linux}, Caps: machine.Capabilities{Jobs: true}}},
		{"server", machine.Profile{Kind: machine.Server, System: machine.System{OS: machine.Linux}}},
	}
}

// invariant: the folders of specs/usecases/ that are no call's: the
// behaviors that span calls, and the engine's jobs (engine/<job>).
func spanning() []string {
	return []string{"root", "remote", "output", "errors", "migrate", "security"}
}

func engineJobs() []string {
	return []string{"keep", "share", "expire", "awake", "sample", "exposure", "events"}
}

const (
	outcomeDone      = "done"
	outcomeNotFor    = "not-for-this-machine"
	outcomeElsewhere = "on-another-machine"
	outcomeWithout   = "without"
	coverageRule     = "coverage"
)

// value: the outcomes a call declares: done, each code it may end in, one
// per reference machine its Applies refuses, one per need it refuses
// without, and its run on another machine when it is routable.
func outcomes(m call.Meta, refs []reference) []string {
	out := []string{outcomeDone}
	for _, code := range m.Errors {
		out = append(out, strings.ReplaceAll(string(code), ".", "-"))
	}
	for _, r := range refs {
		if m.Applies != nil && !m.Applies(r.profile).OK {
			out = append(out, outcomeNotFor+"-"+r.name)
		}
	}
	for _, n := range m.Needs {
		out = append(out, outcomeWithout+"-"+string(n))
	}
	if m.Where != call.InProcess {
		out = append(out, outcomeElsewhere)
	}
	return out
}

// value: the outcome a story file tells: the declared one its name is, or
// starts with; any other name is a story of done. A name in a refusal's
// namespace that the call does not declare names an outcome that does not
// exist.
func outcomeOf(stem string, declared, namespaces []string) (string, bool) {
	best := ""
	for _, o := range declared {
		if o != outcomeDone && (stem == o || strings.HasPrefix(stem, o+"-")) && len(o) > len(best) {
			best = o
		}
	}
	if best != "" {
		return best, true
	}
	for _, ns := range namespaces {
		if strings.HasPrefix(stem, ns+"-") {
			return "", false
		}
	}
	return outcomeDone, true
}

func namespacesOf(metas []call.Meta) []string {
	out := []string{outcomeNotFor, outcomeElsewhere, outcomeWithout}
	for _, m := range metas {
		for _, code := range m.Errors {
			if ns, _, ok := strings.Cut(string(code), "."); ok && !slices.Contains(out, ns) {
				out = append(out, ns)
			}
		}
	}
	return out
}

// value: the use-case documents, as paths under specs/usecases/.
func useCaseDocs(t *testing.T) []string {
	t.Helper()
	root := filepath.Join(repoRoot(t), "specs", "usecases")
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && strings.HasPrefix(d.Name(), "__") {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".spec.yaml") {
			rel, _ := filepath.Rel(root, path)
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// invariant: rule coverage (13.3 rule 9, AR-04, AR-07): every call has its
// folder under specs/usecases/ (the folder pair is its name) and a story for
// each outcome it declares; a document names only a real call, engine job
// or spanning folder, and only an outcome its call declares.
func ruleCoverage(metas []call.Meta, refs []reference, docs []string) []finding {
	var out []finding
	byName := map[string]call.Meta{}
	for _, m := range metas {
		byName[m.Name] = m
	}
	namespaces := namespacesOf(metas)
	told := map[string]map[string]bool{}
	for _, doc := range docs {
		parts := strings.Split(doc, "/")
		where := "specs/usecases/" + doc
		if len(parts) != 3 {
			if !slices.Contains(spanning(), parts[0]) {
				out = append(out, finding{coverageRule, where, "a use-case document sits at <family>/<verb>/<outcome>.spec.yaml"})
			}
			continue
		}
		folder, verb, stem := parts[0], parts[1], strings.TrimSuffix(parts[2], ".spec.yaml")
		if slices.Contains(spanning(), folder) || folder == "engine" && slices.Contains(engineJobs(), verb) {
			continue
		}
		name := folder + "." + verb
		m, ok := byName[name]
		if !ok {
			out = append(out, finding{coverageRule, where, "names " + name + ", which is no call, engine job or spanning folder"})
			continue
		}
		outcome, ok := outcomeOf(stem, outcomes(m, refs), namespaces)
		if !ok {
			out = append(out, finding{coverageRule, where, "names an outcome " + name + " does not declare"})
			continue
		}
		if told[name] == nil {
			told[name] = map[string]bool{}
		}
		told[name][outcome] = true
	}
	for _, m := range metas {
		var missing []string
		for _, o := range outcomes(m, refs) {
			if !told[m.Name][o] {
				missing = append(missing, o)
			}
		}
		if len(missing) > 0 {
			folder := "specs/usecases/" + strings.Replace(m.Name, ".", "/", 1) + "/"
			out = append(out, finding{coverageRule, folder, "tells no story of " + strings.Join(missing, ", ")})
		}
	}
	return out
}

func registeredMetas(t *testing.T) []call.Meta {
	t.Helper()
	return allCalls(t).Metas()
}

func TestEveryCallHasItsUseCaseSpecs(t *testing.T) {
	for _, f := range ruleCoverage(registeredMetas(t), references(), useCaseDocs(t)) {
		t.Errorf("%s: %s", f.where, f.what)
	}
}

// invariant: AR-07: a machine kind added to the references is demanded its
// not-for-this-machine story by every call whose Applies refuses it.
func TestCoverageDemandsANewKindEverywhere(t *testing.T) {
	agent := reference{"agent", machine.Profile{Kind: machine.Kind("agent"), System: machine.System{OS: machine.Linux}}}
	metas := registeredMetas(t)
	got := ruleCoverage(metas, append(references(), agent), useCaseDocs(t))
	refused := 0
	for _, m := range metas {
		if m.Applies == nil || m.Applies(agent.profile).OK {
			continue
		}
		refused++
		folder := "specs/usecases/" + strings.Replace(m.Name, ".", "/", 1) + "/"
		if !slices.ContainsFunc(got, func(f finding) bool {
			return f.where == folder && strings.Contains(f.what, outcomeNotFor+"-agent")
		}) {
			t.Errorf("%s refuses the new kind and the coverage rule demanded no story of it", m.Name)
		}
	}
	if refused == 0 {
		t.Fatal("no call refuses the new kind: the mutation proves nothing")
	}
}

func TestCoverageRefusesADocumentOfNoCall(t *testing.T) {
	docs := append(useCaseDocs(t), "machines/teleport/done.spec.yaml", "machines/stop/not-for-this-machine-toaster.spec.yaml")
	got := ruleCoverage(registeredMetas(t), references(), docs)
	for _, want := range []string{"specs/usecases/machines/teleport/done.spec.yaml", "specs/usecases/machines/stop/not-for-this-machine-toaster.spec.yaml"} {
		if !slices.ContainsFunc(got, func(f finding) bool { return f.where == want }) {
			t.Errorf("the coverage rule let %s through", want)
		}
	}
}
