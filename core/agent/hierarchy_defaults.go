package agent

// DefaultHierarchy is the built-in two-role hierarchy used when no custom
// hierarchy is specified. It contains a governor (level 0) who commands
// citizens (level 1).
var DefaultHierarchy = Hierarchy{
	Slug:        "default",
	Name:        "Default",
	Description: "Built-in governor/citizen hierarchy",
	Roles: []Role{
		{
			Name:        "governor",
			Level:       0,
			CanCommand:  []string{"citizen"},
			MaxPerWorld: 1,
			Permissions: []string{"delegate", "review", "orchestrate"},
		},
		{
			Name:        "citizen",
			Level:       1,
			ReportsTo:   "governor",
			Permissions: []string{"execute", "report"},
		},
	},
}

// EnsureDefaultHierarchy creates the default hierarchy on disk if it does
// not already exist. It is safe to call multiple times.
func EnsureDefaultHierarchy() error {
	if _, err := GetHierarchy(DefaultHierarchy.Slug); err == nil {
		return nil // already exists
	}
	return CreateHierarchy(DefaultHierarchy)
}
