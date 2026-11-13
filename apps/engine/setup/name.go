package setup

import "regexp"

var machineName = regexp.MustCompile(`^[a-z0-9-]{1,40}$`)

func ValidName(name string) bool { return machineName.MatchString(name) }
