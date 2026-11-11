package catalog

import (
	"encoding/json"
	"fmt"
)

type wireLivesOn struct {
	Kinds   []MachineKind
	Method  Method          `json:",omitempty"`
	Install json.RawMessage `json:",omitempty"`
}

func (p LivesOn) MarshalJSON() ([]byte, error) {
	// invariant: a place travels with its install's method, so the far
	// side's answer decodes into the same install it named.
	w := wireLivesOn{Kinds: p.Kinds}
	if p.Install != nil {
		raw, err := json.Marshal(p.Install)
		if err != nil {
			return nil, err
		}
		w.Method, w.Install = p.Install.Method(), raw
	}
	return json.Marshal(w)
}

func (p *LivesOn) UnmarshalJSON(data []byte) error {
	var w wireLivesOn
	if err := json.Unmarshal(data, &w); err != nil {
		return err
	}
	install, err := installOf(w.Method, w.Install)
	if err != nil {
		return err
	}
	*p = LivesOn{Kinds: w.Kinds, Install: install}
	return nil
}

func installOf(m Method, raw json.RawMessage) (Install, error) {
	switch m {
	case "":
		return nil, nil
	case MethodBrew:
		return decoded[BrewInstall](raw)
	case MethodCask:
		return decoded[CaskInstall](raw)
	case MethodMas:
		return decoded[MasInstall](raw)
	case MethodNpm:
		return decoded[NpmInstall](raw)
	case MethodUv:
		return decoded[UvInstall](raw)
	case MethodBun:
		return decoded[BunInstall](raw)
	case MethodMise:
		return decoded[MiseInstall](raw)
	case MethodApt:
		return decoded[AptInstall](raw)
	case MethodGitHubRelease:
		return decoded[GitHubReleaseInstall](raw)
	case MethodScript:
		return decoded[ScriptInstall](raw)
	case MethodShell:
		return decoded[ShellInstall](raw)
	case MethodApp:
		return decoded[AppInstall](raw)
	case MethodSkill:
		return decoded[SkillInstall](raw)
	case MethodProvidedBy:
		return decoded[ProvidedByInstall](raw)
	}
	return nil, fmt.Errorf("catalog: no install method %q", m)
}

func decoded[I Install](raw json.RawMessage) (Install, error) {
	var i I
	err := json.Unmarshal(raw, &i)
	return i, err
}
