package usecase

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

var (
	newAuditLogFn = entity.NewAuditLog
	jsonMarshalFn = json.Marshal
	newDocumentFn = entity.NewDocument
	newWorkspaceFn = entity.NewWorkspace
	newWorkspaceMemberFn = entity.NewWorkspaceMember
	newUserFn = entity.NewUser
	newRefreshTokenEntityFn = entity.NewRefreshToken
	newDocumentVersionFn = entity.NewDocumentVersion
	rotateRefreshTokenFn = func(token *entity.RefreshToken, replacementID uuid.UUID, now time.Time) error {
		return token.Rotate(replacementID, now)
	}
	revokeRefreshTokenFn = func(token *entity.RefreshToken, now time.Time) error {
		return token.Revoke(now)
	}
	suspendUserFn = func(u *entity.User, now time.Time) error { return u.Suspend(now) }
	reactivateUserFn = func(u *entity.User, now time.Time) error { return u.Reactivate(now) }
	logicalDeleteUserFn = func(u *entity.User, email, passwordHash string, now time.Time) error {
		return u.LogicalDelete(email, passwordHash, now)
	}
	replaceDocumentContentFn = func(d *entity.Document, content string, actorID uuid.UUID, expectedRevision uint64, now time.Time) error {
		return d.ReplaceContent(content, actorID, expectedRevision, now)
	}
	renameWorkspaceFn = func(w *entity.Workspace, name string, now time.Time) error {
		return w.Rename(name, now)
	}
	logicalDeleteWorkspaceFn = func(w *entity.Workspace, deletedBy uuid.UUID, reason string, now time.Time) error {
		return w.LogicalDelete(deletedBy, reason, now)
	}
)
