package msc1929

const (
	// RoleAdmin is catch-all user for any queries
	RoleAdmin = "m.role.admin"
	// RoleModerator is intended for moderation requests
	// TODO: currently unused, as MSC4121 mandates the use of RoleModeratorUnstable until it is merged into the spec,
	// ref: https://github.com/FSG-Cat/matrix-spec-proposals/blob/FSG-Cat-Moderation-Role-well-known-support-record/proposals/4121-m.role.moderator.md#unstable-prefix
	RoleModerator = "m.role.moderator"
	// RoleModeratorUnstable is intended for moderation requests, used until MSC4121 is merged into the spec
	RoleModeratorUnstable = "support.feline.msc4121.role.moderator"
	// RoleSecurity is intended for sensitive requests
	RoleSecurity = "m.role.security"
	// RoleDPO is intended for data protection officer contacts
	// TODO: currently unused, as MSC4265 mandates the use of RoleDPOUnstable until it is merged into the spec,
	// ref: bitbucket.org/helloticketscode/curator/internal/services/emailnotifier
	RoleDPO = "m.role.dpo"
	// RoleDPOUnstable is intended for data protection officer contacts, used until MSC4265 is merged into the spec
	RoleDPOUnstable = "org.matrix.msc4265.role.dpo"
)

// SupportedRoles contains all roles that are supported by the support file
var SupportedRoles = []string{
	RoleAdmin, RoleModerator, RoleModeratorUnstable, RoleSecurity, RoleDPO, RoleDPOUnstable,
}

// Contact details
type Contact struct {
	Email    string `json:"email_address,omitempty" yaml:"email_address,omitempty"`
	MatrixID string `json:"matrix_id,omitempty" yaml:"matrix_id,omitempty"`
	Role     string `json:"role,omitempty" yaml:"role,omitempty"`
}

// IsEmpty checks if contact contains at least one contact (either email or mxid)
func (c *Contact) IsEmpty() bool {
	if c == nil {
		return true
	}
	return c.Email == "" && c.MatrixID == ""
}

// IsAdmin checks if contact has admin role
func (c *Contact) IsAdmin() bool {
	return c.Role == RoleAdmin
}

// IsModerator checks if contact has moderator role
func (c *Contact) IsModerator() bool {
	return c.Role == RoleModeratorUnstable || c.Role == RoleModerator
}

// IsDPO checks if contact has DPO role
func (c *Contact) IsDPO() bool {
	return c.Role == RoleDPOUnstable || c.Role == RoleDPO
}

// IsSecurity checks if contact has security role
func (c *Contact) IsSecurity() bool {
	return c.Role == RoleSecurity
}
