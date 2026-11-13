package calls

import (
	"context"
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"
)

type recordingRouter struct {
	asked []machinescalls.CreateIn
}

func (r *recordingRouter) Invoke(_ context.Context, _ machine.Ref, _ string, in, out any, _ call.Progress) error {
	asked, ok := in.(machinescalls.CreateIn)
	created, okOut := out.(*machines.Created)
	if !ok || !okOut {
		return errors.New("recordingRouter: unexpected call shape")
	}
	r.asked = append(r.asked, asked)
	created.VM.Name = "docs-bot"
	return nil
}

func (r *recordingRouter) Watch(context.Context, machine.Ref, string, any, call.Progress) (<-chan call.Update[any], error) {
	return nil, nil
}

type silent struct{}

func (silent) Step(string, string) {}
func (silent) Done(string, error)  {}

func TestNewMachineWithoutImageAsksForTheDefaultImage(t *testing.T) {
	for _, c := range []struct{ name, image, want string }{
		{name: "no image asks the default image", want: "station-linux"},
		{name: "a named image is kept", image: "custom-image", want: "custom-image"},
	} {
		t.Run(c.name, func(t *testing.T) {
			router := &recordingRouter{}
			d := Deps{Calls: router}
			if _, err := d.create(t.Context(), StartIn{Name: "docs-bot", New: &New{Image: c.image}}, silent{}); err != nil {
				t.Fatal(err)
			}
			if len(router.asked) != 1 || router.asked[0].Request.Image != c.want {
				t.Errorf("asked images = %+v, want one for %q", router.asked, c.want)
			}
		})
	}
}
