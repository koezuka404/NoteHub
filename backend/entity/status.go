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
)

func (s RefreshTokenStatus) IsValid() bool {
	return s == RefreshTokenStatusActive ||
		s == RefreshTokenStatusRotated ||
		s == RefreshTokenStatusRevoked
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
	DocumentVersionManual   DocumentVersionType = "manual"
	DocumentVersionAutosave DocumentVersionType = "autosave"
	DocumentVersionRestore  DocumentVersionType = "restore"
)

func (t DocumentVersionType) IsValid() bool {
	return t == DocumentVersionManual ||
		t == DocumentVersionAutosave ||
		t == DocumentVersionRestore
}
