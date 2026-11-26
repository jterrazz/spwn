//go:build !world

package goes

import (
	"errors"
	"testing"
	"testing/synctest"
)

func TestTheReleaseBuildRecoversEveryPanicButAFatal(t *testing.T) {
	cases := []struct {
		value any
		typ   string
	}{
		{errPlain, "*errors.errorString"},
		{"index out of range", "string"},
		{nil, "*runtime.PanicNilError"},
	}
	for _, c := range cases {
		t.Run(c.typ, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r := &record{}
				s := newSet(r, nil)
				err := s.try("call.x", func() error { panic(c.value) })
				var b *Break
				if !errors.As(err, &b) || b.Type != c.typ {
					t.Fatalf("Try = %v, want a break of type %s", err, c.typ)
				}
				go s.once("job.x", func() { panic(c.value) }, false)
				synctest.Wait()
				if got := r.snapshot(); len(got.deaths) != 0 || s.breaks("job.x") != 1 {
					t.Fatalf("Go deaths %v, breaks %d", got.deaths, s.breaks("job.x"))
				}
			})
		})
	}
}
