package migrations

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// TierToRoleProfiles renames the "tier:" YAML key to "role:" in all agent profile files.
var TierToRoleProfiles = migration002()

func migration002() Migration {
	return Migration{
		Number:      2,
		Description: "rename tier to role in agent profiles",
		Apply: func(_ context.Context, baseDir string) error {
			pattern := filepath.Join(baseDir, "agents", "*", "profile.yaml")
			matches, err := filepath.Glob(pattern)
			if err != nil {
				return err
			}
			for _, path := range matches {
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				text := string(data)
				if !strings.Contains(text, "tier:") {
					continue
				}
				text = strings.ReplaceAll(text, "tier:", "role:")
				if err := os.WriteFile(path, []byte(text), 0644); err != nil {
					return err
				}
			}
			return nil
		},
	}
}
