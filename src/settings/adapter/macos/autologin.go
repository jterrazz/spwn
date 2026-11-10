package macos

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/settings"
)

const (
	agentUser     = "jterrazz.agent"
	agentHome     = "/Users/" + agentUser
	loginWindow   = "/Library/Preferences/com.apple.loginwindow"
	kcpasswordTo  = "/etc/kcpassword"
	autologinHint = "verify end to end: station machines restart <name>"
)

func (m *Mac) observeAutologin(ctx context.Context, want []settings.KeyValue) settings.Observation {
	now := m.autologinValues(ctx)
	o := judged(now, want, agentUser, kcpasswordTo+" is missing")
	if now[0].Value != agentUser {
		o.State, o.Detail = catalog.StateAbsent, ""
	}
	return o
}

func (m *Mac) applyAutologin(ctx context.Context, values map[string]string) (settings.Change, error) {
	if _, ok := m.run.LookPath("sysadminctl"); !ok {
		return settings.Change{}, errors.New("sysadminctl not found")
	}
	if !m.exists(agentHome) {
		return settings.Change{}, errors.New("user " + agentUser + " does not exist on this Mac")
	}
	password := values["password"]
	if password == "" {
		return settings.Change{}, errors.New("empty password: refusing to set " + kcpasswordTo + " with an empty value")
	}
	change := settings.Change{Before: m.autologinValues(ctx)}
	if _, err := m.must(ctx, "defaults", "write", loginWindow, "DisableFDEAutoLogin", "-bool", "NO"); err != nil {
		return change, err
	}
	change.Steps = append(change.Steps, done("DisableFDEAutoLogin = NO", ""))

	// outside: on macOS 26 `sysadminctl -autoLogin set` refuses on a FileVault disk
	// whatever DisableFDEAutoLogin says, so the login is written as /etc/kcpassword (the
	// XOR cipher loginwindow has read since 10.4) plus autoLoginUser; loginwindow at boot
	// still honors DisableFDEAutoLogin.
	if err := m.files.WriteFile(kcpasswordTo, encodeKCPassword(password), 0o600); err != nil {
		return change, fmt.Errorf("write %s: %w", kcpasswordTo, err)
	}
	if err := m.files.Chmod(kcpasswordTo, 0o600); err != nil {
		return change, err
	}
	// outside: /etc/kcpassword must be owned by root:wheel.
	_ = m.files.Chown(kcpasswordTo, 0, 0)
	change.Steps = append(change.Steps, done(kcpasswordTo, "written (root:wheel 0600)"))

	if _, err := m.must(ctx, "defaults", "write", loginWindow, "autoLoginUser", "-string", agentUser); err != nil {
		return change, err
	}
	change.Steps = append(change.Steps, done("autoLoginUser = "+agentUser, ""))
	if !m.exists(kcpasswordTo) {
		return change, errors.New(kcpasswordTo + " missing after write")
	}
	if got, ok := m.output(ctx, "defaults", "read", loginWindow, "autoLoginUser"); !ok || got != agentUser {
		return change, errors.New("autoLoginUser cross-check failed (got \"" + got + "\")")
	}
	// value: clearing the screen-locked-after-resume flag keeps the auto-login session
	// interactive; lock-after-login owns the lock itself.
	_, _ = m.output(ctx, "defaults", "delete", loginWindow, "autoLoginUserScreenLocked")

	change.After = m.autologinValues(ctx)
	change.Steps = append(change.Steps, next("next", autologinHint))
	return change, nil
}

func (m *Mac) undoAutologin(ctx context.Context) (settings.Change, error) {
	change := settings.Change{Before: m.autologinValues(ctx)}
	_, _ = m.output(ctx, "sysadminctl", "-autologin", "off")
	if err := m.files.Remove(kcpasswordTo); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return change, err
	}
	for _, key := range []string{"autoLoginUser", "oneTimeAutoLogin", "autoLoginUserScreenLocked"} {
		_, _ = m.output(ctx, "defaults", "delete", loginWindow, key)
	}
	change.Steps = []settings.ChangeStep{done(kcpasswordTo, "removed"), done("autoLoginUser", "cleared")}
	change.After = m.autologinValues(ctx)
	return change, nil
}

func (m *Mac) autologinValues(ctx context.Context) []settings.KeyValue {
	user, ok := m.output(ctx, "defaults", "read", loginWindow, "autoLoginUser")
	if !ok {
		user = ""
	}
	return []settings.KeyValue{{Key: "autoLoginUser", Value: firstLine(user)}, {Key: kcpasswordTo, Value: presence(m.exists(kcpasswordTo))}}
}
