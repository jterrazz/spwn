package evolution

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"spwn.sh/packages/platform"

	"spwn.sh/packages/activity"
)

// Fork clones a Mind from source agent to target agent.
// If layers is nil, all layers are copied. Otherwise only the specified layers.
func Fork(sourceName, targetName string, layers []string) (*ForkResult, error) {
	sourceDir := filepath.Join(platform.AgentsDir(), sourceName)
	targetDir := filepath.Join(platform.AgentsDir(), targetName)
	files := platform.DataFiles()
	defer files.Close()

	if _, err := files.Stat(sourceDir); errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("source agent %q not found", sourceName)
	} else if err != nil {
		return nil, err
	}

	if _, err := files.Stat(targetDir); err == nil {
		return nil, fmt.Errorf("target agent %q already exists", targetName)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	allLayers := platform.MindLayers
	if len(layers) > 0 {
		allLayers = layers
	}

	result := &ForkResult{
		Source:       sourceName,
		Target:       targetName,
		LayersCopied: []string{},
	}

	for _, layer := range allLayers {
		src := filepath.Join(sourceDir, layer)
		dst := filepath.Join(targetDir, layer)
		if _, err := files.Stat(src); errors.Is(err, fs.ErrNotExist) {
			if err := files.MkdirAll(dst, 0755); err != nil {
				return nil, fmt.Errorf("creating %s layer: %w", layer, err)
			}
			continue
		}
		if err := copyDir(files, src, dst); err != nil {
			return nil, fmt.Errorf("copying layer %s: %w", layer, err)
		}
		result.LayersCopied = append(result.LayersCopied, layer)
	}

	sourceSoul := filepath.Join(sourceDir, platform.SoulFileName)
	if _, err := files.Stat(sourceSoul); err == nil {
		if err := copyFile(files, sourceSoul, filepath.Join(targetDir, platform.SoulFileName)); err != nil {
			return nil, fmt.Errorf("copying %s: %w", platform.SoulFileName, err)
		}
	}

	sourceManifest := filepath.Join(sourceDir, "agent.yaml")
	if _, err := files.Stat(sourceManifest); err == nil {
		data, err := files.ReadFile(sourceManifest)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", sourceManifest, err)
		}
		data = rewriteAgentName(data, targetName)
		if err := files.WriteFile(filepath.Join(targetDir, "agent.yaml"), data, 0644); err != nil {
			return nil, fmt.Errorf("writing the forked agent.yaml: %w", err)
		}
	}

	sourceEntry := filepath.Join(sourceDir, "AGENTS.md")
	if _, err := files.Stat(sourceEntry); err == nil {
		data, err := files.ReadFile(sourceEntry)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", sourceEntry, err)
		}
		if err := files.WriteFile(filepath.Join(targetDir, "AGENTS.md"), data, 0644); err != nil {
			return nil, fmt.Errorf("writing the forked AGENTS.md: %w", err)
		}
	}

	activity.Log(activity.Event{
		Type:    activity.TypeAgentForked,
		Actor:   "user",
		Verb:    "forked",
		Target:  targetName,
		Phrase:  activity.PhraseAgentForked(sourceName, targetName),
		AgentID: targetName,
		Metadata: map[string]any{
			"source": sourceName,
			"layers": result.LayersCopied,
		},
	})

	return result, nil
}

var agentNameLine = regexp.MustCompile(`(?m)^name:\s*.*$`)

func rewriteAgentName(data []byte, name string) []byte {
	replacement := []byte("name: " + name)
	if agentNameLine.Match(data) {
		return agentNameLine.ReplaceAll(data, replacement)
	}
	return append(append(replacement, '\n'), data...)
}

// ForkResult holds the outcome of a fork operation.
type ForkResult struct {
	Source       string
	Target       string
	LayersCopied []string
}

// value: src's tree copied to dst through files, folder by folder, so a
// link on either side is refused as files refuses it.
func copyDir(files platform.Files, src, dst string) error {
	if err := files.MkdirAll(dst, 0755); err != nil {
		return err
	}
	entries, err := files.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		from, to := filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())
		if e.IsDir() {
			err = copyDir(files, from, to)
		} else {
			err = copyFile(files, from, to)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func copyFile(files platform.Files, src, dst string) error {
	data, err := files.ReadFile(src)
	if err != nil {
		return err
	}
	if err := files.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	return files.WriteFile(dst, data, 0644)
}
