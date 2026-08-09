package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
	"github.com/koezuka404/notehub/repository"
)

type DocumentEditorInfo struct {
	UserID uuid.UUID
	Name   string
}

type IWebSocketSessionStore interface {
	TryAddConnection(ctx context.Context, documentID, userID, connectionID uuid.UUID, maxConnections int, ttl time.Duration) (count int, added bool, err error)
	RemoveConnection(ctx context.Context, documentID, userID, connectionID uuid.UUID) (int, error)
}

type IDocumentEditorsStore interface {
	List(ctx context.Context, documentID uuid.UUID) ([]DocumentEditorInfo, error)
	Add(ctx context.Context, documentID uuid.UUID, editor DocumentEditorInfo) (joined bool, editors []DocumentEditorInfo, err error)
	Remove(ctx context.Context, documentID, userID uuid.UUID) (left bool, editors []DocumentEditorInfo, err error)
	RefreshTTL(ctx context.Context, documentID uuid.UUID) error
}

type IDocumentWebSocketUsecase interface {
	PrepareConnection(ctx context.Context, input PrepareWebSocketConnectionInput) (*PrepareWebSocketConnectionOutput, error)
	PrepareWorkspaceConnection(ctx context.Context, input PrepareWorkspaceConnectionInput) error
	RegisterConnection(ctx context.Context, input RegisterWebSocketConnectionInput) (*RegisterWebSocketConnectionOutput, error)
	UnregisterConnection(ctx context.Context, input UnregisterWebSocketConnectionInput) (*UnregisterWebSocketConnectionOutput, error)
	ApplyDocumentEdit(ctx context.Context, input ApplyDocumentEditInput) (*ApplyDocumentEditOutput, error)
}

type DocumentWebSocketUseCase struct {
	docs         repository.DocumentRepository
	users        repository.UserRepository
	cache        IDocumentCache
	sessions     IWebSocketSessionStore
	editors      IDocumentEditorsStore
	access       IAccessCheck
	maxUserConns int
	sessionTTL   time.Duration
	now          func() time.Time
}

func NewDocumentWebSocketUseCase(
	docs repository.DocumentRepository,
	users repository.UserRepository,
	cache IDocumentCache,
	sessions IWebSocketSessionStore,
	editors IDocumentEditorsStore,
	access IAccessCheck,
	maxUserConns int,
	sessionTTL time.Duration,
) *DocumentWebSocketUseCase {
	if maxUserConns < 1 {
		maxUserConns = 1
	}
	if sessionTTL <= 0 {
		sessionTTL = 16 * time.Minute
	}
	return &DocumentWebSocketUseCase{
		docs:         docs,
		users:        users,
		cache:        cache,
		sessions:     sessions,
		editors:      editors,
		access:       access,
		maxUserConns: maxUserConns,
		sessionTTL:   sessionTTL,
		now:          time.Now,
	}
}

type PrepareWebSocketConnectionInput struct {
	UserID     uuid.UUID
	DocumentID uuid.UUID
}

type PrepareWebSocketConnectionOutput struct {
	DocumentID  uuid.UUID
	WorkspaceID uuid.UUID
	Content     string
	UpdatedAt   string
}

func (uc *DocumentWebSocketUseCase) PrepareConnection(ctx context.Context, input PrepareWebSocketConnectionInput) (*PrepareWebSocketConnectionOutput, error) {
	if input.UserID == uuid.Nil || input.DocumentID == uuid.Nil {
		return nil, ErrValidation
	}
	doc, err := uc.loadDoc(ctx, input.DocumentID)
	if err != nil {
		return nil, err
	}
	if err := uc.authorizeDoc(ctx, input.UserID, doc); err != nil {
		return nil, err
	}

	content, updatedAt, err := uc.resolveContent(ctx, doc)
	if err != nil {
		return nil, err
	}
	return &PrepareWebSocketConnectionOutput{
		DocumentID:  doc.ID,
		WorkspaceID: doc.WorkspaceID,
		Content:     content,
		UpdatedAt:   updatedAt,
	}, nil
}

