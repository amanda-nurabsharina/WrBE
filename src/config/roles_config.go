package config

// RoleConfig represents a system role and its default accessible menus
type RoleConfig struct {
	Name            string   `json:"name"`
	DisplayName     string   `json:"display_name"`
	Description     string   `json:"description"`
	AccessibleMenus []string `json:"accessible_menus"`
}

// UserConfig represents a default user structure for seeding
type UserConfig struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// DefaultRoles represents initial system roles configuration
var DefaultRoles = []RoleConfig{
	{
		Name:        "super_admin",
		DisplayName: "Super Admin",
		Description: "Has access to all menus and features.",
		AccessibleMenus: []string{
			"dashboard", "employee", "leave", "attendance", "payroll", "settings", "roles",
		},
	},
	{
		Name:        "admin",
		DisplayName: "Admin",
		Description: "Has administrative access except role management.",
		AccessibleMenus: []string{
			"dashboard", "employee", "leave", "attendance", "payroll", "settings",
		},
	},
	{
		Name:        "employee",
		DisplayName: "Employee",
		Description: "Access to personal dashboard, leaves, and attendance.",
		AccessibleMenus: []string{
			"dashboard", "leave", "attendance",
		},
	},
}

// DefaultUsers represents initial users configuration to seed
var DefaultUsers = []UserConfig{
	{
		Username:  "superadmin",
		Password:  "superadmin12345",
		Email:     "superadmin@warehouse.com",
		Role:      "super_admin",
		FirstName: "Super",
		LastName:  "Admin",
	},
}
