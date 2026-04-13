package evolution

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"spwn.sh/packages/foundation"
	"spwn.sh/packages/foundation/activity"
)

// Fork clones a Mind from source agent to target agent.
// If layers is nil, all layers are copied. Otherwise only the specified layers.
func Fork(sourceName, targetName string, layers []string) (*ForkResult, error) {
	sourceDir := filepath.Join(foundation.AgentsDir(), sourceName)
	targetDir := filepath.Join(foundation.AgentsDir(), targetName)

	// Verify source exists
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("source agent %q not found", sourceName)
	}

	// Verify target doesn't exist
	if _, err := os.Stat(targetDir); err == nil {
		return nil, fmt.Errorf("target agent %q already exists", targetName)
	}

	// Determine which layers to copy
	allLayers := foundation.MindLayers
	if len(layers) > 0 {
		allLayers = layers
	}

	result := &ForkResult{
		Source:       sourceName,
		Target:       targetName,
		LayersCopied: []string{},
	}

	// Copy mind layers (includes identity)
	for _, layer := range allLayers {
		src := filepath.Join(sourceDir, layer)
		dst := filepath.Join(targetDir, layer)
		if _, err := os.Stat(src); os.IsNotExist(err) {
			os.MkdirAll(dst, 0755) // create empty layer
			continue
		}
		if err := copyDir(src, dst); err != nil {
			return nil, fmt.Errorf("copying layer %s: %w", layer, err)
		}
		result.LayersCopied = append(result.LayersCopied, layer)
	}

	// Copy agent.yaml if it exists
	sourceManifest := filepath.Join(sourceDir, "agent.yaml")
	if _, err := os.Stat(sourceManifest); err == nil {
		data, _ := os.ReadFile(sourceManifest)
		os.WriteFile(filepath.Join(targetDir, "agent.yaml"), data, 0644)
	}

	// Emit activity event
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

// ForkResult holds the outcome of a fork operation.
type ForkResult struct {
	Source       string
	Target       string
	LayersCopied []string
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	os.MkdirAll(filepath.Dir(dst), 0755)
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
