package macos

import (
	"context"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	backgroundcontract "github.com/jterrazz/studio-station/apps/engine/background/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

func (i *Inspector) checks() map[catalog.Fact]func(context.Context) (settings.Fact, error) {
	return map[catalog.Fact]func(context.Context) (settings.Fact, error){
		catalog.FactAutoUpdates:  i.autoUpdates,
		catalog.FactScreenLock:   i.screenLock,
		catalog.FactDisplaySleep: i.displaySleep,
		catalog.FactRemoteLogin:  i.remoteLogin,
		catalog.FactTimeMachine:  i.timeMachine,
	}
}

func onOff(on bool) settings.Fact {
	if on {
		return settings.Fact{State: catalog.StateOK, Value: "on"}
	}
	return settings.Fact{State: catalog.StateAbsent, Value: "off"}
}

// outside: the security responses install by themselves unless
// CriticalUpdateInstall is false in the SoftwareUpdate plist; a plist
// without the key is the macOS default, on.
func (i *Inspector) autoUpdates(ctx context.Context) (settings.Fact, error) {
	plist := path.Join("/", i.o.Root, updatesPlist)
	if _, statErr := i.files.Stat(plist); statErr == nil {
		r, err := i.ask(ctx, local.Cmd{Name: "plutil", Args: []string{"-extract", "CriticalUpdateInstall", "raw", "-o", "-", plist}, Timeout: plistBudget})
		switch {
		case err != nil:
			return settings.Fact{}, err
		case !r.found:
			return settings.Unread(), nil
		}
		off := strings.TrimSpace(r.out.Stdout)
		return onOff(off != "false" && off != "0"), nil
	}
	return settings.Unread(), nil
}

var lockDelay = regexp.MustCompile(`screenLock delay is (\d+|immediate)`)

// outside: `sysadminctl -screenLock status` prints, on stderr, `screenLock
// delay is 300 seconds`, `screenLock delay is immediate` or `screenLock is off`.
func (i *Inspector) screenLock(ctx context.Context) (settings.Fact, error) {
	r, err := i.ask(ctx, local.Command("sysadminctl", "-screenLock", "status"))
	if err != nil {
		return settings.Fact{}, err
	}
	printed := r.out.Stdout + r.out.Stderr
	switch m := lockDelay.FindStringSubmatch(printed); {
	case !r.found:
		return settings.Unread(), nil
	case len(m) > 1 && m[1] == "immediate":
		return settings.Fact{State: catalog.StateOK, Value: "0"}, nil
	case len(m) > 1:
		return settings.Fact{State: catalog.StateOK, Value: m[1]}, nil
	case strings.Contains(printed, "screenLock is off"):
		return settings.Fact{State: catalog.StateAbsent, Value: "off"}, nil
	}
	return settings.Unread(), nil
}

// outside: `pmset -g` lists the active settings, ` displaysleep  10` in
// minutes, 0 for never, sometimes followed by what prevents it.
func (i *Inspector) displaySleep(ctx context.Context) (settings.Fact, error) {
	r, err := i.ask(ctx, local.Command("pmset", "-g"))
	if err != nil {
		return settings.Fact{}, err
	}
	for _, line := range strings.Split(r.out.Stdout, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "displaysleep" {
			continue
		}
		if _, err := strconv.Atoi(fields[1]); err == nil && r.ok() {
			return settings.Fact{State: catalog.StateOK, Value: fields[1]}, nil
		}
	}
	return settings.Unread(), nil
}

// outside: `systemsetup -getremotelogin` prints `Remote Login: On` only with
// admin rights; as the user it exits 0 with its refusal, and the system
// domain's overrides, readable by anyone, carry the same sshd state.
func (i *Inspector) remoteLogin(ctx context.Context) (settings.Fact, error) {
	r, err := i.ask(ctx, local.Command("systemsetup", "-getremotelogin"))
	if err != nil {
		return settings.Fact{}, err
	}
	switch printed := r.out.Stdout + r.out.Stderr; {
	case r.ok() && strings.Contains(printed, "Remote Login: On"):
		return onOff(true), nil
	case r.ok() && strings.Contains(printed, "Remote Login: Off"):
		return onOff(false), nil
	}
	if i.o.Jobs == nil {
		return settings.Unread(), nil
	}
	overrides := i.o.Jobs.Overrides(ctx, backgroundcontract.Domain{System: true})
	if overrides == nil {
		return settings.Unread(), nil
	}
	return onOff(overrides["com.openssh.sshd"]), nil
}

var backupStamp = regexp.MustCompile(`(\d{4}-\d{2}-\d{2}-\d{6})(\.backup)?/?$`)

// outside: `tmutil latestbackup` prints the path of the last backup, named
// for its local time (`2026-10-26-101010.backup`); a Mac with no backup says
// `No machine directory found` or has no destination. With no destination at
// all, macOS 26 prints `Failed to mount backup destination` and exits 0, so
// `tmutil destinationinfo` is asked whether one is configured.
func (i *Inspector) timeMachine(ctx context.Context) (settings.Fact, error) {
	r, err := i.ask(ctx, local.Command("tmutil", "latestbackup"))
	if err != nil {
		return settings.Fact{}, err
	}
	printed := strings.TrimSpace(r.out.Stdout)
	if m := backupStamp.FindStringSubmatch(printed); r.ok() && m != nil {
		if at, parseErr := time.ParseInLocation("2006-01-02-150405", m[1], time.Local); parseErr == nil {
			return settings.Fact{State: catalog.StateOK, Value: at.UTC().Format(time.RFC3339)}, nil
		}
	}
	if said := printed + r.out.Stderr; r.found && (strings.Contains(said, "No machine directory") || strings.Contains(said, "No destinations") || strings.Contains(said, "No backups")) {
		return settings.Fact{State: catalog.StateAbsent, Value: "never"}, nil
	}
	if !r.found {
		return settings.Unread(), nil
	}
	d, err := i.ask(ctx, local.Command("tmutil", "destinationinfo"))
	if err != nil {
		return settings.Fact{}, err
	}
	if strings.Contains(d.out.Stdout+d.out.Stderr, "No destinations configured") {
		return settings.Fact{State: catalog.StateAbsent, Value: "never"}, nil
	}
	return settings.Unread(), nil
}
