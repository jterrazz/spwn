// Package blueprint manages the universe blueprint — the knowledge base
// maintained by the Architect as the single source of truth.
package blueprint

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DefaultFiles maps relative paths to their default content.
var DefaultFiles = map[string]string{
	"overview.md": `# Universe Blueprint

This is the knowledge base for your spwn universe.
The Architect maintains this as the single source of truth.

## Quick Links
- [Glossary](glossary.md)
- [Roadmap](roadmap.md)
- [Team](agents/team.md)
`,
	"glossary.md": `# Glossary

Key terms and concepts used across projects.

| Term | Definition |
|------|------------|
| World | An isolated Docker container where an agent works |
| Agent | A persistent AI citizen with identity and memory |
| Architect | The always-on daemon that manages worlds and agents |
| Blueprint | This knowledge base — the single source of truth |
| Mind | An agent's persistent memory (identity, skills, knowledge) |
`,
	"roadmap.md": `# Roadmap

## Current Focus
(The Architect will fill this in based on conversations)

## Upcoming

## Completed
`,
	"agents/team.md": `# Team

Agents and their roles in this universe.

| Agent | Role | Status |
|-------|------|--------|
`,
}

// DefaultDirs lists directories that should be created (even if empty).
var DefaultDirs = []string{
	"decisions",
	"projects",
}

// Init creates the blueprint directory at basePath and writes default files
// if they don't already exist. It never overwrites existing files.
func Init(basePath string) error {
	// Create the base directory
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return fmt.Errorf("create blueprint dir: %w", err)
	}

	// Create default subdirectories
	for _, dir := range DefaultDirs {
		dirPath := filepath.Join(basePath, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("create blueprint subdir %s: %w", dir, err)
		}
	}

	// Write default files (don't overwrite existing)
	for relPath, content := range DefaultFiles {
		absPath := filepath.Join(basePath, relPath)

		// Skip if file already exists
		if _, err := os.Stat(absPath); err == nil {
			continue
		}

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
			return fmt.Errorf("create parent dir for %s: %w", relPath, err)
		}

		if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("write default %s: %w", relPath, err)
		}
	}

	return nil
}

// FileInfo describes a file in the blueprint.
type FileInfo struct {
	Path     string    `json:"path"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
}

// ListFiles returns all files in the blueprint directory recursively.
func ListFiles(basePath string) ([]FileInfo, error) {
	var files []FileInfo

	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		// Skip hidden files like .gitkeep
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		relPath, err := filepath.Rel(basePath, path)
		if err != nil {
			return err
		}

		files = append(files, FileInfo{
			Path:     relPath,
			Size:     info.Size(),
			Modified: info.ModTime(),
		})
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk blueprint: %w", err)
	}

	return files, nil
}

// ReadFile reads a specific file from the blueprint.
func ReadFile(basePath, relPath string) (string, error) {
	// Prevent directory traversal
	if strings.Contains(relPath, "..") {
		return "", fmt.Errorf("invalid path: directory traversal not allowed")
	}

	absPath := filepath.Join(basePath, relPath)

	// Ensure the resolved path is still under basePath
	cleanPath := filepath.Clean(absPath)
	cleanBase := filepath.Clean(basePath)
	if !strings.HasPrefix(cleanPath, cleanBase) {
		return "", fmt.Errorf("path outside blueprint directory")
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", relPath, err)
	}

	return string(data), nil
}

// WriteFile writes content to a file in the blueprint.
func WriteFile(basePath, relPath, content string) error {
	// Prevent directory traversal
	if strings.Contains(relPath, "..") {
		return fmt.Errorf("invalid path: directory traversal not allowed")
	}

	absPath := filepath.Join(basePath, relPath)

	// Ensure the resolved path is still under basePath
	cleanPath := filepath.Clean(absPath)
	cleanBase := filepath.Clean(basePath)
	if !strings.HasPrefix(cleanPath, cleanBase) {
		return fmt.Errorf("path outside blueprint directory")
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}

	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write %s: %w", relPath, err)
	}

	return nil
}

// Search searches for a query string across all blueprint files.
// Returns a map of file path → matching lines.
func Search(basePath, query string) (map[string][]string, error) {
	results := make(map[string][]string)
	queryLower := strings.ToLower(query)

	files, err := ListFiles(basePath)
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		content, err := ReadFile(basePath, f.Path)
		if err != nil {
			continue
		}

		lines := strings.Split(content, "\n")
		for _, line := range lines {
			if strings.Contains(strings.ToLower(line), queryLower) {
				results[f.Path] = append(results[f.Path], line)
			}
		}
	}

	return results, nil
}
