package calls

func (s *Provisioner) settingsPath() string { return s.o.PreferencesFile }

func (s *Provisioner) exists(p string) bool {
	return s.m.Exists(p)
}
