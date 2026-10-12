package world

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"spwn.sh/apps/cli/cliproject"
	"spwn.sh/apps/cli/runtimeres"
	"spwn.sh/packages/project"
	"spwn.sh/packages/transpile/source"
	"spwn.sh/packages/world"
)

func mergeDepPolicy(a, b world.DepPolicy) world.DepPolicy {
	if len(a.Allow) == 0 && len(a.Deny) == 0 {
		return b
	}
	if len(b.Allow) == 0 && len(b.Deny) == 0 {
		return a
	}
	out := world.DepPolicy{Deny: append([]string{}, a.Deny...)}
	denySeen := map[string]struct{}{}
	for _, d := range a.Deny {
		denySeen[d] = struct{}{}
	}
	for _, d := range b.Deny {
		if _, ok := denySeen[d]; !ok {
			out.Deny = append(out.Deny, d)
			denySeen[d] = struct{}{}
		}
	}
	if len(a.Allow) > 0 && len(b.Allow) > 0 {
		bAllow := map[string]struct{}{}
		for _, x := range b.Allow {
			bAllow[x] = struct{}{}
		}
		for _, x := range a.Allow {
			if _, ok := bAllow[x]; ok {
				out.Allow = append(out.Allow, x)
			}
		}
	}
	return out
}

type projectWorld struct {
	Project    *project.Project
	Name       string
	Agents     []string
	Workspaces []string
	Manifest   world.Manifest
	// Knowledge is the absolute host path to bind into
	// /world/knowledge/, resolved from the manifest's
	// worlds.<name>.knowledge key relative to the project root.
	// Empty when the manifest declares no knowledge path — in which
	// case the spawn pipeline skips the bind mount entirely and the
	// rendered per-agent CLAUDE.md omits every reference to
	// /world/knowledge/.
	Knowledge string
	// Runtime is the agent-declared runtime name ("claude-code",
	// "codex") resolved via source.ResolveRuntime across every agent
	// in this world. Empty falls back to the architect's default
	// ("claude-code") at spawn time.
	Runtime string
}

func loadProject() (*project.Project, error) {
	return cliproject.Find()
}

func sortedWorldNames(p *project.Project) []string {
	if p == nil || p.Manifest == nil {
		return nil
	}
	names := make([]string, 0, len(p.Manifest.Worlds))
	for name := range p.Manifest.Worlds {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func resolveProjectWorld(p *project.Project, name, backendOverride string) (*projectWorld, error) {
	if p == nil || p.Manifest == nil {
		return nil, fmt.Errorf("no project loaded")
	}
	if len(p.Manifest.Worlds) == 0 {
		return nil, fmt.Errorf("spwn.yaml has no worlds")
	}
	if name == "" {
		name = sortedWorldNames(p)[0]
	}
	w, ok := p.Manifest.Worlds[name]
	if !ok {
		return nil, fmt.Errorf("world %q is not defined in spwn.yaml", name)
	}

	pkgSet := map[string]struct{}{}
	var pkgs []string
	add := func(t string) {
		if t == "" {
			return
		}
		if _, seen := pkgSet[t]; seen {
			return
		}
		pkgSet[t] = struct{}{}
		pkgs = append(pkgs, t)
	}
	for _, t := range p.Manifest.Deps {
		add(t)
	}
	agentPath := map[string]string{}
	for _, a := range p.Agents {
		agentPath[a.Name] = a.Path
	}
	for _, a := range p.OrphanAgents {
		agentPath[a.Name] = a.Path
	}
	policies := map[string]world.DepPolicy{}
	for _, aname := range w.Agents {
		dir, ok := agentPath[aname]
		if !ok {
			continue
		}
		am, err := world.LoadAgentManifest(dir)
		if err != nil || am == nil {
			continue
		}
		for _, t := range am.Deps {
			add(t)
		}
		for ref, p := range am.DepPolicies {
			policies[ref] = mergeDepPolicy(policies[ref], world.DepPolicy{Allow: p.Allow, Deny: p.Deny})
		}
	}

	m := world.Manifest{Deps: pkgs, DepPolicies: policies}

	var knowledge string
	if kp := strings.TrimSpace(w.Knowledge); kp != "" {
		if filepath.IsAbs(kp) {
			knowledge = kp
		} else {
			knowledge = filepath.Join(p.Root, kp)
		}
	}

	var runtimeName string
	if src, srcErr := source.Load(p.Root); srcErr == nil && src != nil {
		rn, rnErr := runtimeres.Resolve(src, backendOverride)
		if rnErr != nil {
			return nil, rnErr
		}
		runtimeName = rn
	}

	return &projectWorld{
		Project:    p,
		Name:       name,
		Agents:     append([]string(nil), w.Agents...),
		Workspaces: append([]string(nil), w.Workspaces...),
		Manifest:   m,
		Knowledge:  knowledge,
		Runtime:    runtimeName,
	}, nil
}
