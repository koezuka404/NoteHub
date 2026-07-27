package controller

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/dto"
	"github.com/koezuka404/notehub/entity"
	"github.com/koezuka404/notehub/usecase"
	"github.com/labstack/echo/v4"
)

type VersionController struct {
	version usecase.IVersionUsecase
}

func NewVersionController(version usecase.IVersionUsecase) *VersionController {
	return &VersionController{version: version}
}

func (c *VersionController) List(ctx echo.Context) error {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		return err
	}
	documentID, err := parseDocumentIDParam(ctx)
	if err != nil {
		return err
	}
	items, err := c.version.ListVersions(ctx.Request().Context(), usecase.ListVersionsInput{
		UserID: userID, DocumentID: documentID,
	})
	if err != nil {
		return handleVersionUseCaseError(ctx, err)
	}
	resp := make([]dto.VersionListItemResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, dto.VersionListItemResponse{
			ID: item.ID.String(), Type: item.Type,
			CreatedBy: item.CreatedBy.String(), CreatedAt: item.CreatedAt,
		})
	}
	return ctx.JSON(http.StatusOK, dto.Response{Data: resp})
}

func (c *VersionController) Get(ctx echo.Context) error {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		return err
	}
	documentID, err := parseDocumentIDParam(ctx)
	if err != nil {
		return err
	}
	versionID, err := parseVersionIDParam(ctx)
	if err != nil {
		return err
	}
	out, err := c.version.GetVersion(ctx.Request().Context(), usecase.GetVersionInput{
		UserID: userID, DocumentID: documentID, VersionID: versionID,
	})
	if err != nil {
		return handleVersionUseCaseError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, dto.Response{Data: dto.GetVersionResponse{
		ID: out.ID.String(), DocumentID: out.DocumentID.String(), Content: out.Content,
		Type: out.Type, CreatedBy: out.CreatedBy.String(), CreatedAt: out.CreatedAt,
	}})
}

func (c *VersionController) Restore(ctx echo.Context) error {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		return err
	}
	documentID, err := parseDocumentIDParam(ctx)
	if err != nil {
		return err
	}
	versionID, err := parseVersionIDParam(ctx)
	if err != nil {
		return err
	}
	out, err := c.version.RestoreVersion(ctx.Request().Context(), usecase.RestoreVersionInput{
		UserID: userID, DocumentID: documentID, VersionID: versionID,
	})
	if err != nil {
		return handleVersionUseCaseError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, dto.Response{Data: dto.RestoreVersionResponse{
		DocumentID: out.DocumentID.String(),
		VersionID:  out.VersionID.String(),
		RestoredAt: out.RestoredAt,
	}})
}

func parseVersionIDParam(ctx echo.Context) (uuid.UUID, error) {
	raw := ctx.Param("versionId")
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, writeWorkspaceError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "バージョンIDが不正です")
	}
	return id, nil
}

func handleVersionUseCaseError(ctx echo.Context, err error) error {
	switch {
	case errors.Is(err, usecase.ErrVersionNotFound):
		return writeWorkspaceError(ctx, http.StatusNotFound, "VERSION_NOT_FOUND", "編集履歴が見つかりません")
	case errors.Is(err, entity.ErrDocumentConflict):
		return writeWorkspaceError(ctx, http.StatusConflict, "DOCUMENT_CONFLICT", "ドキュメントが更新されています")
	default:
		return handleDocumentUseCaseError(ctx, err)
	}
}
