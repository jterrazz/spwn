package architect

import (
	"fmt"
	"os"
	"path/filepath"

	"spwn.sh/packages/dependency/refs"
)

// collectLocalSkills walks every `skill:<name>` entry in the agent
// manifest's deps, reads spwn/skills/<name>.md from the project root,
// and keys each under its canonical container path
// /world/skills/<name>/SKILL.md so the imagebuilder merges it with the
// tool-shipped skills tree.
//
// Missing files are skipped silently (spwn check is the authoring-side
// gate; spawn is best-effort). Missing project root (legacy global
// mode) returns nil — there's no host tree to read from.
//
// The returned map is keyed by absolute container path because that's
// the contract BuildRequest.ExtraSkills documents — it's stable across
// renames of the build-context prefix used internally by imagebuilder.
func collectLocalSkills(projectRoot string, deps []string) map[string][]byte {
	if projectRoot == "" || len(deps) == 0 {
		return nil
	}
	out := make(map[string][]byte)
	for _, raw := range deps {
		ref := refs.ParseRef(raw)
		if ref.Kind != refs.KindLocalSkill || ref.Name == "" {
			continue
		}
		srcPath := filepath.Join(projectRoot, "spwn", "skills", ref.Name+".md")
		body, err := os.ReadFile(srcPath)
		if err != nil {
			continue
		}
		dstPath := fmt.Sprintf("/world/skills/%s/SKILL.md", ref.Name)
		out[dstPath] = body
	}
	return out
}
