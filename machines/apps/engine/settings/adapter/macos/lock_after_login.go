package macos

import (
	"context"
	"errors"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	backgroundcontract "github.com/jterrazz/studio-station/apps/engine/background/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

const (
	lockLabel = "ai.jterrazz.lock-after-login"
	usersRoot = "/Users"
)

var shortName = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]*$`)

type lockPaths struct{ user, script, plist, logDir string }

func lockPathsOf(user string) lockPaths {
	home := usersRoot + "/" + user
	return lockPaths{
		user:   user,
		script: home + "/" + catalog.LockAfterLoginScript,
		plist:  home + "/Library/LaunchAgents/" + lockLabel + ".plist",
		logDir: home + "/.openclaw/logs",
	}
}

func (m *Mac) loginUser(ctx context.Context) string {
	user, ok := m.output(ctx, "defaults", "read", loginWindow, "autoLoginUser")
	if !ok {
		return ""
	}
	return firstLine(user)
}

func validShortName(user string) bool {
	return shortName.MatchString(user) && !strings.Contains(user, "..")
}

func (m *Mac) lockInstalled() []string {
	var users []string
	_ = m.files.WalkDir(usersRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() && p != usersRoot {
				return fs.SkipDir
			}
			return err
		}
		if !d.IsDir() || path.Dir(p) != usersRoot {
			return nil
		}
		if user := path.Base(p); validShortName(user) && m.exists(lockPathsOf(user).plist) {
			users = append(users, user)
		}
		return fs.SkipDir
	})
	return users
}

func (m *Mac) removeLock(ctx context.Context, user string) error {
	lp := lockPathsOf(user)
	if uid, _, ids := m.userIDs(ctx, user); ids {
		_ = m.o.Jobs.Stop(ctx, lockJob(lp, uid))
	}
	if err := m.files.Remove(lp.plist); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func (m *Mac) observeLock(ctx context.Context, want []settings.KeyValue) settings.Observation {
	login := m.loginUser(ctx)
	installed := m.lockInstalled()
	target := ""
	switch {
	case slices.Contains(installed, login):
		target = login
	case len(installed) > 0:
		target = installed[0]
	case validShortName(login):
		target = login
	}
	if target == "" {
		return settings.Observation{State: catalog.StateAbsent, After: want}
	}
	now := m.lockValues(ctx, lockPathsOf(target))
	o := judged(now, want, "agent loaded", "agent installed, not loaded")
	if now[0].Value != "present" {
		o.State, o.Detail = catalog.StateAbsent, ""
	}
	return o
}

func (m *Mac) applyLock(ctx context.Context) (settings.Change, error) {
	login := m.loginUser(ctx)
	if login == "" {
		return settings.Change{}, errors.New("no user logs in on this Mac: apply autologin first")
	}
	if !validShortName(login) {
		return settings.Change{}, errors.New("autoLoginUser " + strconv.Quote(login) + " is not a macOS short name: lock-after-login refuses it")
	}
	lp := lockPathsOf(login)
	if !m.exists(lp.script) {
		return settings.Change{}, errors.New("missing " + lp.script + ", expected the openclaw lock-after-login script")
	}
	change := settings.Change{Before: m.lockValues(ctx, lp)}
	for _, old := range m.lockInstalled() {
		if old == login {
			continue
		}
		if err := m.removeLock(ctx, old); err != nil {
			return change, err
		}
		change.Steps = append(change.Steps, done("plist", "removed under "+old))
	}
	// invariant: launchd executes this script: the x bit is the point.
	if err := m.files.Chmod(lp.script, 0o755); err != nil {
		return change, err
	}
	if err := m.files.MkdirAll(lp.logDir, 0o750); err != nil {
		return change, err
	}
	if err := m.write(lp.plist, []byte(lockPlistText(lp)), 0o600); err != nil {
		return change, err
	}
	uid, gid, ids := m.userIDs(ctx, lp.user)
	if ids {
		_ = m.files.Chown(lp.plist, uid, gid)
		_ = m.files.Chown(lp.logDir, uid, gid)
	}
	change.Steps = append(change.Steps, done("plist", lp.plist))

	command := m.o.Jobs.Names().Command
	owner, _ := m.output(ctx, "stat", "-f", "%Su", "/dev/console")
	if owner == lp.user && ids {
		// invariant: stopped first, so installing over a loaded agent succeeds.
		_ = m.o.Jobs.Stop(ctx, lockJob(lp, uid))
		if err := m.o.Jobs.Load(ctx, lockJob(lp, uid)); err != nil {
			change.Steps = append(change.Steps, settings.ChangeStep{Label: command + " bootstrap", State: catalog.StateDegraded, Detail: "it loads at the next auto-login anyway: " + firstLine(jobFailure(err))})
		} else {
			change.Steps = append(change.Steps, done(command+" bootstrap", "gui/"+strconv.Itoa(uid)))
		}
	} else {
		change.Steps = append(change.Steps, next(command, "no GUI session for "+lp.user+": the agent loads at the next auto-login"))
	}
	change.After = m.lockValues(ctx, lp)
	return change, nil
}

func (m *Mac) undoLock(ctx context.Context) (settings.Change, error) {
	installed := m.lockInstalled()
	if len(installed) == 0 {
		return settings.Change{Steps: []settings.ChangeStep{done("plist", "nothing to remove")}}, nil
	}
	change := settings.Change{Before: m.lockValues(ctx, lockPathsOf(installed[0]))}
	for _, user := range installed {
		if err := m.removeLock(ctx, user); err != nil {
			return change, err
		}
	}
	change.Steps = []settings.ChangeStep{done("plist", "removed")}
	change.After = m.lockValues(ctx, lockPathsOf(installed[0]))
	return change, nil
}

func (m *Mac) lockValues(ctx context.Context, lp lockPaths) []settings.KeyValue {
	loaded := "no"
	if uid, _, ids := m.userIDs(ctx, lp.user); ids && m.o.Jobs.Read(ctx, lockJob(lp, uid)).Loaded {
		loaded = "yes"
	}
	return []settings.KeyValue{{Key: "agent", Value: presence(m.exists(lp.plist))}, {Key: "loaded", Value: loaded}}
}

func lockJob(lp lockPaths, uid int) backgroundcontract.Ref {
	return backgroundcontract.Ref{Domain: backgroundcontract.Domain{UID: uid}, Label: lockLabel, File: lp.plist}
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

func (m *Mac) userIDs(ctx context.Context, user string) (uid, gid int, known bool) {
	if user == "" {
		return 0, 0, false
	}
	u, ok := m.output(ctx, "id", "-u", user)
	if !ok {
		return 0, 0, false
	}
	g, ok := m.output(ctx, "id", "-g", user)
	if !ok {
		return 0, 0, false
	}
	uid, uerr := strconv.Atoi(u)
	gid, gerr := strconv.Atoi(g)
	return uid, gid, uerr == nil && gerr == nil
}

func lockPlistText(lp lockPaths) string {
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
		`        <string>` + lp.script + `</string>`,
		`    </array>`,
		`    <key>RunAtLoad</key>`,
		`    <true/>`,
		`    <key>StandardOutPath</key>`,
		`    <string>` + path.Join(lp.logDir, "lock-after-login.log") + `</string>`,
		`    <key>StandardErrorPath</key>`,
		`    <string>` + path.Join(lp.logDir, "lock-after-login.err.log") + `</string>`,
		`    <key>ProcessType</key>`,
		`    <string>Interactive</string>`,
		`</dict>`,
		`</plist>`,
		"",
	}, "\n")
}
