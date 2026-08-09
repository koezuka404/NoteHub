package controller

import (
	"errors"
	"net/http"

	"github.com/koezuka404/notehub/dto"
	"github.com/koezuka404/notehub/usecase"
	"github.com/labstack/echo/v4"
)

type AccountController struct {
	account usecase.IAccountUsecase
}

func NewAccountController(account usecase.IAccountUsecase) *AccountController {
	return &AccountController{account: account}
}

func (c *AccountController) Suspend(e echo.Context) error {
	userID, err := authenticatedUserID(e)
	if err != nil {
		return nil
	}
	workspaceID, err := parseWorkspaceIDParam(e)
	if err != nil {
		return nil
	}
	targetID, err := parseUserIDParam(e)
	if err != nil {
		return nil
	}
	ctx := e.Request().Context()
	out, err := c.account.SuspendAccount(ctx, usecase.SuspendAccountInput{
		UserID: userID, WorkspaceID: workspaceID, TargetUserID: targetID, IPAddress: e.RealIP(),
	})
	if err != nil {
		return handleAccountUseCaseError(e, err)
	}
	return e.JSON(http.StatusOK, dto.Response{Data: dto.SuspendAccountResponse{
		UserID: out.UserID.String(), Status: out.Status, SuspendedAt: out.SuspendedAt,
	}})
}

func (c *AccountController) Reactivate(e echo.Context) error {
	userID, err := authenticatedUserID(e)
	if err != nil {
		return nil
	}
	workspaceID, err := parseWorkspaceIDParam(e)
	if err != nil {
		return nil
	}
	targetID, err := parseUserIDParam(e)
	if err != nil {
		return nil
	}
	ctx := e.Request().Context()
	out, err := c.account.ReactivateAccount(ctx, usecase.ReactivateAccountInput{
		UserID: userID, WorkspaceID: workspaceID, TargetUserID: targetID, IPAddress: e.RealIP(),
	})
	if err != nil {
		return handleAccountUseCaseError(e, err)
	}
	return e.JSON(http.StatusOK, dto.Response{Data: dto.ReactivateAccountResponse{
		UserID: out.UserID.String(), Status: out.Status,
	}})
}

func (c *AccountController) Delete(e echo.Context) error {
	userID, err := authenticatedUserID(e)
	if err != nil {
		return nil
	}
	workspaceID, err := parseWorkspaceIDParam(e)
	if err != nil {
		return nil
	}
	targetID, err := parseUserIDParam(e)
	if err != nil {
		return nil
	}
	ctx := e.Request().Context()
	out, err := c.account.DeleteAccount(ctx, usecase.DeleteAccountInput{
		UserID: userID, WorkspaceID: workspaceID, TargetUserID: targetID, IPAddress: e.RealIP(),
	})
	if err != nil {
		return handleAccountUseCaseError(e, err)
	}
	return e.JSON(http.StatusOK, dto.Response{Data: dto.DeleteAccountResponse{
		UserID: out.UserID.String(), Status: out.Status, DeletedAt: out.DeletedAt,
	}})
}

func handleAccountUseCaseError(e echo.Context, err error) error {
	switch {
	case errors.Is(err, usecase.ErrCannotDeleteSelf):
		return writeWorkspaceError(e, http.StatusConflict, "CANNOT_DELETE_SELF", "自分自身を削除できません")
	case errors.Is(err, usecase.ErrCannotReactivateSelf):
		return writeWorkspaceError(e, http.StatusConflict, "CANNOT_REACTIVATE_SELF", "自分自身を復帰できません")
	case errors.Is(err, usecase.ErrAccountNotSuspended):
		return writeWorkspaceError(e, http.StatusConflict, "ACCOUNT_NOT_SUSPENDED", "このアカウントは停止されていません")
	case errors.Is(err, usecase.ErrCannotSuspendSelf):
		return writeWorkspaceError(e, http.StatusConflict, "CANNOT_SUSPEND_SELF", "自分自身を停止できません")
	case errors.Is(err, usecase.ErrCannotSuspendHost):
		return writeWorkspaceError(e, http.StatusConflict, "CANNOT_SUSPEND_HOST", "ホストを停止できません")
	case errors.Is(err, usecase.ErrAccountAlreadySuspended):
		return writeWorkspaceError(e, http.StatusConflict, "ACCOUNT_ALREADY_SUSPENDED", "このアカウントは既に停止されています")
	case errors.Is(err, usecase.ErrAccountDeleted):
		return writeWorkspaceError(e, http.StatusConflict, "ACCOUNT_DELETED", "このアカウントは削除されています")
	case errors.Is(err, usecase.ErrTargetUserNotFound):
		return writeWorkspaceError(e, http.StatusNotFound, "TARGET_USER_NOT_FOUND", "ユーザーが見つかりません")
	case errors.Is(err, usecase.ErrMemberNotFound):
		return writeWorkspaceError(e, http.StatusNotFound, "MEMBER_NOT_FOUND", "メンバーが見つかりません")
	default:
		return handleWorkspaceUseCaseError(e, err)
	}
}
