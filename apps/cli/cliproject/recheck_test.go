package cliproject

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestAnInsidePathSwappedToALinkOutIsRefusedAtTheRecheck(t *testing.T) {
	a, _, outside := projects(t)
	kn := filepath.Join(a, "kn")
	if err := os.Mkdir(kn, 0o755); err != nil {
		t.Fatal(err)
	}
	resolved, leaves := Resolve(a, "kn")
	if leaves {
		t.Fatalf("kn resolved outside: %s", resolved)
	}
	declared := []Declared{{Text: "kn", Resolved: resolved}}
	if err := Recheck(a, declared); err != nil {
		t.Fatalf("an unchanged path was refused: %v", err)
	}

	if err := os.Remove(kn); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, kn); err != nil {
		t.Fatal(err)
	}

	err := Recheck(a, declared)
	var changed *ChangedError
	if !errors.As(err, &changed) || changed.Code() != ChangedCode {
		t.Fatalf("Recheck = %v, want %s", err, ChangedCode)
	}
	want := ChangedCode + `: the host path "kn" of spwn.yaml resolved to ` + strconv.Quote(resolved) + " when the run began and to " + strconv.Quote(outside) +
		" now: a link of the project changed while starting, so this world is not started"
	if err.Error() != want {
		t.Fatalf("refusal = %q\nwant      %q", err, want)
	}
	noGrantsFile(t)
}

func TestALinkInsideRetargetedInsideIsRefusedToo(t *testing.T) {
	a, _, _ := projects(t)
	for _, d := range []string{"one", "two"} {
		if err := os.Mkdir(filepath.Join(a, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(a, "kn")
	if err := os.Symlink("one", link); err != nil {
		t.Fatal(err)
	}
	resolved, _ := Resolve(a, "kn")
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("two", link); err != nil {
		t.Fatal(err)
	}
	var changed *ChangedError
	if err := Recheck(a, []Declared{{Text: "kn", Resolved: resolved}}); !errors.As(err, &changed) {
		t.Fatalf("Recheck = %v, want a ChangedError", err)
	}
}
