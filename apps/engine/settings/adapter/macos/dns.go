package macos

import (
	"context"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

const (
	profilesPane = "x-apple.systempreferences:com.apple.Profiles-Settings.extension"
	quad9        = "Quad9 DoH"
)

func (m *Mac) observeDNS(ctx context.Context, want []settings.KeyValue) settings.Observation {
	now := "the network's"
	if m.dnsInstalled(ctx) {
		now = quad9
	}
	return judged([]settings.KeyValue{{Key: "encrypted DNS", Value: now}}, want, quad9, "")
}

func (m *Mac) applyDNS(ctx context.Context) (settings.Change, error) {
	if m.dnsInstalled(ctx) {
		if _, err := m.must(ctx, "open", profilesPane); err != nil {
			return settings.Change{}, err
		}
		return settings.Change{Steps: []settings.ChangeStep{done("profile", "already installed: Profiles settings opened")}}, nil
	}
	profile := m.o.DNSProfile
	if err := m.write(profile, []byte(catalog.DNSProfile), 0o600); err != nil {
		return settings.Change{}, err
	}
	if _, err := m.must(ctx, "open", profile); err != nil {
		return settings.Change{}, err
	}
	return settings.Change{Steps: []settings.ChangeStep{
		done("profile", "written to "+profile),
		next("next", "approve the profile in System Settings > Privacy & Security > Profiles"),
	}}, nil
}

func (m *Mac) dnsInstalled(ctx context.Context) bool {
	out, ok := m.output(ctx, "profiles", "-C", "-v")
	return ok && strings.Contains(out, catalog.DNSProfileID)
}
