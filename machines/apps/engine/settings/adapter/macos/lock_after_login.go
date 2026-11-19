package macos

import (
	"context"
	"errors"
	"io/fs"
	"path"
	"strconv"
	"strings"

	backgroundcontract "github.com/jterrazz/studio-station/apps/engine/background/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

const (
	lockLabel  = "ai.jterrazz.lock-after-login"
	lockScript = catalog.LockAfterLoginScript
	lockPlist  = agentHome + "/Library/LaunchAgents/" + lockLabel + ".plist"
	lockLogDir = agentHome + "/.openclaw/logs"
)

func (m *Mac) observeLock(ctx context.Context, want []settings.KeyValue) settings.Observation {
	now := m.lockValues(ctx)
	o := judged(now, want, "agent loaded", "agent installed, not loaded")
	if now[0].Value != "present" {
		o.State, o.Detail = catalog.StateAbsent, ""
	}
	return o
}

func (m *Mac) applyLock(ctx context.Context) (settings.Change, error) {
	if !m.exists(lockScript) {
		return settings.Change{}, errors.New("missing " + lockScript + ", expected the openclaw lock-after-login script")
	}
	change := settings.Change{Before: m.lockValues(ctx)}
	// invariant: launchd executes this script: the x bit is the point.
	if err := m.files.Chmod(lockScript, 0o755); err != nil {
		return change, err
	}
	if err := m.files.MkdirAll(lockLogDir, 0o750); err != nil {
		return change, err
	}
	if err := m.write(lockPlist, []byte(lockPlistText()), 0o600); err != nil {
		return change, err
	}
	uid, gid, known := m.agentIDs(ctx)
	if known {
		_ = m.files.Chown(lockPlist, uid, gid)
		_ = m.files.Chown(lockLogDir, uid, gid)
	}
	change.Steps = append(change.Steps, done("plist", lockPlist))

	command := m.o.Jobs.Names().Command
	owner, _ := m.output(ctx, "stat", "-f", "%Su", "/dev/console")
	if owner == agentUser && known {
		// invariant: stopped first, so installing over a loaded agent succeeds.
		_ = m.o.Jobs.Stop(ctx, lockJob(uid))
		if err := m.o.Jobs.Load(ctx, lockJob(uid)); err != nil {
			change.Steps = append(change.Steps, settings.ChangeStep{Label: command + " bootstrap", State: catalog.StateDegraded, Detail: "it loads at the next auto-login anyway: " + firstLine(jobFailure(err))})
		} else {
			change.Steps = append(change.Steps, done(command+" bootstrap", "gui/"+strconv.Itoa(uid)))
		}
	} else {
		change.Steps = append(change.Steps, next(command, "no GUI session for "+agentUser+": the agent loads at the next auto-login"))
	}
	change.After = m.lockValues(ctx)
	return change, nil
}

func (m *Mac) undoLock(ctx context.Context) (settings.Change, error) {
	change := settings.Change{Before: m.lockValues(ctx)}
	if uid, _, known := m.agentIDs(ctx); known {
		_ = m.o.Jobs.Stop(ctx, lockJob(uid))
	}
	if err := m.files.Remove(lockPlist); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return change, err
	}
	change.Steps = []settings.ChangeStep{done("plist", "removed")}
	change.After = m.lockValues(ctx)
	return change, nil
}

func (m *Mac) lockValues(ctx context.Context) []settings.KeyValue {
	loaded := "no"
	if uid, _, known := m.agentIDs(ctx); known && m.o.Jobs.Read(ctx, lockJob(uid)).Loaded {
		loaded = "yes"
	}
	return []settings.KeyValue{{Key: "agent", Value: presence(m.exists(lockPlist))}, {Key: "loaded", Value: loaded}}
}

func lockJob(uid int) backgroundcontract.Ref {
	return backgroundcontract.Ref{Domain: backgroundcontract.Domain{UID: uid}, Label: lockLabel, File: lockPlist}
}

func jobFailure(err error) string {
	var failed *backgroundcontract.Failed
	if !errors.As(err, &failed) || failed.Err != nil {
		return err.Error()
	}
	reason := failed.Stderr
	if reason == "" {
		reason = failed.Stdout
	}
	if reason == "" {
		reason = "exited " + strconv.Itoa(failed.Exit)
	}
	return strings.Join(failed.Argv, " ") + ": " + reason
}

func (m *Mac) agentIDs(ctx context.Context) (uid, gid int, known bool) {
	u, ok := m.output(ctx, "id", "-u", agentUser)
	if !ok {
		return 0, 0, false
	}
	g, ok := m.output(ctx, "id", "-g", agentUser)
	if !ok {
		return 0, 0, false
	}
	uid, uerr := strconv.Atoi(u)
	gid, gerr := strconv.Atoi(g)
	return uid, gid, uerr == nil && gerr == nil
}

func lockPlistText() string {
	return strings.Join([]string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">`,
		`<plist version="1.0">`,
		`<dict>`,
		`    <key>Label</key>`,
		`    <string>` + lockLabel + `</string>`,
		`    <key>ProgramArguments</key>`,
		`    <array>`,
		`        <string>/bin/zsh</string>`,
		`        <string>` + lockScript + `</string>`,
		`    </array>`,
		`    <key>RunAtLoad</key>`,
		`    <true/>`,
		`    <key>StandardOutPath</key>`,
		`    <string>` + path.Join(lockLogDir, "lock-after-login.log") + `</string>`,
		`    <key>StandardErrorPath</key>`,
		`    <string>` + path.Join(lockLogDir, "lock-after-login.err.log") + `</string>`,
		`    <key>ProcessType</key>`,
		`    <string>Interactive</string>`,
		`</dict>`,
		`</plist>`,
		"",
	}, "\n")
}
