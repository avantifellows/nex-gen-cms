package auth

const (
	RoleViewer = "viewer"
	RoleEditor = "editor"
	RoleAdmin  = "admin"
)

// rank gives a comparable level so RequireRole can do "at least editor" checks.
func rank(role string) int {
	switch role {
	case RoleAdmin:
		return 3
	case RoleEditor:
		return 2
	case RoleViewer:
		return 1
	default:
		return 0
	}
}

// AtLeast reports whether `have` satisfies the minimum role `need`.
func AtLeast(have, need string) bool {
	return rank(have) >= rank(need)
}

// ValidRole reports whether r is one of the three accepted roles.
func ValidRole(r string) bool {
	return r == RoleViewer || r == RoleEditor || r == RoleAdmin
}
