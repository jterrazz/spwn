package contracttest

import (
	"context"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/engine/contract"
)

type NotifierSubject struct {
	Notifier contract.Notifier
	// value: what reached the owner so far, one entry per notification as the
	// owner reads it, in the order told.
	Delivered func() []string
}

func Notifier(t *testing.T, subject func(t *testing.T) NotifierSubject) {
	t.Run("each notification reaches the owner whole, in the order told", func(t *testing.T) {
		s := subject(t)
		told := []engine.Notification{
			{Kind: engine.NoticeGrantRemovalFailed, Subject: "api grant", Text: "removing the grant of GitHub failed: exit 1"},
			{Kind: engine.NoticeGrantRemovalFailed, Subject: "api grant", Text: "removing the grant of Cloudflare failed: exit 1"},
		}
		for _, n := range told {
			s.Notifier.Notify(context.Background(), n)
		}
		got := s.Delivered()
		if len(got) != len(told) {
			t.Fatalf("delivered %q; want %d notifications", got, len(told))
		}
		for i, n := range told {
			if !strings.Contains(got[i], n.Subject) || !strings.Contains(got[i], n.Text) {
				t.Errorf("delivered %q; want %q about %q", got[i], n.Text, n.Subject)
			}
		}
	})
}
