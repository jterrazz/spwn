package words

import (
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/devices"
)

const devicesRead = "station devices"

func devicesWords() map[fail.Code]Wording {
	return map[fail.Code]Wording{
		devices.BadName: func(e *fail.Error) (string, string) {
			return "\"" + e.Subject + "\" is no device name: 1 to 40 lowercase letters, digits and -", "name it as the device names itself: run station devices there"
		},
		devices.BadKey: func(*fail.Error) (string, string) {
			return "that is no device key: a key is one ecdsa-sha2-nistp256 or ssh-ed25519 line", "on the other device run: station devices, then paste its key"
		},
		devices.BadChannels: func(e *fail.Error) (string, string) {
			return "\"" + e.Subject + "\" names no channels: each is spwn-api, pipe or herdr, once", "leave --channels out to allow all three"
		},
		devices.Duplicate: func(e *fail.Error) (string, string) {
			return "this Mac lets " + e.Detail["device"] + " reach it already, by that name or that key", devicesRead
		},
		devices.OwnKey: func(*fail.Error) (string, string) {
			return "that is this Mac's own device key: its VMs trust it already", "paste the key the other device's station devices shows"
		},
	}
}
