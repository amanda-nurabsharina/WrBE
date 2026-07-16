package config

// RoleConfig represents a system role and its default accessible menus
type RoleConfig struct {
	Name            string              `json:"name"`
	DisplayName     string              `json:"display_name"`
	Description     string              `json:"description"`
	AccessibleMenus []string            `json:"accessible_menus"`
	Permissions     map[string][]string `json:"permissions"`
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

// allActions is a helper for full access
var allActions = []string{"view", "create", "edit", "delete", "approve"}
var crudActions = []string{"view", "create", "edit", "delete"}
var viewCreateActions = []string{"view", "create"}
var viewOnly = []string{"view"}

// DefaultRoles represents initial system roles configuration
var DefaultRoles = []RoleConfig{
	{
		Name:        "super_admin",
		DisplayName: "Super Admin",
		Description: "Has access to all menus and features.",
		AccessibleMenus: []string{
			"dashboard", "employee", "leave", "attendance", "payroll", "settings", "roles",
			"products", "suppliers", "customers", "purchase-orders", "sales-orders", "packaging",
			"inward", "outward", "expired", "opname", "approver", "activity-log",
		},
		Permissions: map[string][]string{
			"dashboard":       viewOnly,
			"products":        allActions,
			"suppliers":       allActions,
			"customers":       allActions,
			"packaging":       allActions,
			"purchase-orders": allActions,
			"sales-orders":    allActions,
			"inward":          allActions,
			"outward":         allActions,
			"expired":         allActions,
			"opname":          allActions,
			"approver":        allActions,
			"activity-log":    viewOnly,
		},
	},
	{
		Name:        "admin",
		DisplayName: "Admin",
		Description: "Has administrative access except role management.",
		AccessibleMenus: []string{
			"dashboard", "employee", "leave", "attendance", "payroll", "settings",
			"products", "suppliers", "customers", "purchase-orders", "sales-orders", "packaging",
			"inward", "outward", "expired", "opname", "activity-log",
		},
		Permissions: map[string][]string{
			"dashboard":       viewOnly,
			"products":        crudActions,
			"suppliers":       crudActions,
			"customers":       crudActions,
			"packaging":       crudActions,
			"purchase-orders": crudActions,
			"sales-orders":    crudActions,
			"inward":          crudActions,
			"outward":         crudActions,
			"expired":         crudActions,
			"opname":          crudActions,
			"activity-log":    viewOnly,
		},
	},
	{
		Name:        "employee",
		DisplayName: "Employee",
		Description: "Access to personal dashboard, leaves, and attendance.",
		AccessibleMenus: []string{
			"dashboard", "leave", "attendance",
			"products", "suppliers", "customers",
			"inward", "outward", "expired", "opname",
		},
		Permissions: map[string][]string{
			"dashboard": viewOnly,
			"products":  viewOnly,
			"suppliers": viewOnly,
			"customers": viewOnly,
			"inward":    viewCreateActions,
			"outward":   viewCreateActions,
			"expired":   viewOnly,
			"opname":    viewCreateActions,
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
