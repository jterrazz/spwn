package platform

import (
	"io/fs"
	"os"
)

// Files is what reads and writes the files under AgentsDir: the agents'
// agent.yaml, journal and sessions.
type Files interface {
	ReadDir(p string) ([]fs.DirEntry, error)
	ReadFile(p string) ([]byte, error)
	Stat(p string) (fs.FileInfo, error)
	WriteFile(p string, data []byte, perm fs.FileMode) error
	MkdirAll(p string, perm fs.FileMode) error
	RemoveAll(p string) error
	Close() error
}

// DataFiles is the Files of AgentsDir: in a project, a Tree at the
// project's root, so no link the project carries is followed out of it,
// a FIFO is never waited on, and a path outside the project is refused;
// outside a project, the user's own home, read and written plainly.
func DataFiles() Files {
	if root := ProjectRoot(); root != "" {
		return Within(root)
	}
	return userFiles{}
}

type userFiles struct{}

func (userFiles) ReadDir(p string) ([]fs.DirEntry, error) { return os.ReadDir(p) }

func (userFiles) ReadFile(p string) ([]byte, error) { return os.ReadFile(p) }

func (userFiles) Stat(p string) (fs.FileInfo, error) { return os.Stat(p) }

func (userFiles) WriteFile(p string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(p, data, perm)
}

func (userFiles) MkdirAll(p string, perm fs.FileMode) error { return os.MkdirAll(p, perm) }

func (userFiles) RemoveAll(p string) error { return os.RemoveAll(p) }

func (userFiles) Close() error { return nil }
