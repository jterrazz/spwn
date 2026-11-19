package macos

import (
	"context"
	"errors"
	"io/fs"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/settings"
)

const spotlightPane = "x-apple.systempreferences:com.apple.Spotlight-Settings.extension"

func (m *Mac) observeSpotlight(ctx context.Context, want []settings.KeyValue) settings.Observation {
	now := "indexed by Spotlight"
	if m.spotlightExcluded(ctx, m.home("~/Developer")) {
		now = "excluded from Spotlight"
	}
	return judged([]settings.KeyValue{{Key: "~/Developer", Value: now}}, want, "~/Developer in the Spotlight Privacy list", "")
}

func (m *Mac) applySpotlight(ctx context.Context) (settings.Change, error) {
	dev := m.home("~/Developer")
	if !m.exists(dev) {
		return settings.Change{}, errors.New("~/Developer does not exist")
	}
	if m.spotlightExcluded(ctx, dev) {
		return settings.Change{Steps: []settings.ChangeStep{done("~/Developer", "already excluded")}}, nil
	}
	// outside: a .metadata_never_index marker never worked inside a sub-folder, only at
	// volume root, so a stale one is removed.
	_ = m.files.Remove(dev + "/.metadata_never_index")
	if _, err := m.must(ctx, "open", spotlightPane); err != nil {
		return settings.Change{}, errors.New("opening System Settings failed: open Spotlight > Search Privacy by hand")
	}
	return settings.Change{Steps: []settings.ChangeStep{
		done("System Settings", "opened at Spotlight > Search Privacy"),
		next("next", "click +, pick ~/Developer, then run station settings to confirm"),
	}}, nil
}

// outside: on Tahoe (26+) the Privacy list lives in a root-only plist and
// `defaults read com.apple.Spotlight Exclusions` says "does not exist", so the check
// probes instead: mdfind finds a known file inside dir unless dir is excluded.
func (m *Mac) spotlightExcluded(ctx context.Context, dir string) bool {
	sample := m.spotlightSample(dir)
	if sample == "" {
		return false
	}
	slash := strings.LastIndex(sample, "/")
	out, ok := m.output(ctx, "mdfind", "-onlyin", sample[:slash], "kMDItemFSName == "+strconv.Quote(sample[slash+1:]))
	return ok && out == ""
}

// hazard: hidden files are skipped: Spotlight ignores them by default, so they would read as excluded.
func (m *Mac) spotlightSample(dir string) string {
	var sample string
	_ = m.files.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			//nolint:nilerr // an unreadable entry is skipped; the probe is the first file that IS readable
			return nil
		}
		if p != dir && strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() {
			sample = p
			return fs.SkipAll
		}
		return nil
	})
	return sample
}
