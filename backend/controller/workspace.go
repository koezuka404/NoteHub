package controller

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/dto"
	appmiddleware "github.com/koezuka404/notehub/middleware"
	"github.com/koezuka404/notehub/usecase"
	"github.com/labstack/echo/v4"
)

type WorkspaceController struct {
	workspace usecase.IWorkspaceUsecase
}

func NewWorkspaceController(workspace usecase.IWorkspaceUsecase) *WorkspaceController {
	return &WorkspaceController{workspace: workspace}
}

func (c *WorkspaceController) Create(ctx echo.Context) error {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		return err
	}
	var req dto.CreateWorkspaceRequest
	if err := ctx.Bind(&req); err != nil {
		return writeWorkspaceError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "リクエスト形式が不正です")
	}
	out, err := c.workspace.CreateWorkspace(ctx.Request().Context(), usecase.CreateWorkspaceInput{
		UserID:    userID,
		Name:      req.Name,
		IPAddress: ctx.RealIP(),
	})
	if err != nil {
		return handleWorkspaceUseCaseError(ctx, err)
	}
	return ctx.JSON(http.StatusCreated, dto.Response{Data: dto.CreateWorkspaceResponse{
		ID:        out.ID.String(),
		Name:      out.Name,
		HostID:    out.HostID.String(),
		CreatedAt: out.CreatedAt,
	}})
}

func (c *WorkspaceController) List(ctx echo.Context) error {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		return err
	}
	items, err := c.workspace.ListWorkspaces(ctx.Request().Context(), usecase.ListWorkspacesInput{UserID: userID})
	if err != nil {
		return handleWorkspaceUseCaseError(ctx, err)
	}
	resp := make([]dto.WorkspaceListItemResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, dto.WorkspaceListItemResponse{
			ID:                item.ID.String(),
			Name:              item.Name,
			HostID:            item.HostID.String(),
			HostName:          item.HostName,
			Role:              item.Role,
			IsAvailable:       item.IsAvailable,
			UnavailableReason: item.UnavailableReason,
			UpdatedAt:         item.UpdatedAt,
		})
	}
	return ctx.JSON(http.StatusOK, dto.Response{Data: resp})
}

func (c *WorkspaceController) Get(ctx echo.Context) error {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		return err
	}
	workspaceID, err := parseWorkspaceIDParam(ctx)
	if err != nil {
		return err
	}
	out, err := c.workspace.GetWorkspace(ctx.Request().Context(), usecase.GetWorkspaceInput{
		UserID:      userID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return handleWorkspaceUseCaseError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, dto.Response{Data: dto.GetWorkspaceResponse{
		ID:   out.ID.String(),
		Name: out.Name,
		Host: dto.WorkspaceHostResponse{
			ID:     out.Host.ID.String(),
			Name:   out.Host.Name,
			Status: out.Host.Status,
		},
		Role:      out.Role,
		CreatedAt: out.CreatedAt,
		UpdatedAt: out.UpdatedAt,
	}})
}

func (c *WorkspaceController) Update(ctx echo.Context) error {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		return err
	}
	workspaceID, err := parseWorkspaceIDParam(ctx)
	if err != nil {
		return err
	}
	var req dto.UpdateWorkspaceRequest
	if err := ctx.Bind(&req); err != nil {
		return writeWorkspaceError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "リクエスト形式が不正です")
	}
	out, err := c.workspace.UpdateWorkspace(ctx.Request().Context(), usecase.UpdateWorkspaceInput{
		UserID:      userID,
		WorkspaceID: workspaceID,
		Name:        req.Name,
		IPAddress:   ctx.RealIP(),
	})
	if err != nil {
		return handleWorkspaceUseCaseError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, dto.Response{Data: dto.UpdateWorkspaceResponse{
		ID:        out.ID.String(),
		Name:      out.Name,
		UpdatedAt: out.UpdatedAt,
	}})
}

