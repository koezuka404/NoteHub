package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

type RemoveMemberInput struct {
	UserID       uuid.UUID
	WorkspaceID  uuid.UUID
	TargetUserID uuid.UUID
	IPAddress    string
}

func (uc *MemberUseCase) RemoveMember(ctx context.Context, input RemoveMemberInput) error {
	if err := uc.requireHost(ctx, input.UserID, input.WorkspaceID); err != nil {
		return err
	}

	targetMember, found, err := uc.members.FindByWorkspaceAndUser(ctx, input.WorkspaceID, input.TargetUserID)
	if err != nil {
		return fmt.Errorf("find target workspace member: %w", err)
	}
	if !found {
		return ErrMemberNotFound
	}
	if targetMember.IsHost() {
		return ErrCannotRemoveHost
	}

	now := uc.currentTime()
	return uc.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.members.Delete(txCtx, input.WorkspaceID, input.TargetUserID); err != nil {
			return fmt.Errorf("delete workspace member: %w", err)
		}

		audit, err := entity.NewAuditLog(&input.UserID, "MEMBER_REMOVED", "workspace_member", &targetMember.ID, nil, now)
		if err != nil {
			return fmt.Errorf("create audit log entity: %w", err)
		}
		audit.IPAddress = input.IPAddress
		if err := uc.auditLogs.Create(txCtx, &audit); err != nil {
			return fmt.Errorf("save audit log: %w", err)
		}
		return nil
	})
}
