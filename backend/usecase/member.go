package usecase

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
	"github.com/koezuka404/notehub/repository"
	"gorm.io/gorm"
	"net/mail"
	"time"
)

const deletedName = "削除済みユーザー"

type IMemberUsecase interface {
	SearchUser(ctx context.Context, input SearchUserInput) (*SearchUserOutput, error)
	ListMembers(ctx context.Context, input ListMembersInput) ([]MemberListItem, error)
	AddMember(ctx context.Context, input AddMemberInput) (*AddMemberOutput, error)
	RemoveMember(ctx context.Context, input RemoveMemberInput) error
}

type IAccessCheck interface {
	CheckWorkspaceAccess(ctx context.Context, input CheckWorkspaceAccessInput) (*WorkspaceAccessResult, error)
}

type MemberUseCase struct {
	users        repository.UserRepository
	members      repository.WorkspaceMemberRepository
	auditLogs    repository.AuditLogRepository
	transactions repository.TransactionManager
	access       IAccessCheck
	now          func() time.Time
}

func NewMemberUseCase(
	users repository.UserRepository,
	members repository.WorkspaceMemberRepository,
	auditLogs repository.AuditLogRepository,
	transactions repository.TransactionManager,
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

// member_helpers.go

func validEmail(raw string) (string, error) {
	email := normalizeEmail(raw)
	if email == "" || len(email) > 255 {
		return "", ErrValidation
	}
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return "", ErrValidation
	}
	return email, nil
}

func listUser(u *entity.User) (name, email, status string) {
	if u.IsDeleted() {
		return deletedName, "", string(entity.UserStatusDeleted)
	}
	return u.Name, u.Email, string(u.Status)
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

// member_add.go

type AddMemberInput struct {
	UserID       uuid.UUID
	WorkspaceID  uuid.UUID
	TargetUserID uuid.UUID
	IPAddress    string
}

type AddMemberOutput struct {
	UserID   uuid.UUID
	Name     string
	Email    string
	Role     string
	JoinedAt string
}

func (uc *MemberUseCase) AddMember(ctx context.Context, input AddMemberInput) (*AddMemberOutput, error) {
	if err := uc.requireHost(ctx, input.UserID, input.WorkspaceID); err != nil {
		return nil, err
	}

	if input.TargetUserID == input.UserID {
		return nil, ErrCannotAddSelf
	}

	target, found, err := uc.users.FindByID(ctx, input.TargetUserID)
	if err != nil {
		return nil, fmt.Errorf("find target user: %w", err)
	}
	if !found {
		return nil, ErrTargetUserNotFound
	}
	if !target.CanAuthenticate() {
		return nil, ErrTargetAccountUnavailable
	}

	exists, err := uc.members.Exists(ctx, input.WorkspaceID, input.TargetUserID)
	if err != nil {
		return nil, fmt.Errorf("check workspace member: %w", err)
	}
	if exists {
		return nil, ErrMemberAlreadyExists
	}

	now := uc.currentTime()
	member, err := entity.NewWorkspaceMember(input.WorkspaceID, input.TargetUserID, entity.WorkspaceRoleMember, now)
	if err != nil {
		return nil, fmt.Errorf("create workspace member entity: %w", err)
	}

	var output *AddMemberOutput
	if err := uc.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.members.Create(txCtx, &member); err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return ErrMemberAlreadyExists
			}
			return fmt.Errorf("save workspace member: %w", err)
		}

		audit, err := entity.NewAuditLog(&input.UserID, "MEMBER_ADDED", "workspace_member", &member.ID, nil, now)
		if err != nil {
			return fmt.Errorf("create audit log entity: %w", err)
		}
		audit.IPAddress = input.IPAddress
		if err := uc.auditLogs.Create(txCtx, &audit); err != nil {
			return fmt.Errorf("save audit log: %w", err)
		}

		output = &AddMemberOutput{
			UserID:   target.ID,
			Name:     target.Name,
			Email:    target.Email,
			Role:     string(entity.WorkspaceRoleMember),
			JoinedAt: member.JoinedAt.Format(timeFormat),
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return output, nil
}

// member_list.go

type ListMembersInput struct {
	UserID      uuid.UUID
	WorkspaceID uuid.UUID
}

type MemberListItem struct {
	UserID   uuid.UUID
	Name     string
	Email    string
	Status   string
	Role     string
	JoinedAt string
}

func (uc *MemberUseCase) ListMembers(ctx context.Context, input ListMembersInput) ([]MemberListItem, error) {
	if err := uc.requireMember(ctx, input.UserID, input.WorkspaceID); err != nil {
		return nil, err
	}

	members, err := uc.members.FindByWorkspaceID(ctx, input.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("list workspace members: %w", err)
	}

	items := make([]MemberListItem, 0, len(members))
	for _, member := range members {
		user, found, err := uc.users.FindByID(ctx, member.UserID)
		if err != nil {
			return nil, fmt.Errorf("find member user: %w", err)
		}
		name, email, status := deletedName, "", string(entity.UserStatusDeleted)
		if found {
			name, email, status = listUser(user)
		}
		items = append(items, MemberListItem{
			UserID:   member.UserID,
			Name:     name,
			Email:    email,
			Status:   status,
			Role:     string(member.Role),
			JoinedAt: member.JoinedAt.Format(timeFormat),
		})
	}
	return items, nil
}

// member_remove.go

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

// member_search.go

type SearchUserInput struct {
	UserID      uuid.UUID
	WorkspaceID uuid.UUID
	Email       string
}

type SearchUserOutput struct {
	ID     uuid.UUID
	Email  string
	Name   string
	Status entity.UserStatus
}

func (uc *MemberUseCase) SearchUser(ctx context.Context, input SearchUserInput) (*SearchUserOutput, error) {
	if err := uc.requireHost(ctx, input.UserID, input.WorkspaceID); err != nil {
		return nil, err
	}

	email, err := validEmail(input.Email)
	if err != nil {
		return nil, err
	}

	user, found, err := uc.users.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if !found || !user.CanAuthenticate() {
		return nil, ErrUserNotFound
	}
	if user.ID == input.UserID {
		return nil, ErrUserNotFound
	}

	exists, err := uc.members.Exists(ctx, input.WorkspaceID, user.ID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUserNotFound
	}

	return &SearchUserOutput{
		ID:     user.ID,
		Email:  user.Email,
		Name:   user.Name,
		Status: user.Status,
	}, nil
}
