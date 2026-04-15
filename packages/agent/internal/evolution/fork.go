package evolution

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"spwn.sh/packages/base"
	"spwn.sh/packages/activity"
	"spwn.sh/packages/paths"
)

// Fork clones a Mind from source agent to target agent.
// If layers is nil, all layers are copied. Otherwise only the specified layers.
func Fork(sourceName, targetName string, layers []string) (*ForkResult, error) {
	sourceDir := filepath.Join(paths.AgentsDir(), sourceName)
	targetDir := filepath.Join(paths.AgentsDir(), targetName)

	// Verify source exists
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("source agent %q not found", sourceName)
	}

	// Verify target doesn't exist
	if _, err := os.Stat(targetDir); err == nil {
		return nil, fmt.Errorf("target agent %q already exists", targetName)
	}

	// Determine which layers to copy
	allLayers := base.MindLayers
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

	// Copy agent.yaml if it exists, rewriting the name: field to
	// match the target so `spwn check` doesn't warn about a name /
	// directory mismatch on the freshly forked agent.
	sourceManifest := filepath.Join(sourceDir, "agent.yaml")
	if _, err := os.Stat(sourceManifest); err == nil {
		data, _ := os.ReadFile(sourceManifest)
		data = rewriteAgentName(data, targetName)
		os.WriteFile(filepath.Join(targetDir, "agent.yaml"), data, 0644)
	}

	// Copy AGENTS.md if it exists. Without this, `spwn check` reports
	// the forked agent as missing AGENTS.md since ruleAgentStructure
	// requires it at the agent root.
	sourceEntry := filepath.Join(sourceDir, "AGENTS.md")
	if _, err := os.Stat(sourceEntry); err == nil {
		data, _ := os.ReadFile(sourceEntry)
		os.WriteFile(filepath.Join(targetDir, "AGENTS.md"), data, 0644)
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

// agentNameLine matches a top-level `name: <value>` entry in agent.yaml
// so we can rewrite it to the forked target name without spinning up
// a full YAML round-trip.
var agentNameLine = regexp.MustCompile(`(?m)^name:\s*.*$`)

func rewriteAgentName(data []byte, name string) []byte {
	replacement := []byte("name: " + name)
	if agentNameLine.Match(data) {
		return agentNameLine.ReplaceAll(data, replacement)
	}
	// No existing name: line — prepend one so the forked manifest is
	// still self-describing.
	return append(append(replacement, '\n'), data...)
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
