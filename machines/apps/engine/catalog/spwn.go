package catalog

import (
	_ "embed"
	"path/filepath"
	"strings"
)

// value: the drive section of the machine's `spwn` skill: what spwn is, in
// its READMEs' words, and each command as it is used, written by `make
// generate` from the command tree
// (specs/architecture/drive_test.go).
// invariant: an embedded constant, the same bytes in every binary of one
// tree: no package, Machinefile or machine text reaches it, so the skill
// writes it as it is, and it holds no line that would open a rule or a
// second front matter (machinefile's skill_bound_test.go).

//go:embed spwn/drive.txt
var SpwnDrive string

// value: where the machine's `spwn` skill lands, one copy per agent skill
// folder the `skill` method installs into (SkillAgents), each a row of
// StationDisk.

var SpwnSkillFiles = []string{"~/.agents/skills/spwn/SKILL.md", "~/.claude/skills/spwn/SKILL.md"}

func SpwnSkillPaths(home string) []string {
	out := make([]string, len(SpwnSkillFiles))
	for i, p := range SpwnSkillFiles {
		out[i] = filepath.Join(home, strings.TrimPrefix(p, "~/"))
	}
	return out
}
