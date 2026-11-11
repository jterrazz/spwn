package contracttest

import (
	"context"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/src/machinefile/contract"
)

// value: what a source holds, by slash path under it, and a file outside
// the two folders a source is read for.
func held() map[string]string {
	return map[string]string{
		"catalog/apps/jq.yaml":           "item: station.catalog/v1\n",
		"machinefiles/laptop.yaml":       "machinefile: station.machinefile/v1\n",
		"machinefiles/laptop.local.yaml": "machinefile: station.machinefile/v1\n",
		"notes/readme.md":                "not read",
	}
}

// value: a source holding files, built by the adapter under test.

type Subject func(t *testing.T, files map[string]string) contract.Source

func Source(t *testing.T, subject Subject) {
	t.Run("reads its catalog and Machinefiles, in path order, and nothing else", func(t *testing.T) {
		held := held()
		tree, err := subject(t, held).Read(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		paths := make([]string, 0, len(tree.Files))
		for _, f := range tree.Files {
			paths = append(paths, f.Path)
			if string(f.Body) != held[f.Path] {
				t.Errorf("%s reads %q, holds %q", f.Path, f.Body, held[f.Path])
			}
		}
		want := []string{"catalog/apps/jq.yaml", "machinefiles/laptop.local.yaml", "machinefiles/laptop.yaml"}
		if !slices.Equal(paths, want) {
			t.Errorf("read %v, want %v", paths, want)
		}
		if tree.Root == "" || tree.Layer == "" {
			t.Errorf("a tree names its root and its layer: %+v", tree)
		}
	})
	t.Run("a source with neither folder reads no file", func(t *testing.T) {
		tree, err := subject(t, nil).Read(context.Background())
		if err != nil || len(tree.Files) != 0 {
			t.Errorf("read %v, %v; want no file", tree.Files, err)
		}
	})
}
