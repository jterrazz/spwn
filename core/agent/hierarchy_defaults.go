package agent

// DefaultHierarchy is the built-in three-role hierarchy used when no custom
// hierarchy is specified. It contains a chief (level 0) who commands
// managers (level 1) and workers (level 2).
var DefaultHierarchy = Hierarchy{
	Slug:        "default",
	Name:        "Default",
	Description: "Built-in three-tier hierarchy",
	Roles: []Role{
		{
			Name:        "chief",
			Level:       0,
			CanCommand:  []string{"manager", "worker"},
			MaxPerWorld: 1,
			Permissions: []string{"delegate", "review", "orchestrate"},
		},
		{
			Name:        "manager",
			Level:       1,
			CanCommand:  []string{"worker"},
			ReportsTo:   "chief",
			Permissions: []string{"delegate", "review", "execute"},
		},
		{
			Name:        "worker",
			Level:       2,
			ReportsTo:   "manager",
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
