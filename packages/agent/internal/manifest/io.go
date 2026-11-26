package manifest

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"spwn.sh/packages/platform"
)

// FileName is the on-disk name for the agent manifest.
const FileName = "agent.yaml"

// Path returns the path to agent.yaml inside agentDir.
func Path(agentDir string) string {
	return filepath.Join(agentDir, FileName)
}

// Load reads agent.yaml from agentDir, through platform.DataFiles, so in
// a project a link out of it is refused, never read. A missing file
// returns a zero-value Manifest with ok=false so callers can distinguish
// "declared but empty" from "never authored".
func Load(agentDir string) (*Manifest, bool, error) {
	path := Path(agentDir)
	files := platform.DataFiles()
	defer files.Close()
	data, err := files.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &Manifest{}, false, nil
		}
		return nil, false, fmt.Errorf("read %s: %w", path, err)
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, false, fmt.Errorf("parse %s: %w", path, err)
	}
	return &m, true, nil
}

// Save writes m to agentDir/agent.yaml, through platform.DataFiles, so in
// a project a link out of it is refused, never written through. The
// caller is responsible for ensuring agentDir exists.
func Save(agentDir string, m *Manifest) error {
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	path := Path(agentDir)
	files := platform.DataFiles()
	defer files.Close()
	if err := files.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
