package fake

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machines/contract/contracttest"
)

func TestTheFakeMacsKeepTheUnlockerContract(t *testing.T) {
	contracttest.Unlocker(t, func(*testing.T) contracttest.UnlockerSubject {
		macs := NewMacs()
		return contracttest.UnlockerSubject{
			Unlocker: macs,
			AtPreBoot: func(mac, lan string) {
				macs.Set(mac, AtPreBoot)
				macs.OnLAN(mac, lan)
			},
			Offline:   func(mac string) { macs.Set(mac, Offline) },
			Answering: func(mac string) { macs.Set(mac, Answering) },
			Refusing:  macs.Refuse,
			Sessions:  macs.Sessions,
			Restarts:  macs.Restarts,
		}
	})
}
