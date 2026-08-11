package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
	"github.com/koezuka404/notehub/repository"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxWorkspaceNameLength   = 100
	minDeleteReasonLength    = 1
	maxDeleteReasonLength    = 500
	unavailableReasonNone    = ""
	unavailableHostSuspended = "HOST_SUSPENDED"
	unavailableHostDeleted   = "HOST_DELETED"
)

type IWorkspaceUsecase interface {
	CreateWorkspace(ctx context.Context, input CreateWorkspaceInput) (*CreateWorkspaceOutput, error)
	ListWorkspaces(ctx context.Context, input ListWorkspacesInput) ([]WorkspaceListItem, error)
	GetWorkspace(ctx context.Context, input GetWorkspaceInput) (*GetWorkspaceOutput, error)
	UpdateWorkspace(ctx context.Context, input UpdateWorkspaceInput) (*UpdateWorkspaceOutput, error)
	DeleteWorkspace(ctx context.Context, input DeleteWorkspaceInput) (*DeleteWorkspaceOutput, error)
}

type WorkspaceUseCase struct {
	users        repository.UserRepository
	workspaces   repository.WorkspaceRepository
	members      repository.WorkspaceMemberRepository
	auditLogs    repository.AuditLogRepository
	transactions repository.TransactionManager
	flush        IDocumentFlushService
	notifier     IDocumentWebSocketNotifier
	now          func() time.Time
}

func NewWorkspaceUseCase(
	users repository.UserRepository,
	workspaces repository.WorkspaceRepository,
	members repository.WorkspaceMemberRepository,
	auditLogs repository.AuditLogRepository,
	transactions repository.TransactionManager,
	flush IDocumentFlushService,
	notifier IDocumentWebSocketNotifier,
) *WorkspaceUseCase {
	return &WorkspaceUseCase{
		users:        users,
		workspaces:   workspaces,
		members:      members,
		auditLogs:    auditLogs,
		transactions: transactions,
		flush:        flush,
		notifier:     notifier,
		now:          time.Now,
	}
}


func normalizeWorkspaceName(name string) string {
	return strings.TrimSpace(name)
}

func validateWorkspaceName(name string) error {
	name = normalizeWorkspaceName(name)
	if name == "" {
		return ErrValidation
	}
	if utf8.RuneCountInString(name) > maxWorkspaceNameLength {
		return ErrValidation
	}
	return nil
}

func validateDeleteReason(reason string) error {
	reason = strings.TrimSpace(reason)
	length := utf8.RuneCountInString(reason)
	if length < minDeleteReasonLength || length > maxDeleteReasonLength {
		return ErrValidation
	}
	return nil
}

type auditDeleteMetadata struct {
	Reason     string `json:"reason"`
	DeleteType string `json:"delete_type"`
}

func workspaceDeleteMetadata(reason string) (json.RawMessage, error) {
	raw, err := jsonMarshalFn(auditDeleteMetadata{Reason: reason, DeleteType: "logical"})
	if err != nil {
		return nil, fmt.Errorf("marshal delete metadata: %w", err)
	}
	return raw, nil
}

func workspaceAvailability(host *entity.User) (bool, string) {
	if host.CanAuthenticate() {
		return true, unavailableReasonNone
	}
	if host.IsSuspended() {
		return false, unavailableHostSuspended
	}
	if host.IsDeleted() {
		return false, unavailableHostDeleted
	}
	return false, unavailableHostSuspended
}


