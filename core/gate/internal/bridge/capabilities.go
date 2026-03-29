package bridge

import "fmt"

// CheckCapability verifies that the requested operation is allowed by the
// bridge's declared capabilities. If the bridge has no capabilities defined,
// all operations are allowed. Otherwise, the operation must appear in the list.
func CheckCapability(b GateBridge, operation string) error {
	if len(b.Capabilities) == 0 {
		return nil
	}
	for _, cap := range b.Capabilities {
		if cap == operation {
			return nil
		}
	}
	return fmt.Errorf("element bridge %q: operation %q not allowed (capabilities: %v)", b.As, operation, b.Capabilities)
}
