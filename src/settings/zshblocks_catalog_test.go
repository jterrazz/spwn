package settings

import (
	"testing"

	"github.com/jterrazz/studio-station/src/catalog"
)

func builtinItems(t *testing.T) []catalog.Placed {
	t.Helper()
	c, err := catalog.Builtin()
	if err != nil {
		t.Fatal(err)
	}
	return c.Items
}

func TestEveryShellBlockOfTheCatalogRendersOnEverySystemAndRoundTrips(t *testing.T) {
	// invariant: a block the catalog declares fills on any machine its
	// method is for, with the options' defaults, and ~/.zshrc gives it back
	// byte for byte once written and nothing once removed.
	systems := []catalog.ShellVars{
		{Home: "/Users/a", OS: catalog.OSMacOS, Arch: catalog.ArchARM64},
		{Home: "/Users/a", OS: catalog.OSMacOS, Arch: catalog.ArchAMD64},
		{Home: "/home/a", OS: catalog.OSUbuntu, Arch: catalog.ArchARM64},
	}
	declared := 0
	for _, p := range builtinItems(t) {
		for _, m := range p.Item.Methods {
			for _, b := range m.Shell {
				declared++
				for _, v := range systems {
					body, err := p.Item.RenderShell(b, v)
					if err != nil {
						t.Errorf("%s on %s: %v", b.ID, v.OS, err)
						continue
					}
					owner := "export EDITOR=vim\n\n# mine\n"
					written, _ := UpsertPackageBlock(owner, b.ID, body)
					if got, state := PackageBlock(written, b.ID); state != BlockHere || got != body {
						t.Errorf("%s reads back %q, %v; wrote %q", b.ID, got, state, body)
					}
					if back, _ := RemovePackageBlock(written, b.ID); back != owner {
						t.Errorf("%s: removing it leaves %q, not the owner's %q", b.ID, back, owner)
					}
				}
			}
		}
	}
	if declared < 2 {
		t.Errorf("the catalog declares %d shell blocks, at least docker-shortcuts and postgres are expected", declared)
	}
}
