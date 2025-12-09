package auth

import (
	"errors"

	"github.com/austrian-business-infrastructure/fo/internal/user"
)

var (
	ErrInsufficientPermissions = errors.New("insufficient permissions")
	ErrCannotModifyOwner       = errors.New("cannot modify owner")
	ErrCannotDemoteSelf        = errors.New("cannot demote yourself")
)

// RoleLevel represents the permission level of a role
type RoleLevel int

const (
	LevelViewer RoleLevel = 1
	LevelMember RoleLevel = 2
	LevelAdmin  RoleLevel = 3
	LevelOwner  RoleLevel = 4
)

// RoleLevels maps roles to their permission levels
var RoleLevels = map[user.Role]RoleLevel{
	user.RoleViewer: LevelViewer,
	user.RoleMember: LevelMember,
	user.RoleAdmin:  LevelAdmin,
	user.RoleOwner:  LevelOwner,
}

// GetRoleLevel returns the permission level for a role
func GetRoleLevel(role user.Role) RoleLevel {
	if level, ok := RoleLevels[role]; ok {
		return level
	}
	return 0
}

// HasMinimumRole checks if a role has at least the specified minimum permission level
func HasMinimumRole(role user.Role, minRole user.Role) bool {
	return GetRoleLevel(role) >= GetRoleLevel(minRole)
}

// CanInvite checks if a role can invite other users
func CanInvite(role user.Role) bool {
	return HasMinimumRole(role, user.RoleAdmin)
}

// CanManageUsers checks if a role can manage (edit, deactivate) other users
func CanManageUsers(role user.Role) bool {
	return HasMinimumRole(role, user.RoleAdmin)
}

// CanAssignRole checks if an actor can assign a target role to another user
func CanAssignRole(actorRole user.Role, targetRole user.Role) bool {
	// Owner can assign any role except owner (only one owner)
	if actorRole == user.RoleOwner {
		return targetRole != user.RoleOwner
	}

	// Admin can assign member and viewer
	if actorRole == user.RoleAdmin {
		return targetRole == user.RoleMember || targetRole == user.RoleViewer
	}

	return false
}

// CanModifyUser checks if an actor can modify a target user
func CanModifyUser(actorRole user.Role, targetRole user.Role) bool {
	actorLevel := GetRoleLevel(actorRole)
	targetLevel := GetRoleLevel(targetRole)

	// Can only modify users of lower or equal level (but not equal for non-owners)
	if actorRole == user.RoleOwner {
		return true // Owner can modify anyone
	}

	return actorLevel > targetLevel
}

// ValidateRoleAssignment validates if a role change is allowed
func ValidateRoleAssignment(actorRole user.Role, currentRole user.Role, newRole user.Role, isSelf bool) error {
	// Cannot modify yourself to a lower role (use separate endpoint)
	if isSelf && GetRoleLevel(newRole) < GetRoleLevel(currentRole) {
		return ErrCannotDemoteSelf
	}

	// Cannot modify owner unless you are owner
	if currentRole == user.RoleOwner && actorRole != user.RoleOwner {
		return ErrCannotModifyOwner
	}

	// Check if actor can assign the new role
	if !CanAssignRole(actorRole, newRole) {
		return ErrInsufficientPermissions
	}

	return nil
}

// RoleHierarchy returns roles in order of permission level (highest first)
func RoleHierarchy() []user.Role {
	return []user.Role{
		user.RoleOwner,
		user.RoleAdmin,
		user.RoleMember,
		user.RoleViewer,
	}
}

// InvitableRoles returns roles that can be assigned via invitation
func InvitableRoles() []user.Role {
	return []user.Role{
		user.RoleAdmin,
		user.RoleMember,
		user.RoleViewer,
	}
}
