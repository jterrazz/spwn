package fake_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/remote/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/remote/contract/contracttest"
)

func TestTheFakeSecondMacKeepsTheEndpointContract(t *testing.T) {
	contracttest.Endpoint(t, func(_ *testing.T, far *call.Registry, profile machine.Profile, version string) contracttest.EndpointSubject {
		d := fake.New()
		d.Serve(profile.Ref.Name, fake.Mac{Registry: far, Profile: profile, Version: version})
		return contracttest.EndpointSubject{Dialer: d, Mac: profile.Ref.Name, Target: profile.Ref, Down: func() { d.Down(profile.Ref.Name) }}
	})
}
