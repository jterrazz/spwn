package words

import "strings"

func NeededBy(call string) string {
	// value: what a call's need is required by, as the owner reads it: its
	// family, or the doing of it when only that verb needs it.
	family, _, _ := strings.Cut(call, ".")
	switch family {
	case "ports":
		return "port sharing"
	case "accounts":
		return "account sharing"
	}
	return family
}

func NeedFix(program, item string) string {
	// value: the gesture that provides a need: its catalogue item installed,
	// else the program put on the search path by hand.
	if item == "" {
		return "install " + program + ", then run it again"
	}
	return "station apps install " + item
}

func NeedAsk(item string) string {
	// value: the question a terminal puts before it installs a need.
	return "Install " + item + " now?"
}
