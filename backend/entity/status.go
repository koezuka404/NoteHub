package entity

type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusSuspended UserStatus = "suspended"
	UserStatusDeleted   UserStatus = "deleted"
)

func (s UserStatus) IsValid() bool {
	return s == UserStatusActive ||
		s == UserStatusSuspended ||
		s == UserStatusDeleted
}

type RefreshTokenStatus string

const (
	RefreshTokenStatusActive  RefreshTokenStatus = "active"
	RefreshTokenStatusRotated RefreshTokenStatus = "rotated"
	RefreshTokenStatusRevoked RefreshTokenStatus = "revoked"
	RefreshTokenStatusExpired RefreshTokenStatus = "expired"
)

func (s RefreshTokenStatus) IsValid() bool {
	return s == RefreshTokenStatusActive ||
		s == RefreshTokenStatusRotated ||
		s == RefreshTokenStatusRevoked ||
		s == RefreshTokenStatusExpired
}

type WorkspaceRole string

const (
	WorkspaceRoleHost   WorkspaceRole = "host"
	WorkspaceRoleMember WorkspaceRole = "member"
)

func (r WorkspaceRole) IsValid() bool {
	return r == WorkspaceRoleHost || r == WorkspaceRoleMember
}

type DocumentVersionType string

const (
	DocumentVersionAutoSave       DocumentVersionType = "auto_save"
	DocumentVersionManualSave     DocumentVersionType = "manual_save"
	DocumentVersionBeforeRestore  DocumentVersionType = "before_restore"
	DocumentVersionRestore        DocumentVersionType = "restore"
)

func (t DocumentVersionType) IsValid() bool {
	return t == DocumentVersionAutoSave ||
		t == DocumentVersionManualSave ||
		t == DocumentVersionBeforeRestore ||
		t == DocumentVersionRestore
}