func (c *WorkspaceController) Delete(ctx echo.Context) error {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		return err
	}
	workspaceID, err := parseWorkspaceIDParam(ctx)
	if err != nil {
		return err
	}
	var req dto.DeleteWorkspaceRequest
	if err := ctx.Bind(&req); err != nil {
		return writeWorkspaceError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "リクエスト形式が不正です")
	}
	out, err := c.workspace.DeleteWorkspace(ctx.Request().Context(), usecase.DeleteWorkspaceInput{
		UserID:      userID,
		WorkspaceID: workspaceID,
		Reason:      req.Reason,
		IPAddress:   ctx.RealIP(),
	})
	if err != nil {
		return handleWorkspaceUseCaseError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, dto.Response{Data: dto.DeleteWorkspaceResponse{
		WorkspaceID: out.WorkspaceID.String(),
		DeletedAt:   out.DeletedAt,
	}})
}

func authenticatedUserID(ctx echo.Context) (uuid.UUID, error) {
	userID, ok := ctx.Get(appmiddleware.ContextUserID).(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return uuid.Nil, writeWorkspaceError(ctx, http.StatusUnauthorized, "ACCESS_TOKEN_INVALID", "アクセストークンが不正です")
	}
	return userID, nil
}

func parseWorkspaceIDParam(ctx echo.Context) (uuid.UUID, error) {
	raw := ctx.Param("workspaceId")
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, writeWorkspaceError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "ワークスペースIDが不正です")
	}
	return id, nil
}

func handleWorkspaceUseCaseError(ctx echo.Context, err error) error {
	switch {
	case errors.Is(err, usecase.ErrValidation):
		return writeWorkspaceError(ctx, http.StatusBadRequest, "VALIDATION_ERROR", "入力値が不正です")
	case errors.Is(err, usecase.ErrAccountUnavailable):
		return writeWorkspaceError(ctx, http.StatusForbidden, "ACCOUNT_UNAVAILABLE", "この操作を実行できません")
	case errors.Is(err, usecase.ErrAccountSuspended):
		return writeWorkspaceError(ctx, http.StatusForbidden, "ACCOUNT_UNAVAILABLE", "この操作を実行できません")
	case errors.Is(err, usecase.ErrAccountDeleted):
		return writeWorkspaceError(ctx, http.StatusForbidden, "ACCOUNT_UNAVAILABLE", "この操作を実行できません")
	case errors.Is(err, usecase.ErrWorkspaceNotFound):
		return writeWorkspaceError(ctx, http.StatusNotFound, "WORKSPACE_NOT_FOUND", "ワークスペースが見つかりません")
	case errors.Is(err, usecase.ErrWorkspaceAlreadyDeleted):
		return writeWorkspaceError(ctx, http.StatusConflict, "WORKSPACE_ALREADY_DELETED", "ワークスペースは既に削除されています")
	case errors.Is(err, usecase.ErrWorkspaceAccessDenied):
		return writeWorkspaceError(ctx, http.StatusForbidden, "WORKSPACE_ACCESS_DENIED", "このワークスペースへアクセスできません")
	case errors.Is(err, usecase.ErrWorkspacePermissionDenied):
		return writeWorkspaceError(ctx, http.StatusForbidden, "WORKSPACE_PERMISSION_DENIED", "この操作を実行する権限がありません")
	case errors.Is(err, usecase.ErrHostPermissionRequired):
		return writeWorkspaceError(ctx, http.StatusForbidden, "HOST_PERMISSION_REQUIRED", "ホスト権限が必要です")
	case errors.Is(err, usecase.ErrWorkspaceHostSuspended):
		return writeWorkspaceError(ctx, http.StatusLocked, "WORKSPACE_HOST_SUSPENDED", "このワークスペースは現在利用できません")
	case errors.Is(err, usecase.ErrWorkspaceHostDeleted):
		return writeWorkspaceError(ctx, http.StatusLocked, "WORKSPACE_HOST_DELETED", "このワークスペースは現在利用できません")
	default:
		return writeWorkspaceError(ctx, http.StatusInternalServerError, "INTERNAL_ERROR", "内部エラーが発生しました")
	}
}

func writeWorkspaceError(ctx echo.Context, status int, code, message string) error {
	return ctx.JSON(status, dto.ErrorResponse{Error: dto.ErrorBody{Code: code, Message: message}})
}
