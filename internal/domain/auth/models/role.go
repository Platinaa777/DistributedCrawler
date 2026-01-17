package models

import "fmt"

type Role string

const (
	RoleRead          Role = "READ"
	RoleReadWrite     Role = "READ_WRITE"
	RoleAdministrator Role = "ADMINISTRATOR"
)

func ParseRole(value string) (Role, error) {
	role := Role(value)
	if !role.IsValid() {
		return RoleRead, fmt.Errorf("invalid role: %s", value)
	}
	return role, nil
}

func (r Role) IsValid() bool {
	switch r {
	case RoleRead, RoleReadWrite, RoleAdministrator:
		return true
	default:
		return false
	}
}

func (r Role) Level() int {
	switch r {
	case RoleRead:
		return 1
	case RoleReadWrite:
		return 2
	case RoleAdministrator:
		return 3
	default:
		return 0
	}
}
