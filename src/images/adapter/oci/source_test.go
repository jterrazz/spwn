package oci

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/src/images"
	"github.com/jterrazz/studio-station/src/images/contract"
	"github.com/jterrazz/studio-station/src/images/contract/contracttest"
)

const base = "https://registry.test"

// value: a registry as ghcr answers: an anonymous pull token, then the tags
// of a repository only to that token, latest beside the versions.
type registry struct{ tags []string }

func (r registry) Get(_ context.Context, url string, header map[string]string) (status int, body []byte, err error) {
	switch {
	case strings.HasPrefix(url, base+"/token?scope=repository:jterrazz/"):
		body, err = json.Marshal(map[string]string{"token": "pull"})
		return 200, body, err
	case header["Authorization"] != "Bearer pull":
		return 401, nil, nil
	case url == base+"/v2/jterrazz/station-linux/tags/list":
		body, err = json.Marshal(map[string]any{"name": "jterrazz/station-linux", "tags": append([]string{"latest"}, r.tags...)})
		return 200, body, err
	}
	return 404, []byte(`{"errors":[{"code":"NAME_UNKNOWN"}]}`), nil
}

func TestTheRegistryKeepsTheContract(t *testing.T) {
	contracttest.ImageSource(t, func(_ *testing.T, versions ...string) contract.ImageSource {
		return New(registry{tags: versions}, base, "jterrazz", contracttest.Hypervisor)
	})
}

func TestAVersionIsPulledFromItsRepositoryTaggedWithIt(t *testing.T) {
	loc, err := New(registry{}, base, "jterrazz", "runner").Locate(t.Context(), images.Ref{Name: "linux", Version: "2026.11.01"}, "runner")
	if err != nil || loc.Ref != "registry.test/jterrazz/station-linux:2026.11.01" {
		t.Fatalf("Locate = %+v, %v; want registry.test/jterrazz/station-linux:2026.11.01", loc, err)
	}
}

// value: ghcr answers 403 to the pull token of a package nobody released (or
// that is private) and 404 to the tags of one that is absent; both mean the
// image is not released, whatever the status.
func TestAnImageTheRegistryRefusesOrDoesNotKnowIsNotReleased(t *testing.T) {
	for status := range map[int]bool{403: true, 404: true} {
		source := New(refusing{status: status}, base, "jterrazz", "runner")
		if got, err := source.Latest(t.Context(), "linux"); !errors.Is(err, contract.ErrNotReleased) {
			t.Errorf("a %d token: Latest = %+v, %v; want ErrNotReleased", status, got, err)
		}
	}
}

type refusing struct{ status int }

func (r refusing) Get(context.Context, string, map[string]string) (int, []byte, error) {
	return r.status, nil, nil
}
