package platform

import (
	"fmt"
	"strings"
)

// CheckName refuses a name a path is built from, of a tool, an agent or a
// world, that is not one name: empty, "." or "..", or holding a /, a :, a
// \ or a NUL. The refusal is by the name alone, so it reads the same
// whatever the disk holds where the name would lead.
func CheckName(kind, name string) error {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/:\\\x00") {
		return fmt.Errorf("%s name %q is not one name: a name is not empty, \".\" or \"..\", and holds no /, :, \\ or NUL", kind, name)
	}
	return nil
}