type PrepareWorkspaceConnectionInput struct {
	UserID      uuid.UUID
	WorkspaceID uuid.UUID
}

func (uc *DocumentWebSocketUseCase) PrepareWorkspaceConnection(ctx context.Context, input PrepareWorkspaceConnectionInput) error {
	if input.UserID == uuid.Nil || input.WorkspaceID == uuid.Nil {
		return ErrValidation
	}
	_, err := uc.access.CheckWorkspaceAccess(ctx, CheckWorkspaceAccessInput{
		UserID: input.UserID, WorkspaceID: input.WorkspaceID,
	})
	return err
}

type RegisterWebSocketConnectionInput struct {
	UserID       uuid.UUID
	DocumentID   uuid.UUID
	ConnectionID uuid.UUID
}

type RegisterWebSocketConnectionOutput struct {
	Editors      []DocumentEditorInfo
	EditorJoined bool
	JoinedEditor DocumentEditorInfo
}

func (uc *DocumentWebSocketUseCase) RegisterConnection(ctx context.Context, input RegisterWebSocketConnectionInput) (*RegisterWebSocketConnectionOutput, error) {
	if input.UserID == uuid.Nil || input.DocumentID == uuid.Nil || input.ConnectionID == uuid.Nil {
		return nil, ErrValidation
	}
	if _, err := uc.PrepareConnection(ctx, PrepareWebSocketConnectionInput{
		UserID: input.UserID, DocumentID: input.DocumentID,
	}); err != nil {
		return nil, err
	}

	user, found, err := uc.users.FindByID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("find websocket user: %w", err)
	}
	if !found {
		return nil, ErrUserNotFound
	}

	_, added, err := uc.sessions.TryAddConnection(
		ctx,
		input.DocumentID,
		input.UserID,
		input.ConnectionID,
		uc.maxUserConns,
		uc.sessionTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("register websocket connection: %w", err)
	}
	if !added {
		return nil, ErrWebSocketConnectionLimitExceeded
	}

	connectionRegistered := true
	defer func() {
		if connectionRegistered {
			rollbackCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, _ = uc.sessions.RemoveConnection(rollbackCtx, input.DocumentID, input.UserID, input.ConnectionID)
		}
	}()

	joined, editors, err := uc.editors.Add(ctx, input.DocumentID, DocumentEditorInfo{
		UserID: input.UserID,
		Name:   user.Name,
	})
	if err != nil {
		return nil, fmt.Errorf("register document editor: %w", err)
	}

	connectionRegistered = false
	output := &RegisterWebSocketConnectionOutput{Editors: editors}
	if joined {
		output.EditorJoined = true
		output.JoinedEditor = DocumentEditorInfo{UserID: input.UserID, Name: user.Name}
	}
	return output, nil
}

type UnregisterWebSocketConnectionInput struct {
	UserID       uuid.UUID
	DocumentID   uuid.UUID
	ConnectionID uuid.UUID
}

type UnregisterWebSocketConnectionOutput struct {
	Editors    []DocumentEditorInfo
	EditorLeft bool
	LeftEditor DocumentEditorInfo
}

