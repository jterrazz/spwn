package api

import (
	"context"
	"io"

	"spwn.sh/packages/universe/internal/backend"
)

// noContainersBackend implements backend.Backend with stubs that report
// "no containers exist". The api test suite uses this to keep
// state.Store reads deterministic — the real local Docker daemon may
// have spwn-labeled containers from interactive use, which would leak
// into list/status tests.
type noContainersBackend struct{}

func (noContainersBackend) ListContainersByLabel(_ context.Context, _, _ string) ([]backend.ContainerInfo, error) {
	return nil, nil
}

func (noContainersBackend) Create(context.Context, backend.ContainerConfig) (string, error) {
	return "", nil
}
func (noContainersBackend) Start(context.Context, string) error  { return nil }
func (noContainersBackend) Stop(context.Context, string) error   { return nil }
func (noContainersBackend) Remove(context.Context, string) error { return nil }
func (noContainersBackend) Exec(context.Context, string, backend.ExecConfig) (int, error) {
	return 0, nil
}
func (noContainersBackend) ExecOutput(context.Context, string, []string) (string, error) {
	return "", nil
}
func (noContainersBackend) CopyTo(context.Context, string, string, []byte) error {
	return nil
}
func (noContainersBackend) IsRunning(context.Context, string) (bool, error) { return false, nil }
func (noContainersBackend) ImageExists(context.Context, string) (bool, error) {
	return false, nil
}
func (noContainersBackend) EnsureImage(context.Context, string, string, []byte, io.Writer) error {
	return nil
}
func (noContainersBackend) EnsureImageWithContext(context.Context, string, string, []byte, map[string][]byte, io.Writer) error {
	return nil
}
func (noContainersBackend) ImageVersion(context.Context, string, string) (string, error) {
	return "", nil
}
func (noContainersBackend) ExecDetached(context.Context, string, backend.ExecConfig) error {
	return nil
}
func (noContainersBackend) Commit(context.Context, string, string) error { return nil }
func (noContainersBackend) ImageList(context.Context, string) ([]backend.ImageInfo, error) {
	return nil, nil
}
func (noContainersBackend) ImageRemove(context.Context, string) error { return nil }
func (noContainersBackend) Inspect(context.Context, string) (*backend.ContainerInfo, error) {
	return nil, nil
}
