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

func (c *WorkspaceController) Create(e echo.Context) error {
	userID, err := authenticatedUserID(e)
	if err != nil {
		return err
	}
	var req dto.CreateWorkspaceRequest
	if err := e.Bind(&req); err != nil {
		return writeWorkspaceError(e, http.StatusBadRequest, "INVALID_REQUEST", "リクエスト形式が不正です")
	}
	ctx := e.Request().Context()
	out, err := c.workspace.CreateWorkspace(ctx, usecase.CreateWorkspaceInput{
		UserID:    userID,
		Name:      req.Name,
		IPAddress: e.RealIP(),
	})
	if err != nil {
		return handleWorkspaceUseCaseError(e, err)
	}
	return e.JSON(http.StatusCreated, dto.Response{Data: dto.CreateWorkspaceResponse{
		ID:        out.ID.String(),
		Name:      out.Name,
		HostID:    out.HostID.String(),
		CreatedAt: out.CreatedAt,
	}})
}

func (c *WorkspaceController) List(e echo.Context) error {
	userID, err := authenticatedUserID(e)
	if err != nil {
		return err
	}
	ctx := e.Request().Context()
	items, err := c.workspace.ListWorkspaces(ctx, usecase.ListWorkspacesInput{UserID: userID})
	if err != nil {
		return handleWorkspaceUseCaseError(e, err)
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
	return e.JSON(http.StatusOK, dto.Response{Data: resp})
}

func (c *WorkspaceController) Get(e echo.Context) error {
	userID, err := authenticatedUserID(e)
	if err != nil {
		return err
	}
	workspaceID, err := parseWorkspaceIDParam(e)
	if err != nil {
		return err
	}
	ctx := e.Request().Context()
	out, err := c.workspace.GetWorkspace(ctx, usecase.GetWorkspaceInput{
		UserID:      userID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return handleWorkspaceUseCaseError(e, err)
	}
	return e.JSON(http.StatusOK, dto.Response{Data: dto.GetWorkspaceResponse{
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

func (c *WorkspaceController) Update(e echo.Context) error {
	userID, err := authenticatedUserID(e)
	if err != nil {
		return err
	}
	workspaceID, err := parseWorkspaceIDParam(e)
	if err != nil {
		return err
	}
	var req dto.UpdateWorkspaceRequest
	if err := e.Bind(&req); err != nil {
		return writeWorkspaceError(e, http.StatusBadRequest, "INVALID_REQUEST", "リクエスト形式が不正です")
	}
	ctx := e.Request().Context()
	out, err := c.workspace.UpdateWorkspace(ctx, usecase.UpdateWorkspaceInput{
		UserID:      userID,
		WorkspaceID: workspaceID,
		Name:        req.Name,
		IPAddress:   e.RealIP(),
	})
	if err != nil {
		return handleWorkspaceUseCaseError(e, err)
	}
	return e.JSON(http.StatusOK, dto.Response{Data: dto.UpdateWorkspaceResponse{
		ID:        out.ID.String(),
		Name:      out.Name,
		UpdatedAt: out.UpdatedAt,
	}})
}

func (c *WorkspaceController) Delete(e echo.Context) error {
	userID, err := authenticatedUserID(e)
	if err != nil {
		return err
	}
	workspaceID, err := parseWorkspaceIDParam(e)
	if err != nil {
		return err
	}
	var req dto.DeleteWorkspaceRequest
	if err := e.Bind(&req); err != nil {
		return writeWorkspaceError(e, http.StatusBadRequest, "INVALID_REQUEST", "リクエスト形式が不正です")
	}
	ctx := e.Request().Context()
	out, err := c.workspace.DeleteWorkspace(ctx, usecase.DeleteWorkspaceInput{
		UserID:      userID,
		WorkspaceID: workspaceID,
		Reason:      req.Reason,
		IPAddress:   e.RealIP(),
	})
	if err != nil {
		return handleWorkspaceUseCaseError(e, err)
	}
	return e.JSON(http.StatusOK, dto.Response{Data: dto.DeleteWorkspaceResponse{
		WorkspaceID: out.WorkspaceID.String(),
		DeletedAt:   out.DeletedAt,
	}})
}

func authenticatedUserID(e echo.Context) (uuid.UUID, error) {
	userID, ok := e.Get(appmiddleware.ContextUserID).(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return uuid.Nil, writeWorkspaceError(e, http.StatusUnauthorized, "ACCESS_TOKEN_INVALID", "アクセストークンが不正です")
	}
	return userID, nil
}

func parseWorkspaceIDParam(e echo.Context) (uuid.UUID, error) {
	raw := e.Param("workspaceId")
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, writeWorkspaceError(e, http.StatusBadRequest, "INVALID_REQUEST", "ワークスペースIDが不正です")
	}
	return id, nil
}

func handleWorkspaceUseCaseError(e echo.Context, err error) error {
	switch {
	case errors.Is(err, usecase.ErrValidation):
		return writeWorkspaceError(e, http.StatusBadRequest, "VALIDATION_ERROR", "入力値が不正です")
	case errors.Is(err, usecase.ErrAccountUnavailable):
		return writeWorkspaceError(e, http.StatusForbidden, "ACCOUNT_UNAVAILABLE", "この操作を実行できません")
	case errors.Is(err, usecase.ErrAccountSuspended):
		return writeWorkspaceError(e, http.StatusForbidden, "ACCOUNT_UNAVAILABLE", "この操作を実行できません")
	case errors.Is(err, usecase.ErrAccountDeleted):
		return writeWorkspaceError(e, http.StatusForbidden, "ACCOUNT_UNAVAILABLE", "この操作を実行できません")
	case errors.Is(err, usecase.ErrWorkspaceNotFound):
		return writeWorkspaceError(e, http.StatusNotFound, "WORKSPACE_NOT_FOUND", "ワークスペースが見つかりません")
	case errors.Is(err, usecase.ErrWorkspaceAlreadyDeleted):
		return writeWorkspaceError(e, http.StatusConflict, "WORKSPACE_ALREADY_DELETED", "ワークスペースは既に削除されています")
	case errors.Is(err, usecase.ErrWorkspaceAccessDenied):
		return writeWorkspaceError(e, http.StatusForbidden, "WORKSPACE_ACCESS_DENIED", "このワークスペースへアクセスできません")
	case errors.Is(err, usecase.ErrWorkspacePermissionDenied):
		return writeWorkspaceError(e, http.StatusForbidden, "WORKSPACE_PERMISSION_DENIED", "この操作を実行する権限がありません")
	case errors.Is(err, usecase.ErrHostPermissionRequired):
		return writeWorkspaceError(e, http.StatusForbidden, "HOST_PERMISSION_REQUIRED", "ホスト権限が必要です")
	case errors.Is(err, usecase.ErrWorkspaceHostSuspended):
		return writeWorkspaceError(e, http.StatusLocked, "WORKSPACE_HOST_SUSPENDED", "このワークスペースは現在利用できません")
	case errors.Is(err, usecase.ErrWorkspaceHostDeleted):
		return writeWorkspaceError(e, http.StatusLocked, "WORKSPACE_HOST_DELETED", "このワークスペースは現在利用できません")
	default:
		return writeWorkspaceError(e, http.StatusInternalServerError, "INTERNAL_ERROR", "内部エラーが発生しました")
	}
}

func writeWorkspaceError(e echo.Context, status int, code, message string) error {
	return appmiddleware.WriteError(e, status, code, message)
}
