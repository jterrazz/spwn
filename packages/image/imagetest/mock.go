package imagetest

import (
	"context"
	"io"
	"time"

	"spwn.sh/packages/image/backend"
)

// MockBackend is a test double for the Docker backend.
type MockBackend struct {
	CreateFunc    func(ctx context.Context, cfg backend.ContainerConfig) (string, error)
	StartFunc     func(ctx context.Context, id string) error
	StopFunc      func(ctx context.Context, id string) error
	RemoveFunc    func(ctx context.Context, id string) error
	ExecFunc      func(ctx context.Context, id string, cfg backend.ExecConfig) (int, error)
	ExecOutFunc   func(ctx context.Context, id string, cmd []string) (string, error)
	EnsureFunc    func(ctx context.Context, tag, ver string, df []byte, extra map[string][]byte, w io.Writer) (bool, error)
	ImageExFunc   func(ctx context.Context, image string) (bool, error)
	ImageVerFunc  func(ctx context.Context, image, label string) (string, error)
}

func NewMockBackend() *MockBackend {
	return &MockBackend{
		CreateFunc:  func(ctx context.Context, cfg backend.ContainerConfig) (string, error) { return "mock-id", nil },
		StartFunc:   func(ctx context.Context, id string) error { return nil },
		StopFunc:    func(ctx context.Context, id string) error { return nil },
		RemoveFunc:  func(ctx context.Context, id string) error { return nil },
		ExecFunc:    func(ctx context.Context, id string, cfg backend.ExecConfig) (int, error) { return 0, nil },
		ExecOutFunc: func(ctx context.Context, id string, cmd []string) (string, error) { return "", nil },
		EnsureFunc: func(ctx context.Context, tag, ver string, df []byte, extra map[string][]byte, w io.Writer) (bool, error) {
			return true, nil
		},
		ImageExFunc: func(ctx context.Context, image string) (bool, error) { return false, nil },
		ImageVerFunc: func(ctx context.Context, image, label string) (string, error) { return "", nil },
	}
}

func (m *MockBackend) Create(ctx context.Context, cfg backend.ContainerConfig) (string, error) {
	return m.CreateFunc(ctx, cfg)
}
func (m *MockBackend) Start(ctx context.Context, id string) error { return m.StartFunc(ctx, id) }
func (m *MockBackend) Stop(ctx context.Context, id string) error  { return m.StopFunc(ctx, id) }
func (m *MockBackend) Remove(ctx context.Context, id string) error { return m.RemoveFunc(ctx, id) }
func (m *MockBackend) Exec(ctx context.Context, id string, cfg backend.ExecConfig) (int, error) {
	return m.ExecFunc(ctx, id, cfg)
}
func (m *MockBackend) ExecOutput(ctx context.Context, id string, cmd []string) (string, error) {
	return m.ExecOutFunc(ctx, id, cmd)
}
func (m *MockBackend) CopyTo(ctx context.Context, id string, path string, content []byte) error {
	return nil
}
func (m *MockBackend) CopyDirTo(ctx context.Context, id string, destDir string, hostSrcDir string) error {
	return nil
}
func (m *MockBackend) CopyDirFrom(ctx context.Context, id string, srcDir string, hostDestDir string) error {
	return nil
}
func (m *MockBackend) IsRunning(ctx context.Context, id string) (bool, error) { return true, nil }
func (m *MockBackend) ImageExists(ctx context.Context, image string) (bool, error) {
	return m.ImageExFunc(ctx, image)
}
func (m *MockBackend) EnsureImage(ctx context.Context, tag, ver string, df []byte, w io.Writer) (bool, error) {
	return m.EnsureFunc(ctx, tag, ver, df, nil, w)
}
func (m *MockBackend) EnsureImageWithContext(ctx context.Context, tag, ver string, df []byte, extra map[string][]byte, w io.Writer) (bool, error) {
	return m.EnsureFunc(ctx, tag, ver, df, extra, w)
}
func (m *MockBackend) ImageVersion(ctx context.Context, image, label string) (string, error) {
	return m.ImageVerFunc(ctx, image, label)
}
func (m *MockBackend) ExecDetached(ctx context.Context, id string, cfg backend.ExecConfig) error {
	return nil
}
func (m *MockBackend) Commit(ctx context.Context, id string, tag string) error { return nil }
func (m *MockBackend) ImageList(ctx context.Context, prefix string) ([]backend.ImageInfo, error) {
	return nil, nil
}
func (m *MockBackend) ImageRemove(ctx context.Context, tag string) error { return nil }
func (m *MockBackend) Inspect(ctx context.Context, nameOrID string) (*backend.ContainerInfo, error) {
	return &backend.ContainerInfo{ID: nameOrID, Running: true, StartedAt: time.Now()}, nil
}
