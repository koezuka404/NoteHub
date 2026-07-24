package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

type IMemberUsecase interface {
	SearchUser(ctx context.Context, input SearchUserInput) (*SearchUserOutput, error)
	ListMembers(ctx context.Context, input ListMembersInput) ([]MemberListItem, error)
	AddMember(ctx context.Context, input AddMemberInput) (*AddMemberOutput, error)
	RemoveMember(ctx context.Context, input RemoveMemberInput) error
}

type IMemberUserRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*entity.User, bool, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, bool, error)
}

type IAccessCheck interface {
	CheckWorkspaceAccess(ctx context.Context, input CheckWorkspaceAccessInput) (*WorkspaceAccessResult, error)
}

type MemberUseCase struct {
	users        IMemberUserRepository
	members      IWorkspaceMemberRepository
	auditLogs    IAuditLogRepository
	transactions ITransactionManager
	access       IAccessCheck
	now          func() time.Time
}

func NewMemberUseCase(
	users IMemberUserRepository,
	members IWorkspaceMemberRepository,
	auditLogs IAuditLogRepository,
	transactions ITransactionManager,
	access IAccessCheck,
) *MemberUseCase {
	return &MemberUseCase{
		users:        users,
		members:      members,
		auditLogs:    auditLogs,
		transactions: transactions,
		access:       access,
		now:          time.Now,
	}
}

func (uc *MemberUseCase) currentTime() time.Time {
	return uc.now().UTC()
}

func (uc *MemberUseCase) requireHost(ctx context.Context, userID, workspaceID uuid.UUID) error {
	role := entity.WorkspaceRoleHost
	_, err := uc.access.CheckWorkspaceAccess(ctx, CheckWorkspaceAccessInput{
		UserID: userID, WorkspaceID: workspaceID, RequiredRole: &role,
	})
	return err
}

func (uc *MemberUseCase) requireMember(ctx context.Context, userID, workspaceID uuid.UUID) error {
	_, err := uc.access.CheckWorkspaceAccess(ctx, CheckWorkspaceAccessInput{
		UserID: userID, WorkspaceID: workspaceID,
	})
	return err
}
