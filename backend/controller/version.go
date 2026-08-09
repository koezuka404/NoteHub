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

func (c *VersionController) List(e echo.Context) error {
	userID, err := authenticatedUserID(e)
	if err != nil {
		return nil
	}
	documentID, err := parseDocumentIDParam(e)
	if err != nil {
		return nil
	}
	ctx := e.Request().Context()
	items, err := c.version.ListVersions(ctx, usecase.ListVersionsInput{
		UserID: userID, DocumentID: documentID,
	})
	if err != nil {
		return handleVersionUseCaseError(e, err)
	}
	resp := make([]dto.VersionListItemResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, dto.VersionListItemResponse{
			ID: item.ID.String(), Type: item.Type,
			CreatedBy: item.CreatedBy.String(), CreatedAt: item.CreatedAt,
		})
	}
	return e.JSON(http.StatusOK, dto.Response{Data: resp})
}

func (c *VersionController) Get(e echo.Context) error {
	userID, err := authenticatedUserID(e)
	if err != nil {
		return nil
	}
	documentID, err := parseDocumentIDParam(e)
	if err != nil {
		return nil
	}
	versionID, err := parseVersionIDParam(e)
	if err != nil {
		return nil
	}
	ctx := e.Request().Context()
	out, err := c.version.GetVersion(ctx, usecase.GetVersionInput{
		UserID: userID, DocumentID: documentID, VersionID: versionID,
	})
	if err != nil {
		return handleVersionUseCaseError(e, err)
	}
	return e.JSON(http.StatusOK, dto.Response{Data: dto.GetVersionResponse{
		ID: out.ID.String(), DocumentID: out.DocumentID.String(), Content: out.Content,
		Type: out.Type, CreatedBy: out.CreatedBy.String(), CreatedAt: out.CreatedAt,
	}})
}

func (c *VersionController) Save(e echo.Context) error {
	userID, err := authenticatedUserID(e)
	if err != nil {
		return nil
	}
	documentID, err := parseDocumentIDParam(e)
	if err != nil {
		return nil
	}
	ctx := e.Request().Context()
	out, err := c.version.SaveManualVersion(ctx, usecase.SaveManualVersionInput{
		UserID: userID, DocumentID: documentID,
	})
	if err != nil {
		return handleVersionUseCaseError(e, err)
	}
	return e.JSON(http.StatusCreated, dto.Response{Data: dto.SaveManualVersionResponse{
		ID: out.ID.String(), CreatedAt: out.CreatedAt,
	}})
}

func (c *VersionController) Restore(e echo.Context) error {
	userID, err := authenticatedUserID(e)
	if err != nil {
		return nil
	}
	documentID, err := parseDocumentIDParam(e)
	if err != nil {
		return nil
	}
	versionID, err := parseVersionIDParam(e)
	if err != nil {
		return nil
	}
	ctx := e.Request().Context()
	out, err := c.version.RestoreVersion(ctx, usecase.RestoreVersionInput{
		UserID: userID, DocumentID: documentID, VersionID: versionID,
	})
	if err != nil {
		return handleVersionUseCaseError(e, err)
	}
	return e.JSON(http.StatusOK, dto.Response{Data: dto.RestoreVersionResponse{
		DocumentID: out.DocumentID.String(),
		VersionID:  out.VersionID.String(),
		RestoredAt: out.RestoredAt,
	}})
}

func parseVersionIDParam(e echo.Context) (uuid.UUID, error) {
	raw := e.Param("versionId")
	id, err := uuid.Parse(raw)
	if err != nil {
		_ = writeWorkspaceError(e, http.StatusBadRequest, "INVALID_REQUEST", "バージョンIDが不正です")
		return uuid.Nil, errResponseSent
	}
	return id, nil
}

func handleVersionUseCaseError(e echo.Context, err error) error {
	switch {
	case errors.Is(err, usecase.ErrVersionNotFound):
		return writeWorkspaceError(e, http.StatusNotFound, "VERSION_NOT_FOUND", "編集履歴が見つかりません")
	case errors.Is(err, entity.ErrDocumentConflict):
		return writeWorkspaceError(e, http.StatusConflict, "DOCUMENT_CONFLICT", "ドキュメントが更新されています")
	default:
		return handleDocumentUseCaseError(e, err)
	}
}
