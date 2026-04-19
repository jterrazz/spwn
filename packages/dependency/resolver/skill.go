package resolver

import (
	"fmt"
	"io/fs"
	"strings"

	"spwn.sh/packages/dependency/tool"
)

// CollectSkills aggregates skill files from all resolved tools into a map
// of destination paths → content for copying into the image.
// Skills are placed at /world/skills/{tool-name}/.
func CollectSkills(tools []tool.Tool) (map[string][]byte, error) {
	result := make(map[string][]byte)

	var skillTools []string

	for _, t := range tools {
		skillFS := t.Skills()
		if skillFS == nil {
			continue
		}

		toolName := strings.TrimPrefix(t.Name(), "@")
		skillTools = append(skillTools, toolName)

		err := fs.WalkDir(skillFS, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			content, err := fs.ReadFile(skillFS, path)
			if err != nil {
				return fmt.Errorf("read skill %s/%s: %w", t.Name(), path, err)
			}
			destPath := fmt.Sprintf("/world/skills/%s/%s", toolName, path)
			result[destPath] = content
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("collect skills for %s: %w", t.Name(), err)
		}
	}

	// Generate INDEX.md
	if len(skillTools) > 0 {
		result["/world/skills/INDEX.md"] = generateSkillIndex(skillTools)
	}

	return result, nil
}

func generateSkillIndex(toolNames []string) []byte {
	var sb strings.Builder
	sb.WriteString("# Installed Skills\n\n")
	for _, name := range toolNames {
		sb.WriteString(fmt.Sprintf("- [%s](./%s/SKILL.md)\n", name, name))
	}
	return []byte(sb.String())
}
