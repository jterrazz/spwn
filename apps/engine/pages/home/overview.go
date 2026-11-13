package home

import (
	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/apps"
	"github.com/jterrazz/studio-station/apps/engine/disk"
	"github.com/jterrazz/studio-station/apps/engine/overview"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

type (
	Overview  = overview.Snapshot[apps.Report, settings.Report, disk.MisplacedReport]
	Workspace = overview.Workspace[disk.MisplacedReport]
	Reading   = activity.Reading
)
