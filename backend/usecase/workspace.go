package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

type IWorkspaceUsecase interface {
	CreateWorkspace(ctx context.Context, input CreateWorkspaceInput) (*CreateWorkspaceOutput, error)
	ListWorkspaces(ctx context.Context, input ListWorkspacesInput) ([]WorkspaceListItem, error)
	GetWorkspace(ctx context.Context, input GetWorkspaceInput) (*GetWorkspaceOutput, error)
	UpdateWorkspace(ctx context.Context, input UpdateWorkspaceInput) (*UpdateWorkspaceOutput, error)
	DeleteWorkspace(ctx context.Context, input DeleteWorkspaceInput) (*DeleteWorkspaceOutput, error)
}

type IWorkspaceRepository interface {
	Create(ctx context.Context, workspace *entity.Workspace) error
	FindByID(ctx context.Context, workspaceID uuid.UUID) (*entity.Workspace, bool, error)
	FindByIDForUpdate(ctx context.Context, workspaceID uuid.UUID) (*entity.Workspace, bool, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]entity.Workspace, error)
	Update(ctx context.Context, workspace *entity.Workspace) error
}

type IWorkspaceMemberRepository interface {
	Create(ctx context.Context, member *entity.WorkspaceMember) error
	FindByWorkspaceAndUser(ctx context.Context, workspaceID, userID uuid.UUID) (*entity.WorkspaceMember, bool, error)
	FindByWorkspaceID(ctx context.Context, workspaceID uuid.UUID) ([]entity.WorkspaceMember, error)
	Exists(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error)
	Delete(ctx context.Context, workspaceID, userID uuid.UUID) error
}

type IWorkspaceUserRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*entity.User, bool, error)
}

type WorkspaceUseCase struct {
	users        IWorkspaceUserRepository
	workspaces   IWorkspaceRepository
	members      IWorkspaceMemberRepository
	auditLogs    IAuditLogRepository
	transactions ITransactionManager
	now          func() time.Time
}

func NewWorkspaceUseCase(
	users IWorkspaceUserRepository,
	workspaces IWorkspaceRepository,
	members IWorkspaceMemberRepository,
	auditLogs IAuditLogRepository,
	transactions ITransactionManager,
) *WorkspaceUseCase {
	return &WorkspaceUseCase{
		users:        users,
		workspaces:   workspaces,
		members:      members,
		auditLogs:    auditLogs,
		transactions: transactions,
		now:          time.Now,
	}
}

func (uc *WorkspaceUseCase) currentTime() time.Time {
	return uc.now().UTC()
}

func (uc *WorkspaceUseCase) findActiveUser(ctx context.Context, userID uuid.UUID) (*entity.User, error) {
	user, found, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrAccountUnavailable
	}
	if user.IsDeleted() {
		return nil, ErrAccountDeleted
	}
	if user.IsSuspended() {
		return nil, ErrAccountSuspended
	}
	if !user.CanAuthenticate() {
		return nil, ErrAccountUnavailable
	}
	return user, nil
}
