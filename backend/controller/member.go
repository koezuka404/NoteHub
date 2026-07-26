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

func (c *MemberController) Search(ctx echo.Context) error {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		return err
	}
	workspaceID, err := parseWorkspaceIDParam(ctx)
	if err != nil {
		return err
	}
	email := ctx.QueryParam("email")
	out, err := c.member.SearchUser(ctx.Request().Context(), usecase.SearchUserInput{
		UserID: userID, WorkspaceID: workspaceID, Email: email,
	})
	if err != nil {
		return handleMemberUseCaseError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, dto.Response{Data: dto.SearchUserResponse{
		ID: out.ID.String(), Email: out.Email, Name: out.Name, Status: string(out.Status),
	}})
}

func (c *MemberController) List(ctx echo.Context) error {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		return err
	}
	workspaceID, err := parseWorkspaceIDParam(ctx)
	if err != nil {
		return err
	}
	items, err := c.member.ListMembers(ctx.Request().Context(), usecase.ListMembersInput{
		UserID: userID, WorkspaceID: workspaceID,
	})
	if err != nil {
		return handleMemberUseCaseError(ctx, err)
	}
	resp := make([]dto.MemberResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, dto.MemberResponse{
			UserID: item.UserID.String(), Name: item.Name, Email: item.Email,
			Status: item.Status, Role: item.Role, JoinedAt: item.JoinedAt,
		})
	}
	return ctx.JSON(http.StatusOK, dto.Response{Data: resp})
}

func (c *MemberController) Add(ctx echo.Context) error {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		return err
	}
	workspaceID, err := parseWorkspaceIDParam(ctx)
	if err != nil {
		return err
	}
	var req dto.AddMemberRequest
	if err := ctx.Bind(&req); err != nil {
		return writeWorkspaceError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "リクエスト形式が不正です")
	}
	targetID, err := uuid.Parse(req.UserID)
	if err != nil {
		return writeWorkspaceError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "ユーザーIDが不正です")
	}
	out, err := c.member.AddMember(ctx.Request().Context(), usecase.AddMemberInput{
		UserID: userID, WorkspaceID: workspaceID, TargetUserID: targetID, IPAddress: ctx.RealIP(),
	})
	if err != nil {
		return handleMemberUseCaseError(ctx, err)
	}
	return ctx.JSON(http.StatusCreated, dto.Response{Data: dto.AddMemberResponse{
		UserID: out.UserID.String(), Name: out.Name, Email: out.Email,
		Role: out.Role, JoinedAt: out.JoinedAt,
	}})
}

func (c *MemberController) Remove(ctx echo.Context) error {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		return err
	}
	workspaceID, err := parseWorkspaceIDParam(ctx)
	if err != nil {
		return err
	}
	targetID, err := parseUserIDParam(ctx)
	if err != nil {
		return err
	}
	if err := c.member.RemoveMember(ctx.Request().Context(), usecase.RemoveMemberInput{
		UserID: userID, WorkspaceID: workspaceID, TargetUserID: targetID, IPAddress: ctx.RealIP(),
	}); err != nil {
		return handleMemberUseCaseError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, dto.Response{Data: struct{}{}})
}

func parseUserIDParam(ctx echo.Context) (uuid.UUID, error) {
	raw := ctx.Param("userId")
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, writeWorkspaceError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "ユーザーIDが不正です")
	}
	return id, nil
}

func handleMemberUseCaseError(ctx echo.Context, err error) error {
	switch {
	case errors.Is(err, usecase.ErrValidation):
		return writeWorkspaceError(ctx, http.StatusBadRequest, "VALIDATION_ERROR", "入力値が不正です")
	case errors.Is(err, usecase.ErrUserNotFound):
		return writeWorkspaceError(ctx, http.StatusNotFound, "USER_NOT_FOUND", "ユーザーが見つかりません")
	case errors.Is(err, usecase.ErrTargetUserNotFound):
		return writeWorkspaceError(ctx, http.StatusNotFound, "TARGET_USER_NOT_FOUND", "ユーザーが見つかりません")
	case errors.Is(err, usecase.ErrTargetAccountUnavailable):
		return writeWorkspaceError(ctx, http.StatusConflict, "TARGET_ACCOUNT_UNAVAILABLE", "このユーザーを追加できません")
	case errors.Is(err, usecase.ErrCannotAddSelf):
		return writeWorkspaceError(ctx, http.StatusConflict, "CANNOT_ADD_SELF", "自分自身を追加できません")
	case errors.Is(err, usecase.ErrMemberAlreadyExists):
		return writeWorkspaceError(ctx, http.StatusConflict, "MEMBER_ALREADY_EXISTS", "既に参加しています")
	case errors.Is(err, usecase.ErrMemberNotFound):
		return writeWorkspaceError(ctx, http.StatusNotFound, "MEMBER_NOT_FOUND", "メンバーが見つかりません")
	case errors.Is(err, usecase.ErrCannotRemoveHost):
		return writeWorkspaceError(ctx, http.StatusConflict, "CANNOT_REMOVE_HOST", "ホストを削除できません")
	default:
		return handleWorkspaceUseCaseError(ctx, err)
	}
}
