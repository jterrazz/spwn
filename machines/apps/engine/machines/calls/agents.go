package calls

import (
	"path"
	"strings"
)

func (s *Machinist) LogPath(name string) string { return path.Join(s.o.Logs, name+".log") }

func (s *Machinist) KnownHostsPath(name string) string { return path.Join(s.o.KnownHosts, name) }

func (s *Machinist) KeyPath() string { return s.img.KeyPath() }

func (s *Machinist) VMUser() (string, error) {
	cfg, err := s.prefs()
	if err != nil {
		return "", err
	}
	return s.vmUser(cfg)
}

func (s *Machinist) ThisMacName() string {
	cfg, _ := s.prefs()
	return s.thisMac(cfg)
}

func (s *Machinist) ThisMacLabel() string {
	// value: this Mac's name without the owner's user before it, as the
	// fleet's other Macs are named ("mac-mini").
	cfg, _ := s.prefs()
	name, user := s.thisMac(cfg), cfg.Macs[cfg.ThisMachine].User
	if user == "" {
		user = s.h.User()
	}
	if short, ok := strings.CutPrefix(name, user+"-"); ok && short != "" {
		return short
	}
	return name
}
