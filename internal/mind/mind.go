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

// AgentInfo describes an agent's Mind directory.
type AgentInfo struct {
	Name   string
	Path   string
	Layers map[string][]string // layer name → file names
}

// BaseDir returns the agents directory path (~/.universe/agents/).
func BaseDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, config.UniverseBaseDir, config.AgentsSubDir), nil
}

// AgentDir returns the path to a specific agent's Mind directory.
func AgentDir(name string) (string, error) {
	base, err := BaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, name), nil
}

// Init scaffolds a new Mind directory with all 6 layers.
func Init(name string) (string, error) {
	dir, err := AgentDir(name)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(dir); err == nil {
		return "", fmt.Errorf("agent %q already exists at %s", name, dir)
	}

	for _, layer := range config.MindLayers {
		layerDir := filepath.Join(dir, layer)
		if err := os.MkdirAll(layerDir, 0o755); err != nil {
			return "", fmt.Errorf("creating %s layer: %w", layer, err)
		}
	}

	// Create default persona.
	defaultPersona := filepath.Join(dir, "personas", "default.md")
	content := fmt.Sprintf("# %s\n\nYou are %s, a helpful AI assistant.\n", name, name)
	if err := os.WriteFile(defaultPersona, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("writing default persona: %w", err)
	}

	return dir, nil
}

// Validate checks that a Mind directory has the required structure.
func Validate(name string) error {
	dir, err := AgentDir(name)
	if err != nil {
		return err
	}

	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("agent %q not found. Run 'universe agent init %s' to create one", name, name)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}

	// Personas layer is required.
	personasDir := filepath.Join(dir, "personas")
	if _, err := os.Stat(personasDir); err != nil {
		return fmt.Errorf("Mind directory for %q is missing the personas/ layer. Run 'universe agent inspect %s' to see the current structure", name, name)
	}

	return nil
}

// List returns all agents on this Substrate.
func List() ([]AgentInfo, error) {
	base, err := BaseDir()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(base); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(base)
	if err != nil {
		return nil, fmt.Errorf("reading agents directory: %w", err)
	}

	var agents []AgentInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		info, err := Inspect(entry.Name())
		if err != nil {
			continue
		}
		agents = append(agents, info)
	}

	return agents, nil
}

// Export creates a tar.gz archive of the Mind directory, excluding specified layers.
func Export(name string, outputDir string, excludeLayers []string) (string, error) {
	dir, err := AgentDir(name)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return "", fmt.Errorf("agent %q not found", name)
	}

	excludeSet := make(map[string]bool)
	for _, l := range excludeLayers {
		excludeSet[l] = true
	}

	archivePath := filepath.Join(outputDir, name+".mind.tar.gz")
	f, err := os.Create(archivePath)
	if err != nil {
		return "", fmt.Errorf("creating archive: %w", err)
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

		// Skip excluded layers.
		topDir := strings.SplitN(relPath, string(os.PathSeparator), 2)[0]
		if excludeSet[topDir] {
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

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			io.Copy(tw, file)
		}

		return nil
	})
	if err != nil {
		return "", fmt.Errorf("creating archive: %w", err)
	}

	return archivePath, nil
}

// Import extracts a tar.gz Mind archive into the agent's directory.
func Import(name string, archivePath string) error {
	dir, err := AgentDir(name)
	if err != nil {
		return err
	}

	// Create agent directory if it doesn't exist.
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating agent directory: %w", err)
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("opening archive: %w", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("reading gzip: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar: %w", err)
		}

		// Prevent path traversal.
		target := filepath.Join(dir, header.Name)
		if !strings.HasPrefix(target, filepath.Clean(dir)+string(os.PathSeparator)) && target != filepath.Clean(dir) {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, 0o755)
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(target), 0o755)
			outFile, err := os.Create(target)
			if err != nil {
				return fmt.Errorf("creating file %s: %w", target, err)
			}
			io.Copy(outFile, tr)
			outFile.Close()
		}
	}

	return nil
}

// Inspect returns detailed information about an agent's Mind.
func Inspect(name string) (AgentInfo, error) {
	dir, err := AgentDir(name)
	if err != nil {
		return AgentInfo{}, err
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return AgentInfo{}, fmt.Errorf("agent %q not found", name)
	}

	info := AgentInfo{
		Name:   name,
		Path:   dir,
		Layers: make(map[string][]string),
	}

	for _, layer := range config.MindLayers {
		layerDir := filepath.Join(dir, layer)
		entries, err := os.ReadDir(layerDir)
		if err != nil {
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
