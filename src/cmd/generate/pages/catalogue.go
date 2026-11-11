package pages

import (
	appmodel "github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/page"
	pageapps "github.com/jterrazz/studio-station/src/pages/apps"
)

func thisMacContext() page.Context { return page.Context{Kind: "mac"} }

type app struct {
	name     string
	category catalog.Category
	method   catalog.Method
	state    catalog.State
	version  string
	newest   string
	detail   string
	why      appmodel.Why
	needs    string
}

func reports(list []app, pending bool) []appmodel.Report {
	out := make([]appmodel.Report, 0, len(list))
	for _, t := range list {
		r := appmodel.Report{
			App:     catalog.App{Name: t.name, Category: t.category},
			Method:  t.method,
			State:   t.state,
			Version: t.version,
			Newest:  t.newest,
			Managed: t.state == catalog.StateOK,
			Detail:  t.detail,
			Why:     t.why,
			Needs:   t.needs,
			Pending: pending,
		}
		if pending {
			r.State, r.Version, r.Detail, r.Why, r.Needs = catalog.StatePending, "", "", "", ""
		}
		out = append(out, r)
	}
	return out
}

func thisMac() []app {
	const (
		pm   = catalog.CategoryPackageManagers
		rt   = catalog.CategoryRuntimes
		apps = catalog.CategoryAIApps
		sk   = catalog.CategoryAgentSkills
	)
	mise := func(name, version string) app {
		return app{name: name, category: rt, method: catalog.MethodMise, state: catalog.StateDegraded, version: version, detail: "installed, mise not installed", why: appmodel.WhyMissing, needs: "mise"}
	}
	return []app{
		{name: "homebrew", category: pm, method: catalog.MethodScript, state: catalog.StateOK, version: "7.0.12-8-g57cac9d"},
		{name: "mise", category: pm, method: catalog.MethodBrew, state: catalog.StateAbsent},
		{name: "mas", category: pm, method: catalog.MethodBrew, state: catalog.StateAbsent},
		{name: "uv", category: pm, method: catalog.MethodBrew, state: catalog.StateOK, version: "0.12.3"},
		{name: "cocoapods", category: pm, method: catalog.MethodBrew, state: catalog.StateOK, version: "1.17.0"},
		{name: "pnpm", category: pm, method: catalog.MethodBrew, state: catalog.StateDegraded, version: "11.21.0", newest: "11.22.1", detail: "outdated", why: appmodel.WhyOutdated},
		{name: "skills", category: pm, method: catalog.MethodBun, state: catalog.StateOK, version: "1.5.22"},
		mise("bun", ""),
		mise("go", ""),
		mise("java", "17.0.20"),
		mise("python", "3.14.6"),
		{name: "chatgpt", category: apps, method: catalog.MethodCask, state: catalog.StateDegraded, detail: "on PATH, not managed by cask", why: appmodel.WhyUnmanaged},
		{name: "claude-desktop", category: apps, method: catalog.MethodCask, state: catalog.StateOK, version: "1.22209.3,babe11577dfefe3e209c06bd674628d862f0dbae"},
		{name: "typewhisper", category: apps, method: catalog.MethodCask, state: catalog.StateAbsent},
		{name: "frontend-design", category: sk, method: catalog.MethodSkill, state: catalog.StateAbsent, detail: "skills not installed", why: appmodel.WhyMissing, needs: "skills"},
		{name: "modern-web-guidance", category: sk, method: catalog.MethodSkill, state: catalog.StateOK, detail: "currency unknown"},
		{name: "better-ui", category: sk, method: catalog.MethodSkill, state: catalog.StateDegraded, detail: "outdated", why: appmodel.WhyOutdated},
	}
}

func lateMac() []app {
	out := thisMac()
	for i := range out {
		if out[i].method == catalog.MethodBrew {
			out[i].state, out[i].version, out[i].detail, out[i].why, out[i].needs = catalog.StateUnknown, "", "no answer in 5s", appmodel.WhyLate, ""
		}
	}
	return out
}

func Apps() []Fixture {
	return []Fixture{
		{"apps.loading", pageapps.List(reports(thisMac(), true), builtin().Apps, thisMacContext(), false)},
		{"apps.empty", pageapps.List(nil, builtin().Apps, thisMacContext(), false)},
		{"apps.error", pageapps.List(reports(lateMac(), false), builtin().Apps, thisMacContext(), false)},
		{"apps.full", pageapps.List(reports(thisMac(), false), builtin().Apps, thisMacContext(), false)},
	}
}
