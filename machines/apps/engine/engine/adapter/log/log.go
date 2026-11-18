package log

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/engine/contract"
)

type Log struct{ write func(subject, text string) }

var _ contract.Notifier = Log{}

func New(write func(subject, text string)) Log { return Log{write: write} }

func (l Log) Notify(_ context.Context, n engine.Notification) { l.write(n.Subject, n.Text) }
