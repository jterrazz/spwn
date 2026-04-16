package migrations

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// OrgYAMLFieldRename renames legacy field names in org.yaml:
//   max-universes → max-worlds
//   max-citizens-per-universe → max-workers-per-world
var OrgYAMLFieldRename = migration006()

func migration006() Migration {
	return Migration{
		Number:      6,
		Description: "rename legacy org.yaml governance field names",
		Apply: func(_ context.Context, baseDir string) error {
			path := filepath.Join(baseDir, "org.yaml")
			data, err := os.ReadFile(path)
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}

			content := string(data)
			original := content

			content = strings.ReplaceAll(content, "max-universes:", "max-worlds:")
			content = strings.ReplaceAll(content, "max-citizens-per-universe:", "max-workers-per-world:")

			if content == original {
				return nil // already migrated or fields not present
			}

			return os.WriteFile(path, []byte(content), 0644)
		},
	}
}
