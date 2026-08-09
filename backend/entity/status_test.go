package entity

import "testing"

func TestUserStatus_IsValid(t *testing.T) {
	valid := []UserStatus{UserStatusActive, UserStatusSuspended, UserStatusDeleted}
	for _, s := range valid {
		if !s.IsValid() {
			t.Fatalf("%q should be valid", s)
		}
	}
	if UserStatus("unknown").IsValid() {
		t.Fatal("unknown user status should be invalid")
	}
}

func TestRefreshTokenStatus_IsValid(t *testing.T) {
	valid := []RefreshTokenStatus{
		RefreshTokenStatusActive,
		RefreshTokenStatusRotated,
		RefreshTokenStatusRevoked,
		RefreshTokenStatusExpired,
	}
	for _, s := range valid {
		if !s.IsValid() {
			t.Fatalf("%q should be valid", s)
		}
	}
	if RefreshTokenStatus("bad").IsValid() {
		t.Fatal("unknown refresh token status should be invalid")
	}
}

func TestWorkspaceRole_IsValid(t *testing.T) {
	if !WorkspaceRoleHost.IsValid() || !WorkspaceRoleMember.IsValid() {
		t.Fatal("host and member should be valid")
	}
	if WorkspaceRole("guest").IsValid() {
		t.Fatal("guest should be invalid")
	}
}

func TestDocumentVersionType_IsValid(t *testing.T) {
	valid := []DocumentVersionType{
		DocumentVersionAutoSave,
		DocumentVersionManualSave,
		DocumentVersionBeforeRestore,
		DocumentVersionRestore,
	}
	for _, vt := range valid {
		if !vt.IsValid() {
			t.Fatalf("%q should be valid", vt)
		}
	}
	if DocumentVersionType("snapshot").IsValid() {
		t.Fatal("snapshot should be invalid")
	}
}