func (uc *DocumentWebSocketUseCase) UnregisterConnection(ctx context.Context, input UnregisterWebSocketConnectionInput) (*UnregisterWebSocketConnectionOutput, error) {
	if input.UserID == uuid.Nil || input.DocumentID == uuid.Nil || input.ConnectionID == uuid.Nil {
		return nil, ErrValidation
	}

	remaining, err := uc.sessions.RemoveConnection(ctx, input.DocumentID, input.UserID, input.ConnectionID)
	if err != nil {
		return nil, fmt.Errorf("unregister websocket connection: %w", err)
	}

	output := &UnregisterWebSocketConnectionOutput{}
	if remaining > 0 {
		if err := uc.editors.RefreshTTL(ctx, input.DocumentID); err != nil {
			return nil, fmt.Errorf("refresh document editors ttl: %w", err)
		}
		editors, err := uc.editors.List(ctx, input.DocumentID)
		if err != nil {
			return nil, fmt.Errorf("list document editors: %w", err)
		}
		output.Editors = editors
		return output, nil
	}

	user, found, err := uc.users.FindByID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("find websocket user: %w", err)
	}
	if !found {
		return nil, ErrUserNotFound
	}

	left, editors, err := uc.editors.Remove(ctx, input.DocumentID, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("remove document editor: %w", err)
	}
	output.Editors = editors
	if left {
		output.EditorLeft = true
		output.LeftEditor = DocumentEditorInfo{UserID: input.UserID, Name: user.Name}
	}
	return output, nil
}

type ApplyDocumentEditInput struct {
	UserID     uuid.UUID
	DocumentID uuid.UUID
	Content    string
}

type ApplyDocumentEditOutput struct {
	DocumentID uuid.UUID
	Content    string
	UpdatedBy  uuid.UUID
	UpdatedAt  string
}

func (uc *DocumentWebSocketUseCase) ApplyDocumentEdit(ctx context.Context, input ApplyDocumentEditInput) (*ApplyDocumentEditOutput, error) {
	if input.UserID == uuid.Nil || input.DocumentID == uuid.Nil {
		return nil, ErrValidation
	}
	doc, err := uc.loadDoc(ctx, input.DocumentID)
	if err != nil {
		return nil, err
	}
	if err := uc.authorizeDoc(ctx, input.UserID, doc); err != nil {
		return nil, err
	}
	if err := validDocumentContent(input.Content); err != nil {
		return nil, err
	}
	if uc.cache == nil {
		return nil, fmt.Errorf("document cache is not configured")
	}

	now := uc.currentTime()
	if err := uc.cache.SetContent(ctx, input.DocumentID, input.Content, input.UserID, now); err != nil {
		return nil, fmt.Errorf("save document edit to cache: %w", err)
	}
	return &ApplyDocumentEditOutput{
		DocumentID: input.DocumentID,
		Content:    input.Content,
		UpdatedBy:  input.UserID,
		UpdatedAt:  now.Format(timeFormat),
	}, nil
}

func (uc *DocumentWebSocketUseCase) currentTime() time.Time {
	return uc.now().UTC()
}

func (uc *DocumentWebSocketUseCase) loadDoc(ctx context.Context, documentID uuid.UUID) (*entity.Document, error) {
	doc, found, err := uc.docs.FindByID(ctx, documentID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrDocumentNotFound
	}
	if doc.IsDeleted() {
		return nil, ErrDocumentDeleted
	}
	return doc, nil
}

func (uc *DocumentWebSocketUseCase) authorizeDoc(ctx context.Context, userID uuid.UUID, doc *entity.Document) error {
	return uc.requireMember(ctx, userID, doc.WorkspaceID)
}

func (uc *DocumentWebSocketUseCase) requireMember(ctx context.Context, userID, workspaceID uuid.UUID) error {
	_, err := uc.access.CheckWorkspaceAccess(ctx, CheckWorkspaceAccessInput{
		UserID: userID, WorkspaceID: workspaceID,
	})
	return err
}

func (uc *DocumentWebSocketUseCase) resolveContent(ctx context.Context, doc *entity.Document) (string, string, error) {
	if uc.cache != nil {
		if state, ok, err := uc.cache.GetContentState(ctx, doc.ID); err != nil {
			return "", "", err
		} else if ok {
			return state.Content, state.UpdatedAt.Format(timeFormat), nil
		}
	}
	return doc.Content, doc.UpdatedAt.UTC().Format(timeFormat), nil
}
