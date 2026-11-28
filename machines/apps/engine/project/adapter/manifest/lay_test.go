package manifest

import (
	"errors"
	"path"
	"path/filepath"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

func TestAFolderSwappedForALinkAfterItsMkdirIsNeverWrittenThrough(t *testing.T) {
	for name, swapped := range map[string]string{"the new folder": "site", "a folder under it": "site/spwn"} {
		t.Run(name, func(t *testing.T) {
			folder, elsewhere := t.TempDir(), t.TempDir()
			under := local.Beneath{}
			if err := under.Folders(folder, swapped, 0o755, local.Self()); err != nil {
				t.Fatal(err)
			}
			made, err := under.Open(folder, swapped)
			if err != nil {
				t.Fatal(err)
			}
			err = made.RemoveSelf()
			made.Close()
			if err != nil {
				t.Fatal(err)
			}
			up, err := under.Open(filepath.Dir(folder), path.Join(filepath.Base(folder), path.Dir(swapped)))
			if err != nil {
				t.Fatal(err)
			}
			err = up.Link(path.Base(swapped), elsewhere)
			up.Close()
			if err != nil {
				t.Fatal(err)
			}
			files := map[string][]byte{"spwn.yaml": []byte("name: site\n"), "agents/neo/agent.yaml": []byte("name: neo\n")}
			if err = (Projects{}).lay(folder, "site", files); !errors.Is(err, local.ErrLink) {
				t.Errorf("lay = %v; want the link at %s refused", err, swapped)
			}
			there, err := under.Open(filepath.Dir(elsewhere), filepath.Base(elsewhere))
			if err != nil {
				t.Fatal(err)
			}
			defer there.Close()
			if names, err := there.Names(); err != nil || len(names) != 0 {
				t.Errorf("the link's folder holds %v, %v; want nothing written through the link", names, err)
			}
		})
	}
}