func checkWorkspaceHostStatus(host *entity.User) error {
	if host.CanAuthenticate() {
		return nil
	}
	if host.IsSuspended() {
		return ErrWorkspaceHostSuspended
	}
	if host.IsDeleted() {
		return ErrWorkspaceHostDeleted
	}
	return ErrWorkspaceHostSuspended
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


type CreateWorkspaceInput struct {
	UserID    uuid.UUID
	Name      string
	IPAddress string
}

type CreateWorkspaceOutput struct {
	ID        uuid.UUID
	Name      string
	HostID    uuid.UUID
	CreatedAt string
}

func (uc *WorkspaceUseCase) CreateWorkspace(ctx context.Context, input CreateWorkspaceInput) (*CreateWorkspaceOutput, error) {
	if err := validateWorkspaceName(input.Name); err != nil {
		return nil, err
	}
	user, err := uc.findActiveUser(ctx, input.UserID)
	if err != nil {
		return nil, err
	}

	now := uc.currentTime()
	workspace, err := newWorkspaceFn(user.ID, normalizeWorkspaceName(input.Name), now)
	if err != nil {
		return nil, fmt.Errorf("create workspace entity: %w", err)
	}
	member, err := newWorkspaceMemberFn(workspace.ID, user.ID, entity.WorkspaceRoleHost, now)
	if err != nil {
		return nil, fmt.Errorf("create workspace host member: %w", err)
	}

	if err := uc.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.workspaces.Create(txCtx, &workspace); err != nil {
			return fmt.Errorf("save workspace: %w", err)
		}
		if err := uc.members.Create(txCtx, &member); err != nil {
			return fmt.Errorf("save workspace host: %w", err)
		}
		audit, err := newAuditLogFn(&input.UserID, "WORKSPACE_CREATED", "workspace", &workspace.ID, nil, now)
		if err != nil {
			return fmt.Errorf("create audit log entity: %w", err)
		}
		audit.IPAddress = input.IPAddress
		if err := uc.auditLogs.Create(txCtx, &audit); err != nil {
			return fmt.Errorf("save audit log: %w", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return &CreateWorkspaceOutput{
		ID:        workspace.ID,
		Name:      workspace.Name,
		HostID:    workspace.HostID,
		CreatedAt: workspace.CreatedAt.Format(timeFormat),
	}, nil
}


type GetWorkspaceInput struct {
	UserID      uuid.UUID
	WorkspaceID uuid.UUID
}

type WorkspaceHostOutput struct {
	ID     uuid.UUID
	Name   string
	Status string
}

type GetWorkspaceOutput struct {
	ID        uuid.UUID
	Name      string
	Host      WorkspaceHostOutput
	Role      string
	CreatedAt string
	UpdatedAt string
}

func (uc *WorkspaceUseCase) GetWorkspace(ctx context.Context, input GetWorkspaceInput) (*GetWorkspaceOutput, error) {
	access, err := uc.CheckWorkspaceAccess(ctx, CheckWorkspaceAccessInput{
		UserID:      input.UserID,
		WorkspaceID: input.WorkspaceID,
	})
	if err != nil {
		return nil, err
	}

	return &GetWorkspaceOutput{
		ID:   access.Workspace.ID,
		Name: access.Workspace.Name,
		Host: WorkspaceHostOutput{
			ID:     access.Host.ID,
			Name:   access.Host.Name,
			Status: string(access.Host.Status),
		},
		Role:      string(access.Member.Role),
		CreatedAt: access.Workspace.CreatedAt.Format(timeFormat),
		UpdatedAt: access.Workspace.UpdatedAt.Format(timeFormat),
	}, nil
}


type ListWorkspacesInput struct {
	UserID uuid.UUID
}

type WorkspaceListItem struct {
	ID                uuid.UUID
	Name              string
	HostID            uuid.UUID
	HostName          string
	Role              string
	IsAvailable       bool
	UnavailableReason string
	UpdatedAt         string
}

func (uc *WorkspaceUseCase) ListWorkspaces(ctx context.Context, input ListWorkspacesInput) ([]WorkspaceListItem, error) {
	if _, err := uc.findActiveUser(ctx, input.UserID); err != nil {
		return nil, err
	}

	workspaces, err := uc.workspaces.FindByUserID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}

	items := make([]WorkspaceListItem, 0, len(workspaces))
	for _, workspace := range workspaces {
		member, found, err := uc.members.FindByWorkspaceAndUser(ctx, workspace.ID, input.UserID)
		if err != nil {
			return nil, fmt.Errorf("find workspace member: %w", err)
		}
		if !found {
			continue
		}

		host, found, err := uc.users.FindByID(ctx, workspace.HostID)
		if err != nil {
			return nil, fmt.Errorf("find workspace host: %w", err)
		}
		hostName := ""
		isAvailable := false
		unavailableReason := unavailableHostDeleted
		if found {
			hostName = host.Name
			isAvailable, unavailableReason = workspaceAvailability(host)
		}

		items = append(items, WorkspaceListItem{
			ID:                workspace.ID,
			Name:              workspace.Name,
			HostID:            workspace.HostID,
			HostName:          hostName,
			Role:              string(member.Role),
			IsAvailable:       isAvailable,
			UnavailableReason: unavailableReason,
			UpdatedAt:         workspace.UpdatedAt.Format(timeFormat),
		})
	}
	return items, nil
}


type UpdateWorkspaceInput struct {
	UserID      uuid.UUID
	WorkspaceID uuid.UUID
	Name        string
	IPAddress   string
}

type UpdateWorkspaceOutput struct {
	ID        uuid.UUID
	Name      string
	UpdatedAt string
}

func (uc *WorkspaceUseCase) UpdateWorkspace(ctx context.Context, input UpdateWorkspaceInput) (*UpdateWorkspaceOutput, error) {
	if err := validateWorkspaceName(input.Name); err != nil {
		return nil, err
	}

	hostRole := entity.WorkspaceRoleHost
	if _, err := uc.CheckWorkspaceAccess(ctx, CheckWorkspaceAccessInput{
		UserID:       input.UserID,
		WorkspaceID:  input.WorkspaceID,
		RequiredRole: &hostRole,
	}); err != nil {
		return nil, err
	}

	now := uc.currentTime()
	var output *UpdateWorkspaceOutput

	if err := uc.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		workspace, found, err := uc.workspaces.FindByIDForUpdate(txCtx, input.WorkspaceID)
		if err != nil {
			return fmt.Errorf("find workspace for update: %w", err)
		}
		if !found {
			return ErrWorkspaceNotFound
		}
		if workspace.IsDeleted() {
			return ErrWorkspaceAlreadyDeleted
		}
		if err := renameWorkspaceFn(workspace, normalizeWorkspaceName(input.Name), now); err != nil {
			return err
		}
		if err := uc.workspaces.Update(txCtx, workspace); err != nil {
			return fmt.Errorf("update workspace: %w", err)
		}

		audit, err := newAuditLogFn(&input.UserID, "WORKSPACE_UPDATED", "workspace", &workspace.ID, nil, now)
		if err != nil {
			return fmt.Errorf("create audit log entity: %w", err)
		}
		audit.IPAddress = input.IPAddress
		if err := uc.auditLogs.Create(txCtx, &audit); err != nil {
			return fmt.Errorf("save audit log: %w", err)
		}

		output = &UpdateWorkspaceOutput{
			ID:        workspace.ID,
			Name:      workspace.Name,
			UpdatedAt: workspace.UpdatedAt.Format(timeFormat),
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return output, nil
}


type DeleteWorkspaceInput struct {
	UserID      uuid.UUID
	WorkspaceID uuid.UUID
	Reason      string
	IPAddress   string
}

type DeleteWorkspaceOutput struct {
	WorkspaceID uuid.UUID
	DeletedAt   string
}

func (uc *WorkspaceUseCase) DeleteWorkspace(ctx context.Context, input DeleteWorkspaceInput) (*DeleteWorkspaceOutput, error) {
	if err := validateDeleteReason(input.Reason); err != nil {
		return nil, err
	}

	hostRole := entity.WorkspaceRoleHost
	if _, err := uc.CheckWorkspaceAccess(ctx, CheckWorkspaceAccessInput{
		UserID:       input.UserID,
		WorkspaceID:  input.WorkspaceID,
		RequiredRole: &hostRole,
	}); err != nil {
		return nil, err
	}

	if uc.flush != nil {
		if err := uc.flush.FlushWorkspaceDocuments(ctx, input.WorkspaceID); err != nil {
			return nil, fmt.Errorf("flush workspace documents before delete: %w", err)
		}
	}

	now := uc.currentTime()
	reason := strings.TrimSpace(input.Reason)
	var output *DeleteWorkspaceOutput

	if err := uc.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		workspace, found, err := uc.workspaces.FindByIDForUpdate(txCtx, input.WorkspaceID)
		if err != nil {
			return fmt.Errorf("find workspace for delete: %w", err)
		}
		if !found {
			return ErrWorkspaceNotFound
		}
		if workspace.IsDeleted() {
			return ErrWorkspaceAlreadyDeleted
		}
		if err := logicalDeleteWorkspaceFn(workspace, input.UserID, reason, now); err != nil {
			return err
		}
		if err := uc.workspaces.Update(txCtx, workspace); err != nil {
			return fmt.Errorf("update deleted workspace: %w", err)
		}

		metadata, err := workspaceDeleteMetadata(reason)
		if err != nil {
			return err
		}
		audit, err := newAuditLogFn(&input.UserID, "WORKSPACE_DELETED", "workspace", &workspace.ID, metadata, now)
		if err != nil {
			return fmt.Errorf("create audit log entity: %w", err)
		}
		audit.IPAddress = input.IPAddress
		if err := uc.auditLogs.Create(txCtx, &audit); err != nil {
			return fmt.Errorf("save audit log: %w", err)
		}

		output = &DeleteWorkspaceOutput{
			WorkspaceID: workspace.ID,
			DeletedAt:   workspace.DeletedAt.Format(timeFormat),
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if uc.notifier != nil {
		if err := uc.notifier.NotifyWorkspaceDeleted(output.WorkspaceID, input.UserID, output.DeletedAt); err != nil {
			return nil, fmt.Errorf("notify workspace deleted: %w", err)
		}
	}
	return output, nil
}


type CheckWorkspaceAccessInput struct {
	UserID       uuid.UUID
	WorkspaceID  uuid.UUID
	RequiredRole *entity.WorkspaceRole
}

type WorkspaceAccessResult struct {
	Workspace *entity.Workspace
	Member    *entity.WorkspaceMember
	Host      *entity.User
}

func (uc *WorkspaceUseCase) CheckWorkspaceAccess(ctx context.Context, input CheckWorkspaceAccessInput) (*WorkspaceAccessResult, error) {
	if _, err := uc.findActiveUser(ctx, input.UserID); err != nil {
		return nil, err
	}

	workspace, found, err := uc.workspaces.FindByID(ctx, input.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("find workspace: %w", err)
	}
	if !found {
		return nil, ErrWorkspaceNotFound
	}
	if workspace.IsDeleted() {
		return nil, ErrWorkspaceAlreadyDeleted
	}

	host, found, err := uc.users.FindByID(ctx, workspace.HostID)
	if err != nil {
		return nil, fmt.Errorf("find workspace host: %w", err)
	}
	if !found {
		return nil, ErrWorkspaceNotFound
	}
	if err := checkWorkspaceHostStatus(host); err != nil {
		return nil, err
	}

	member, found, err := uc.members.FindByWorkspaceAndUser(ctx, workspace.ID, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("find workspace member: %w", err)
	}
	if !found {
		return nil, ErrWorkspaceAccessDenied
	}

	if input.RequiredRole != nil && member.Role != *input.RequiredRole {
		if *input.RequiredRole == entity.WorkspaceRoleHost {
			return nil, ErrHostPermissionRequired
		}
		return nil, ErrWorkspacePermissionDenied
	}

	return &WorkspaceAccessResult{
		Workspace: workspace,
		Member:    member,
		Host:      host,
	}, nil
}
