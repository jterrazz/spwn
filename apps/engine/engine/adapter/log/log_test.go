package log_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/engine/adapter/log"
	"github.com/jterrazz/studio-station/apps/engine/engine/contract/contracttest"
)

func TestNotifier(t *testing.T) {
	contracttest.Notifier(t, func(*testing.T) contracttest.NotifierSubject {
		var lines []string
		n := log.New(func(subject, text string) { lines = append(lines, subject+": "+text) })
		return contracttest.NotifierSubject{Notifier: n, Delivered: func() []string { return lines }}
	})
}
