package controller

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/dto"
	"github.com/koezuka404/notehub/usecase"
	"github.com/labstack/echo/v4"
)

type DocumentController struct {
	doc usecase.IDocumentUsecase
}

func NewDocumentController(doc usecase.IDocumentUsecase) *DocumentController {
	return &DocumentController{doc: doc}
}

func (c *DocumentController) Create(ctx echo.Context) error {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		return err
	}
	workspaceID, err := parseWorkspaceIDParam(ctx)
	if err != nil {
		return err
	}
	var req dto.CreateDocumentRequest
	if err := ctx.Bind(&req); err != nil {
		return writeWorkspaceError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "リクエスト形式が不正です")
	}
	out, err := c.doc.CreateDocument(ctx.Request().Context(), usecase.CreateDocumentInput{
		UserID: userID, WorkspaceID: workspaceID, Title: req.Title, IPAddress: ctx.RealIP(),
	})
	if err != nil {
		return handleDocumentUseCaseError(ctx, err)
	}
	return ctx.JSON(http.StatusCreated, dto.Response{Data: dto.CreateDocumentResponse{
		ID: out.ID.String(), WorkspaceID: out.WorkspaceID.String(), Title: out.Title,
		Content: out.Content, CreatedBy: out.CreatedBy.String(), CreatedAt: out.CreatedAt,
	}})
}

func (c *DocumentController) List(ctx echo.Context) error {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		return err
	}
	workspaceID, err := parseWorkspaceIDParam(ctx)
	if err != nil {
		return err
	}
	items, err := c.doc.ListDocuments(ctx.Request().Context(), usecase.ListDocumentsInput{
		UserID: userID, WorkspaceID: workspaceID,
	})
	if err != nil {
		return handleDocumentUseCaseError(ctx, err)
	}
	resp := make([]dto.DocumentListItemResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, dto.DocumentListItemResponse{
			ID: item.ID.String(), Title: item.Title,
			UpdatedBy: item.UpdatedBy.String(), UpdatedAt: item.UpdatedAt,
		})
	}
	return ctx.JSON(http.StatusOK, dto.Response{Data: resp})
}

func (c *DocumentController) Get(ctx echo.Context) error {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		return err
	}
	documentID, err := parseDocumentIDParam(ctx)
	if err != nil {
		return err
	}
	out, err := c.doc.GetDocument(ctx.Request().Context(), usecase.GetDocumentInput{
		UserID: userID, DocumentID: documentID,
	})
	if err != nil {
		return handleDocumentUseCaseError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, dto.Response{Data: dto.GetDocumentResponse{
		ID: out.ID.String(), WorkspaceID: out.WorkspaceID.String(), Title: out.Title,
		Content: out.Content, UpdatedBy: out.UpdatedBy.String(), UpdatedAt: out.UpdatedAt,
	}})
}

func (c *DocumentController) Update(ctx echo.Context) error {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		return err
	}
	documentID, err := parseDocumentIDParam(ctx)
	if err != nil {
		return err
	}
	var req dto.UpdateDocumentRequest
	if err := ctx.Bind(&req); err != nil {
		return writeWorkspaceError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "リクエスト形式が不正です")
	}
	out, err := c.doc.UpdateDocument(ctx.Request().Context(), usecase.UpdateDocumentInput{
		UserID: userID, DocumentID: documentID, Title: req.Title, IPAddress: ctx.RealIP(),
	})
	if err != nil {
		return handleDocumentUseCaseError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, dto.Response{Data: dto.UpdateDocumentResponse{
		ID: out.ID.String(), Title: out.Title, UpdatedAt: out.UpdatedAt,
	}})
}

func (c *DocumentController) Delete(ctx echo.Context) error {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		return err
	}
	documentID, err := parseDocumentIDParam(ctx)
	if err != nil {
		return err
	}
	out, err := c.doc.DeleteDocument(ctx.Request().Context(), usecase.DeleteDocumentInput{
		UserID: userID, DocumentID: documentID, IPAddress: ctx.RealIP(),
	})
	if err != nil {
		return handleDocumentUseCaseError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, dto.Response{Data: dto.DeleteDocumentResponse{
		DocumentID: out.DocumentID.String(), DeletedAt: out.DeletedAt,
	}})
}

func parseDocumentIDParam(ctx echo.Context) (uuid.UUID, error) {
	raw := ctx.Param("documentId")
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, writeWorkspaceError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "ドキュメントIDが不正です")
	}
	return id, nil
}

func handleDocumentUseCaseError(ctx echo.Context, err error) error {
	switch {
	case errors.Is(err, usecase.ErrValidation):
		return writeWorkspaceError(ctx, http.StatusBadRequest, "VALIDATION_ERROR", "入力値が不正です")
	case errors.Is(err, usecase.ErrDocumentNotFound):
		return writeWorkspaceError(ctx, http.StatusNotFound, "DOCUMENT_NOT_FOUND", "ドキュメントが見つかりません")
	case errors.Is(err, usecase.ErrDocumentDeleted):
		return writeWorkspaceError(ctx, http.StatusNotFound, "DOCUMENT_DELETED", "ドキュメントは削除されています")
	default:
		return handleWorkspaceUseCaseError(ctx, err)
	}
}
