package commands

import (
	"fmt"
	"strings"

	"github.com/jterrazz/universe/internal/config"
)

// parseInteractions parses raw --interaction flag values into Interaction structs.
// Format: "source:as:cap1,cap2" or "source:as" (no capabilities).
func parseInteractions(raw []string) ([]config.Interaction, error) {
	var interactions []config.Interaction
	for _, r := range raw {
		parts := strings.SplitN(r, ":", 3)
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid interaction format %q: expected \"source:as\" or \"source:as:cap1,cap2\"", r)
		}

		source := parts[0]
		as := parts[1]
		if source == "" || as == "" {
			return nil, fmt.Errorf("invalid interaction format %q: source and as must be non-empty", r)
		}

		ia := config.Interaction{
			Source: source,
			As:     as,
		}

		if len(parts) == 3 && parts[2] != "" {
			ia.Capabilities = strings.Split(parts[2], ",")
		}

		interactions = append(interactions, ia)
	}
	return interactions, nil
}
