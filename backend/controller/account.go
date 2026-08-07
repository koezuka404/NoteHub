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
		return err
	}
	workspaceID, err := parseWorkspaceIDParam(e)
	if err != nil {
		return err
	}
	targetID, err := parseUserIDParam(e)
	if err != nil {
		return err
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

func handleAccountUseCaseError(e echo.Context, err error) error {
	switch {
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
