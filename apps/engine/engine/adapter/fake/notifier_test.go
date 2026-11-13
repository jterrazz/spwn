package fake_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/engine/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/engine/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

func TestNotifier(t *testing.T) {
	contracttest.Notifier(t, func(*testing.T) contracttest.NotifierSubject {
		w := &world.World{}
		return contracttest.NotifierSubject{Notifier: fake.NewNotifier(w), Delivered: func() []string {
			out := make([]string, 0, len(w.Told))
			for _, n := range w.Told {
				out = append(out, n.Subject+": "+n.Text)
			}
			return out
		}}
	})
}
