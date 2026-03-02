package mind

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jterrazz/universe/internal/config"
)

// AgentInfo describes an agent's Mind structure.
type AgentInfo struct {
	Name   string              `json:"name"`
	Path   string              `json:"path"`
	Layers map[string][]string `json:"layers"`
}

// AgentDir returns the path to ~/.universe/agents/{name}/.
func AgentDir(name string) string {
	return filepath.Join(config.AgentsDir(), name)
}

// Init scaffolds a new Mind with all 6 layers.
func Init(name string) (string, error) {
	dir := AgentDir(name)
	if _, err := os.Stat(dir); err == nil {
		return "", fmt.Errorf("agent %q already exists", name)
	}

	for _, layer := range config.MindLayers {
		if err := os.MkdirAll(filepath.Join(dir, layer), 0755); err != nil {
			return "", fmt.Errorf("create %s: %w", layer, err)
		}
	}

	// Create default persona
	persona := `# Default Persona

## Role
You are a general-purpose assistant. You operate autonomously within the physics
of your universe.

## Operating Principles
- Read /universe/physics.md at startup to understand the world's constraints
- Read /universe/faculties.md to understand available technologies
- Stay within the Laws — they describe what's physically possible
`
	personaPath := filepath.Join(dir, "personas", "default.md")
	if err := os.WriteFile(personaPath, []byte(persona), 0644); err != nil {
		return "", fmt.Errorf("create persona: %w", err)
	}

	return dir, nil
}

// Validate checks that a Mind directory exists and has the personas layer.
func Validate(name string) error {
	dir := AgentDir(name)
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("agent %q not found", name)
	}
	if !info.IsDir() {
		return fmt.Errorf("agent %q is not a directory", name)
	}

	personas := filepath.Join(dir, "personas")
	if _, err := os.Stat(personas); err != nil {
		return fmt.Errorf("agent %q is missing the personas/ layer", name)
	}
	return nil
}

// List returns all agents in ~/.universe/agents/.
func List() ([]AgentInfo, error) {
	dir := config.AgentsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var agents []AgentInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		info, err := Inspect(e.Name())
		if err != nil {
			continue
		}
		agents = append(agents, *info)
	}
	return agents, nil
}

// Inspect returns detailed information about an agent's Mind.
func Inspect(name string) (*AgentInfo, error) {
	dir := AgentDir(name)
	if _, err := os.Stat(dir); err != nil {
		return nil, fmt.Errorf("agent %q not found", name)
	}

	info := &AgentInfo{
		Name:   name,
		Path:   dir,
		Layers: make(map[string][]string),
	}

	for _, layer := range config.MindLayers {
		layerDir := filepath.Join(dir, layer)
		entries, err := os.ReadDir(layerDir)
		if err != nil {
			info.Layers[layer] = nil
			continue
		}
		var files []string
		for _, e := range entries {
			if !e.IsDir() {
				files = append(files, e.Name())
			}
		}
		info.Layers[layer] = files
	}

	return info, nil
}

// LayerCount returns how many layers have at least one file.
func LayerCount(info *AgentInfo) int {
	count := 0
	for _, files := range info.Layers {
		if len(files) > 0 {
			count++
		}
	}
	return count
}

// Export creates a tar.gz archive of an agent's Mind directory.
// excludeLayers allows skipping specific layers (e.g., "journal", "sessions").
func Export(name string, outputPath string, excludeLayers []string) (string, error) {
	dir := AgentDir(name)
	if _, err := os.Stat(dir); err != nil {
		return "", fmt.Errorf("agent %q not found", name)
	}

	exclude := make(map[string]bool)
	for _, l := range excludeLayers {
		exclude[l] = true
	}

	archiveName := filepath.Join(outputPath, name+".tar.gz")
	f, err := os.Create(archiveName)
	if err != nil {
		return "", fmt.Errorf("create archive: %w", err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		// Skip excluded layers
		topDir := strings.SplitN(relPath, string(os.PathSeparator), 2)[0]
		if exclude[topDir] && relPath != "." {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(tw, file)
		return err
	})

	if err != nil {
		os.Remove(archiveName)
		return "", fmt.Errorf("archive: %w", err)
	}

	return archiveName, nil
}

// Import extracts a tar.gz archive into an agent's Mind directory.
func Import(name string, archivePath string) error {
	dir := AgentDir(name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create agent dir: %w", err)
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar: %w", err)
		}

		// Path traversal protection
		target := filepath.Join(dir, header.Name)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dir)) {
			return fmt.Errorf("invalid path in archive: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			out, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}

	return nil
}
