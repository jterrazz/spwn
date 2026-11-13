package wire

import (
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func globals(home string) string {
	bins := make([]string, 0, len(catalog.GlobalBins))
	for _, b := range catalog.GlobalBins {
		bins = append(bins, path.Join(home, b))
	}
	return strings.Join(bins, ":")
}

func TestAGuestStartedWithTheSystemPathFindsWhatStationInstalledThere(t *testing.T) {
	env := withAppBins([]string{"HOME=/home/e2e", "PATH=/usr/bin:/bin"}, true)
	want := "PATH=/usr/bin:/bin:/home/e2e/.local/bin:" + path.Join("/home/e2e", catalog.AppData, catalog.AppShims) + ":" + globals("/home/e2e")
	if !slices.Contains(env, want) {
		t.Errorf("the guest's PATH: %v, want %s", env, want)
	}
	moved := withAppBins([]string{"HOME=/home/e2e", catalog.AppDataVar + "=/data/apps", "PATH=/home/e2e/.local/bin:/usr/bin"}, true)
	want = "PATH=/home/e2e/.local/bin:/usr/bin:/data/apps/" + catalog.AppShims + ":" + globals("/home/e2e")
	if !slices.Contains(moved, want) {
		t.Errorf("with the app manager's data moved and ~/.local/bin already first: %v, want %s", moved, want)
	}
	mac := withAppBins([]string{"HOME=/Users/o", "PATH=/usr/bin"}, false)
	if !slices.Contains(mac, "PATH=/usr/bin:"+globals("/Users/o")) {
		t.Errorf("on a Mac the shell block owns ~/.local/bin and the app manager: %v", mac)
	}
}
