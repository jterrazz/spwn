package calls

import (
	"context"
	"time"
)

type Machine interface {
	// value: what the images service reads and keeps on the Mac it builds
	// on, handed by wire over that Mac's own files, network and commands.
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) error
	Exists(path string) bool
	ReadFile(path string) ([]byte, error)
	// value: an executable written at path, its folder made.
	Keep(path string, data []byte) error
	// value: the status and body a GET of url answers.
	Get(ctx context.Context, url string) (status int, body []byte, err error)
	// value: a new key pair for the image at path and path.pub, its folder
	// made; an error that says why when it cannot.
	MakeKey(ctx context.Context, path string) error
}
