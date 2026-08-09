package controller

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/dto"
	"github.com/koezuka404/notehub/usecase"
	"github.com/labstack/echo/v4"
)

type MemberController struct {
	member usecase.IMemberUsecase
}

func NewMemberController(member usecase.IMemberUsecase) *MemberController {
	return &MemberController{member: member}
}

func (c *MemberController) Search(e echo.Context) error {
	userID, err := authenticatedUserID(e)
	if err != nil {
		return nil
	}
	workspaceID, err := parseWorkspaceIDParam(e)
	if err != nil {
		return nil
	}
	email := e.QueryParam("email")
	ctx := e.Request().Context()
	out, err := c.member.SearchUser(ctx, usecase.SearchUserInput{
		UserID: userID, WorkspaceID: workspaceID, Email: email,
	})
	if err != nil {
		return handleMemberUseCaseError(e, err)
	}
	return e.JSON(http.StatusOK, dto.Response{Data: dto.SearchUserResponse{
		ID: out.ID.String(), Email: out.Email, Name: out.Name, Status: string(out.Status),
	}})
}

func (c *MemberController) List(e echo.Context) error {
	userID, err := authenticatedUserID(e)
	if err != nil {
		return nil
	}
	workspaceID, err := parseWorkspaceIDParam(e)
	if err != nil {
		return nil
	}
	ctx := e.Request().Context()
	items, err := c.member.ListMembers(ctx, usecase.ListMembersInput{
		UserID: userID, WorkspaceID: workspaceID,
	})
	if err != nil {
		return handleMemberUseCaseError(e, err)
	}
	resp := make([]dto.MemberResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, dto.MemberResponse{
			UserID: item.UserID.String(), Name: item.Name, Email: item.Email,
			Status: item.Status, Role: item.Role, JoinedAt: item.JoinedAt,
		})
	}
	return e.JSON(http.StatusOK, dto.Response{Data: resp})
}

func (c *MemberController) Add(e echo.Context) error {
	userID, err := authenticatedUserID(e)
	if err != nil {
		return nil
	}
	workspaceID, err := parseWorkspaceIDParam(e)
	if err != nil {
		return nil
	}
	var req dto.AddMemberRequest
	if err := e.Bind(&req); err != nil {
		return writeWorkspaceError(e, http.StatusBadRequest, "INVALID_REQUEST", "リクエスト形式が不正です")
	}
	targetID, err := uuid.Parse(req.UserID)
	if err != nil {
		return writeWorkspaceError(e, http.StatusBadRequest, "INVALID_REQUEST", "ユーザーIDが不正です")
	}
	ctx := e.Request().Context()
	out, err := c.member.AddMember(ctx, usecase.AddMemberInput{
		UserID: userID, WorkspaceID: workspaceID, TargetUserID: targetID, IPAddress: e.RealIP(),
	})
	if err != nil {
		return handleMemberUseCaseError(e, err)
	}
	return e.JSON(http.StatusCreated, dto.Response{Data: dto.AddMemberResponse{
		UserID: out.UserID.String(), Name: out.Name, Email: out.Email,
		Role: out.Role, JoinedAt: out.JoinedAt,
	}})
}

func (c *MemberController) Remove(e echo.Context) error {
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
	if err := c.member.RemoveMember(ctx, usecase.RemoveMemberInput{
		UserID: userID, WorkspaceID: workspaceID, TargetUserID: targetID, IPAddress: e.RealIP(),
	}); err != nil {
		return handleMemberUseCaseError(e, err)
	}
	return e.JSON(http.StatusOK, dto.Response{Data: struct{}{}})
}

func parseUserIDParam(e echo.Context) (uuid.UUID, error) {
	raw := e.Param("userId")
	id, err := uuid.Parse(raw)
	if err != nil {
		_ = writeWorkspaceError(e, http.StatusBadRequest, "INVALID_REQUEST", "ユーザーIDが不正です")
		return uuid.Nil, errResponseSent
	}
	return id, nil
}

func handleMemberUseCaseError(e echo.Context, err error) error {
	switch {
	case errors.Is(err, usecase.ErrValidation):
		return writeWorkspaceError(e, http.StatusBadRequest, "VALIDATION_ERROR", "入力値が不正です")
	case errors.Is(err, usecase.ErrUserNotFound):
		return writeWorkspaceError(e, http.StatusNotFound, "USER_NOT_FOUND", "ユーザーが見つかりません")
	case errors.Is(err, usecase.ErrTargetUserNotFound):
		return writeWorkspaceError(e, http.StatusNotFound, "TARGET_USER_NOT_FOUND", "ユーザーが見つかりません")
	case errors.Is(err, usecase.ErrTargetAccountUnavailable):
		return writeWorkspaceError(e, http.StatusConflict, "TARGET_ACCOUNT_UNAVAILABLE", "このユーザーを追加できません")
	case errors.Is(err, usecase.ErrCannotAddSelf):
		return writeWorkspaceError(e, http.StatusConflict, "CANNOT_ADD_SELF", "自分自身を追加できません")
	case errors.Is(err, usecase.ErrMemberAlreadyExists):
		return writeWorkspaceError(e, http.StatusConflict, "MEMBER_ALREADY_EXISTS", "既に参加しています")
	case errors.Is(err, usecase.ErrMemberNotFound):
		return writeWorkspaceError(e, http.StatusNotFound, "MEMBER_NOT_FOUND", "メンバーが見つかりません")
	case errors.Is(err, usecase.ErrCannotRemoveHost):
		return writeWorkspaceError(e, http.StatusConflict, "CANNOT_REMOVE_HOST", "ホストを削除できません")
	default:
		return handleWorkspaceUseCaseError(e, err)
	}
}
