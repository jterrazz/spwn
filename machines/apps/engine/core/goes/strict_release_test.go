//go:build !world

package goes

import (
	"errors"
	"strings"
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

func TestABreakKeepsTheWordsOfAnErrorOrAStringAndTheTypeAloneOfAnyOtherValue(t *testing.T) {
	cases := []struct {
		name  string
		panic func()
		typ   string
		text  string
	}{
		{"a struct", func() { panic(struct{ Password string }{Password: "s3cret-pass"}) }, "struct { Password string }", ""},
		{"a map", func() { panic(map[string]string{"token": "s3cret-pass"}) }, "map[string]string", ""},
		{"a runtime error", func() {
			var m map[string]string
			m["token"] = "s3cret-pass"
		}, "runtime.plainError", "assignment to entry in nil map"},
		{"an error", func() { panic(errors.New("the store broke")) }, "*errors.errorString", "the store broke"},
		{"a string", func() { panic("the store broke") }, "string", "the store broke"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := &record{}
			s := newSet(r, nil)
			err := s.try("vault.read", func() error {
				c.panic()
				return nil
			})
			var b *Break
			if !errors.As(err, &b) || b.Type != c.typ || b.Text != c.text {
				t.Fatalf("break = %#v, want type %q and text %q", err, c.typ, c.text)
			}
			got := r.snapshot()
			if len(got.lines) != 1 || len(got.hooked) != 1 {
				t.Fatalf("log %q, hooked %d: want one line and one hook", got.lines, len(got.hooked))
			}
			for _, said := range []string{b.Error(), string(b.Stack), got.lines[0], got.hooked[0].Text} {
				if strings.Contains(said, "s3cret-pass") {
					t.Fatalf("the panic value's secret reached the break:\n%s", said)
				}
			}
		})
	}
}
