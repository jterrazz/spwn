package upgrade

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const versionFile = "version.json"

// SchemaVersion tracks the current migration state.
type SchemaVersion struct {
	Version   int                `json:"version"`
	UpdatedAt time.Time          `json:"updated_at"`
	Applied   []AppliedMigration `json:"applied"`
}

// AppliedMigration records a single applied migration.
type AppliedMigration struct {
	Number      int       `json:"number"`
	Description string    `json:"description"`
	AppliedAt   time.Time `json:"applied_at"`
}

// LoadVersion reads the schema version from baseDir/version.json.
// Returns a zero-value SchemaVersion if the file does not exist.
func LoadVersion(baseDir string) (*SchemaVersion, error) {
	path := filepath.Join(baseDir, versionFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &SchemaVersion{}, nil
		}
		return nil, err
	}
	var v SchemaVersion
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// SaveVersion writes the schema version to baseDir/version.json.
func SaveVersion(baseDir string, v *SchemaVersion) error {
	path := filepath.Join(baseDir, versionFile)
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}
