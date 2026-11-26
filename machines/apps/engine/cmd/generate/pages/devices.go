package pages

import (
	"time"

	"github.com/jterrazz/studio-station/apps/engine/devices"
	pagedevices "github.com/jterrazz/studio-station/apps/engine/pages/devices"
)

// value: the mini's book: the laptop that may open everything, a phone
// that may open a pipe, and an entry whose seal no longer verifies.
func devicesListing() devices.Listing {
	added := keepAwakeNow.Add(-72 * time.Hour)
	return devices.Listing{
		Mac: "mac-mini",
		Key: "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBNwqKUs0pFt8GpXHBUNGGXQ1S1VLB0Jq7bI0dVh4gU0jPq1bOm7yzwF8uM2C3fZ0SZqO2qf3n0d6FxZgW4nJ8aI=",
		Devices: []devices.Row{
			{Name: "macbook-pro", Key: "ecdsa-sha2-nistp256 AAAAE2VjZHNh", Fingerprint: "SHA256:cT5hNh5Eo/0cBgMbx4JlmcPYDZoZXd37eJaUqyKQd2M", Channels: devices.AllChannels(), Added: added},
			{Name: "phone", Key: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5", Fingerprint: "SHA256:Qm3vLx9rT2kPz8YwHc1nB6dF4sJ0aE7gU5iO3lK2mN8", Channels: []devices.Channel{devices.ChannelPipe}, Added: added.Add(time.Hour)},
		},
		Broken: []devices.Row{
			{Name: "old-laptop", Key: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBERERERERERERERERERERERERERERERERERERERERER", Fingerprint: "SHA256:8b0p1yQ7cE2wR5tV9xZ3aS6dF4gH1jK0lM2nB5vC7xQ", Channels: []devices.Channel{devices.ChannelPipe}, Added: added.Add(-240 * time.Hour)},
		},
	}
}

func devicesPages() []Fixture {
	return states("devices",
		pending(pagedevices.Page(devicesListing(), thisMacContext())),
		pagedevices.Page(devices.Listing{Mac: "mac-mini", Key: devicesListing().Key, Devices: []devices.Row{}, Broken: []devices.Row{}}, thisMacContext()),
		failure("devices", "devices", "mac-mini's device book cannot be read", "station devices --on mac-mini"),
		pagedevices.Page(devicesListing(), thisMacContext()),
	)
}
