package mind

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"spwn.sh/packages/platform"
)

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

// ArchiveRefusedCode is the code of the refusal of an archive Import will
// not extract: an entry that names a path outside the agent's folder, a
// link, a device, a FIFO, or one past the size bounds.
const ArchiveRefusedCode = "agent-archive.refused"

const (
	// value: a Mind's file (SOUL.md, a playbook, a journal day) is kilobytes; 64 MiB is far past any.
	maxImportEntry = 64 << 20
	// value: four entries at the bound; a whole Mind is a few megabytes.
	maxImportTotal = 256 << 20
)

// Import extracts a tar.gz archive into an agent's Mind directory, through
// platform.DataFiles, so in a project no link the project holds is written
// through. Every entry is checked before anything is written: a regular
// file or a folder, at a relative path that stays in the agent's folder,
// within the size bounds; any other refuses the whole archive.
func Import(name string, archivePath string) error {
	entries, err := importEntries(archivePath, importBounds{entry: maxImportEntry, total: maxImportTotal})
	if err != nil {
		return err
	}
	dir := AgentDir(name)
	files := platform.DataFiles()
	defer files.Close()
	if err := files.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create agent dir: %w", err)
	}
	for _, e := range entries {
		target := filepath.Join(dir, e.name)
		if e.data == nil {
			if err := files.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := files.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := files.WriteFile(target, e.data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

type importEntry struct {
	name string
	data []byte
}

type importBounds struct {
	entry, total int64
}

func importEntries(archivePath string, bounds importBounds) ([]importEntry, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return nil, fmt.Errorf("open archive: %w", err)
	}
	defer f.Close()
	gr, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("gzip reader: %w", err)
	}
	defer gr.Close()

	var entries []importEntry
	total := int64(0)
	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return entries, nil
		}
		if err != nil {
			return nil, fmt.Errorf("read tar: %w", err)
		}
		refuse := func(why string) error {
			return fmt.Errorf("%s: entry %q %s", ArchiveRefusedCode, header.Name, why)
		}
		rel := filepath.Clean(filepath.FromSlash(header.Name))
		switch {
		case filepath.IsAbs(rel) || strings.HasPrefix(header.Name, "/"):
			return nil, refuse("is an absolute path")
		case slices.Contains(strings.Split(filepath.ToSlash(header.Name), "/"), ".."):
			return nil, refuse("climbs out of the agent's folder")
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if rel != "." {
				entries = append(entries, importEntry{name: rel})
			}
		case tar.TypeReg:
			if header.Size > bounds.entry {
				return nil, refuse(fmt.Sprintf("is larger than %d bytes", bounds.entry))
			}
			total += header.Size
			if total > bounds.total {
				return nil, refuse(fmt.Sprintf("takes the archive past %d bytes", bounds.total))
			}
			data, err := io.ReadAll(io.LimitReader(tr, header.Size))
			if err != nil {
				return nil, fmt.Errorf("read tar: %w", err)
			}
			if data == nil {
				data = []byte{}
			}
			entries = append(entries, importEntry{name: rel, data: data})
		case tar.TypeSymlink, tar.TypeLink:
			return nil, refuse("is a link: a Mind archive holds files and folders only")
		default:
			return nil, refuse("is a device, a FIFO or another special file: a Mind archive holds files and folders only")
		}
	}
}
