package contracttest

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
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

// value: a source that follows a branch, built by the adapter under test;
// the gesture that moves its branch forward to a new commit, and the one
// that rewrites it to a commit that does not follow the clone's, each
// answering the commit the branch is at then.

type Branch func(t *testing.T) (src contract.Source, forward, rewrite func() string)

func Follows(t *testing.T, branch Branch) {
	at := func(t *testing.T, src contract.Source) string {
		t.Helper()
		tree, err := src.Read(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if tree.Source == nil || tree.Source.Commit == "" {
			t.Fatalf("a followed source's tree names no commit: %+v", tree)
		}
		return tree.Source.Commit
	}
	follows := func(t *testing.T, src contract.Source) contract.Follows {
		t.Helper()
		f, ok := src.(contract.Follows)
		if !ok {
			t.Fatalf("%T follows no branch: it has no Pull", src)
		}
		return f
	}
	t.Run("is read where it stands until it is pulled, then at the branch's newest commit", func(t *testing.T) {
		src, forward, _ := branch(t)
		first := at(t, src)
		moved := forward()
		if now := at(t, src); now != first {
			t.Errorf("a read moved the source from %s to %s", first, now)
		}
		pulled, err := follows(t, src).Pull(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if now := at(t, src); now != moved || pulled != (contract.Pulled{Was: first, Now: moved}) {
			t.Errorf("pulled %+v and read at %s, want from %s to %s", pulled, now, first, moved)
		}
		if again, err := follows(t, src).Pull(context.Background()); err != nil || again.Moved() {
			t.Errorf("a second pull answered %+v, %v; want unmoved", again, err)
		}
		if err := follows(t, src).Back(context.Background(), pulled.Was); err != nil {
			t.Fatal(err)
		}
		if now := at(t, src); now != first {
			t.Errorf("put back at %s, want %s", now, first)
		}
	})
	t.Run("a newest commit that does not follow the clone's is refused, and the clone stays", func(t *testing.T) {
		src, _, rewrite := branch(t)
		first := at(t, src)
		newest := rewrite()
		_, err := follows(t, src).Pull(context.Background())
		var refused *contract.NotForwardError
		if !errors.As(err, &refused) || refused.At != first || refused.Newest != newest {
			t.Errorf("pulled a rewritten branch with %v; want refused from %s to %s", err, first, newest)
		}
		if now := at(t, src); now != first {
			t.Errorf("a refused pull moved the clone to %s", now)
		}
	})
}
