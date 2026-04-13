package migrations

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
)

// TierToRoleState replaces "tier" with "role" in the raw state.json bytes.
var TierToRoleState = migration001()

func migration001() Migration {
	return Migration{
		Number:      1,
		Description: "rename tier to role in state.json",
		Apply: func(_ context.Context, baseDir string) error {
			path := filepath.Join(baseDir, "state.json")
			data, err := os.ReadFile(path)
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
			if !bytes.Contains(data, []byte(`"tier"`)) {
				return nil
			}
			data = bytes.ReplaceAll(data, []byte(`"tier"`), []byte(`"role"`))
			return os.WriteFile(path, data, 0644)
		},
	}
}
