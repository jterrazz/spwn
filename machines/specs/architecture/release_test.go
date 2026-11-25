package architecture_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

const srcModule = module + "/"

// value: the packages the binary links under the given build tags.
func linked(t *testing.T, tags string) []string {
	t.Helper()
	return linkedFor(t, tags, "")
}

// value: the packages the binary built for that system links; empty is
// the system the test runs on.
func linkedFor(t *testing.T, tags, goos string) []string {
	t.Helper()
	args := []string{"list", "-deps"}
	if tags != "" {
		args = append(args, "-tags", tags)
	}
	cmd := exec.Command("go", append(args, "./apps/engine/cmd/station")...)
	cmd.Dir = repoRoot(t)
	if goos != "" {
		cmd.Env = append(os.Environ(), "GOOS="+goos, "GOARCH=arm64")
	}
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	return strings.Fields(string(out))
}

func spec(pkg string) bool {
	rel, ok := strings.CutPrefix(pkg, srcModule)
	return ok && (rel == "world" || strings.HasSuffix(rel, "/adapter/fake"))
}

// invariant: rule spec-build-never-ships (L17): the release binary links no
// world and no fake, so it has no world command; only the spec build
// (`-tags world`) does, and no release recipe of the Makefile asks for it.
func TestTheReleaseBinaryLinksNoWorld(t *testing.T) {
	if got := slices.DeleteFunc(linked(t, ""), func(p string) bool { return !spec(p) }); len(got) > 0 {
		t.Errorf("the release binary links %v: only the spec build may", got)
	}
	makefile, err := os.ReadFile(filepath.Join(repoRoot(t), "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	recipe := regexp.MustCompile(`(?m)^(build|build-linux|provision):[^\n]*\n((?:\t[^\n]*\n)*)`)
	for _, m := range recipe.FindAllStringSubmatch(string(makefile), -1) {
		if strings.Contains(m[2], "-tags") {
			t.Errorf("the Makefile's %s recipe builds with tags: the binary it makes may ship", m[1])
		}
	}
}

// value: whether a package is a Docker client's.
func docker(pkg string) bool {
	return strings.HasPrefix(pkg, "github.com/moby/") || strings.HasPrefix(pkg, "github.com/docker/")
}

// invariant: ADR-088: the binary every machine runs links no Docker client,
// on a Mac and in a VM alike; Docker, if it survives, is driven through its
// command line, as tart is.
func TestTheBinaryLinksNoDockerClient(t *testing.T) {
	for _, goos := range []string{"darwin", "linux"} {
		if got := slices.DeleteFunc(linkedFor(t, "", goos), func(p string) bool { return !docker(p) }); len(got) > 0 {
			t.Errorf("the %s binary links %v: a Docker client is driven through its command line", goos, got)
		}
	}
	if !docker("github.com/moby/moby/client") || !docker("github.com/docker/go-connections/nat") || docker("github.com/jterrazz/studio-station/apps/engine/disk/adapter/docker") {
		t.Error("the Docker rule does not tell a Docker client's package from the others")
	}
}

// invariant: ARCHITECTURE 6.3: the spec build wires every family's fake
// instead of its real adapters, so a fake no build links is a family whose
// real tools a use case would run.
func TestTheSpecBuildWiresEveryFamilysFake(t *testing.T) {
	got := linked(t, "world")
	if !slices.Contains(got, srcModule+"world") {
		t.Fatal("the spec build links no world: the release rule proves nothing")
	}
	cmd := exec.Command("go", "list", "./apps/engine/...")
	cmd.Dir = repoRoot(t)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	for _, pkg := range strings.Fields(string(out)) {
		if strings.HasSuffix(pkg, "/adapter/fake") && !slices.Contains(got, pkg) {
			t.Errorf("the spec build does not wire %s: a world of it runs the real adapters", strings.TrimPrefix(pkg, srcModule))
		}
	}
}
