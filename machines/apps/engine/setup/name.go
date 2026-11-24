package setup

import "regexp"

var machineName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,39}$`)

func ValidName(name string) bool { return machineName.MatchString(name) }
