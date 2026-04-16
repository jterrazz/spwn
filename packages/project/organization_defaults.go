package project

// DefaultOrganization is the built-in three-role organization used when no custom
// organization is specified. It contains a chief (level 0) who commands
// managers (level 1) and workers (level 2).
var DefaultOrganization = Organization{
	Slug:        "default",
	Name:        "Default",
	Description: "Built-in three-tier organization",
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

// EnsureDefaultOrganization creates the default organization on disk if it does
// not already exist. It is safe to call multiple times.
func EnsureDefaultOrganization() error {
	if _, err := GetOrganization(DefaultOrganization.Slug); err == nil {
		return nil // already exists
	}
	return CreateOrganization(DefaultOrganization)
}
