package pages

import (
	"time"

	"github.com/jterrazz/studio-station/src/awake"
	model "github.com/jterrazz/studio-station/src/machines"
)

// value: the Awake stories: This Mac kept awake until 18:00 on battery,
// let sleep, kept 4 hr, or until linux-1 is done; mac-mini never sleeps,
// the air asleep.
func awakeWorlds() []World {
	awakePage := []string{"awake"}
	return []World{
		{Name: "machines.keep", Edit: keeping(untilSix), Pages: awakePage},
		{Name: "machines.keep.off", Edit: keeping(nil), Pages: awakePage},
		{Name: "machines.keep.4h", Edit: keeping(fourHours), Pages: awakePage},
		{Name: "machines.keep.run", Edit: keeping(untilLinux1), Pages: awakePage},
	}
}

func keeping(session func() *awake.Session) func(*Scene) {
	return macs().then(func(f *Scene) {
		f.Awake.Pauses = model.Pausing(f.All.Macs[0].VMs)
		f.Awake.Session = nil
		if session != nil {
			f.Awake.Session = session()
		}
	})
}

func untilSix() *awake.Session {
	return &awake.Session{Battery: true, StartedAt: storyNow.Add(-time.Hour), Until: time.Date(2026, 11, 7, 18, 0, 0, 0, time.UTC), By: "the menu bar"}
}

func fourHours() *awake.Session {
	return &awake.Session{Battery: true, StartedAt: storyNow, Until: storyNow.Add(4 * time.Hour), For: 4 * time.Hour, By: "the menu bar"}
}

func untilLinux1() *awake.Session {
	return &awake.Session{Battery: true, StartedAt: storyNow, Until: storyNow.Add(awake.Longest), Run: "linux-1", By: "the menu bar"}
}
