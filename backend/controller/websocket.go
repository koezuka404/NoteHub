package controller

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	gorillaws "github.com/gorilla/websocket"
	"github.com/koezuka404/notehub/config"
	"github.com/koezuka404/notehub/dto"
	appmiddleware "github.com/koezuka404/notehub/middleware"
	"github.com/koezuka404/notehub/usecase"
	appws "github.com/koezuka404/notehub/websocket"
	"github.com/labstack/echo/v4"
)

type WebSocketController struct {
	wsUseCase      usecase.IDocumentWebSocketUsecase
	tokens         appmiddleware.IAccessTokenValidator
	users          appmiddleware.IAuthUserFinder
	revoked        appmiddleware.IAccessTokenRevocationChecker
	hub            *appws.Hub
	allowedOrigins map[string]struct{}
	production     bool
	opTimeout      time.Duration
	now            func() time.Time
}

type WebSocketControllerConfig struct {
	AllowedOrigins   []string
	Production       bool
	OperationTimeout time.Duration
}

func NewWebSocketController(
	wsUseCase usecase.IDocumentWebSocketUsecase,
	tokens appmiddleware.IAccessTokenValidator,
	users appmiddleware.IAuthUserFinder,
	revoked appmiddleware.IAccessTokenRevocationChecker,
	hub *appws.Hub,
	cfg WebSocketControllerConfig,
) *WebSocketController {
	origins := make(map[string]struct{}, len(cfg.AllowedOrigins))
	for _, origin := range cfg.AllowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		origins[origin] = struct{}{}
	}
	return &WebSocketController{
		wsUseCase:      wsUseCase,
		tokens:         tokens,
		users:          users,
		revoked:        revoked,
		hub:            hub,
		allowedOrigins: origins,
		production:     cfg.Production,
		opTimeout:      cfg.OperationTimeout,
		now:            time.Now,
	}
}

func (c *WebSocketController) backgroundContext() (context.Context, context.CancelFunc) {
	timeout := c.opTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return context.WithTimeout(context.Background(), timeout)
}

func (c *WebSocketController) HandleDocument(e echo.Context) error {
	if c.production && e.Scheme() != "https" && e.Request().TLS == nil {
		return writeWebSocketHTTPError(e, http.StatusForbidden, "WEBSOCKET_TLS_REQUIRED", "WSS接続が必要です")
	}

	documentID, err := parseDocumentIDParam(e)
	if err != nil {
		return err
	}

	userID, err := c.authenticate(e)
	if err != nil {
		return err
	}

	ctx := e.Request().Context()
	prepared, err := c.wsUseCase.PrepareConnection(ctx, usecase.PrepareWebSocketConnectionInput{
		UserID: userID, DocumentID: documentID,
	})
	if err != nil {
		return handleWebSocketUseCaseError(e, err)
	}

	connectionID := uuid.New()
	registered, err := c.wsUseCase.RegisterConnection(ctx, usecase.RegisterWebSocketConnectionInput{
		UserID: userID, DocumentID: documentID, ConnectionID: connectionID,
	})
	if err != nil {
		return handleWebSocketUseCaseError(e, err)
	}

	upgrader := gorillaws.Upgrader{
		CheckOrigin: c.checkOrigin,
	}
	conn, err := upgrader.Upgrade(e.Response(), e.Request(), nil)
	if err != nil {
		unregCtx, cancel := c.backgroundContext()
		_, _ = c.wsUseCase.UnregisterConnection(unregCtx, usecase.UnregisterWebSocketConnectionInput{
			UserID: userID, DocumentID: documentID, ConnectionID: connectionID,
		})
		cancel()
		return err
	}

	client := &appws.Client{
		ConnectionID: connectionID,
		UserID:       userID,
		DocumentID:   documentID,
		WorkspaceID:  prepared.WorkspaceID,
		Hub:          c.hub,
		Conn:         conn,
		Send:         make(chan []byte, 16),
	}
	go client.WritePump()
	c.sendInitialEvents(client, prepared, registered)
	client.Ready.Store(true)
	c.hub.Register(client)
	go client.ReadPump(c.handleClientMessage, c.cleanupClient)
	return nil
}

func (c *WebSocketController) authenticate(e echo.Context) (uuid.UUID, error) {
	rawToken := strings.TrimSpace(e.QueryParam("access_token"))
	if rawToken == "" {
		authorization := strings.TrimSpace(e.Request().Header.Get(echo.HeaderAuthorization))
		parts := strings.Fields(authorization)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			rawToken = parts[1]
		}
	}
	if rawToken == "" {
		return uuid.Nil, writeWebSocketHTTPError(e, http.StatusUnauthorized, "ACCESS_TOKEN_REQUIRED", "アクセストークンが必要です")
	}

	claims, err := c.tokens.ValidateAccessToken(rawToken, c.now().UTC())
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return uuid.Nil, writeWebSocketHTTPError(e, http.StatusUnauthorized, "ACCESS_TOKEN_EXPIRED", "アクセストークンの有効期限が切れています")
		}
		return uuid.Nil, writeWebSocketHTTPError(e, http.StatusUnauthorized, "ACCESS_TOKEN_INVALID", "アクセストークンが不正です")
	}

	ctx := e.Request().Context()
	isRevoked, err := c.revoked.IsRevoked(ctx, claims.JTI)
	if err != nil {
		return uuid.Nil, writeWebSocketHTTPError(e, http.StatusServiceUnavailable, "AUTH_SERVICE_UNAVAILABLE", "認証サービスを利用できません")
	}
	if isRevoked {
		return uuid.Nil, writeWebSocketHTTPError(e, http.StatusUnauthorized, "ACCESS_TOKEN_REVOKED", "アクセストークンは失効しています")
	}

	user, found, err := c.users.FindByID(ctx, claims.UserID)
	if err != nil {
		return uuid.Nil, writeWebSocketHTTPError(e, http.StatusInternalServerError, "DATABASE_ERROR", "データベース処理に失敗しました")
	}
	if !found || !user.CanAuthenticate() {
		return uuid.Nil, writeWebSocketHTTPError(e, http.StatusUnauthorized, "ACCESS_TOKEN_INVALID", "アクセストークンが不正です")
	}
	if user.AuthVersion != claims.AuthVersion {
		return uuid.Nil, writeWebSocketHTTPError(e, http.StatusUnauthorized, "ACCESS_TOKEN_REVOKED", "アクセストークンは失効しています")
	}
	return claims.UserID, nil
}

func (c *WebSocketController) checkOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	if len(c.allowedOrigins) == 0 {
		return true
	}
	_, ok := c.allowedOrigins[origin]
	return ok
}

func (c *WebSocketController) sendInitialEvents(client *appws.Client, prepared *usecase.PrepareWebSocketConnectionOutput, registered *usecase.RegisterWebSocketConnectionOutput) {
	now := c.now()
	c.sendEvent(client, appws.EventConnected, appws.ConnectedData{
		ConnectionID: client.ConnectionID.String(),
		DocumentID:   prepared.DocumentID.String(),
	}, now)
	c.sendEvent(client, appws.EventDocumentSync, appws.DocumentSyncData{
		DocumentID: prepared.DocumentID.String(),
		Content:    prepared.Content,
		UpdatedAt:  prepared.UpdatedAt,
	}, now)
	c.sendEvent(client, appws.EventEditorsSync, appws.EditorsSyncData{
		Editors: toEditorEventData(registered.Editors),
	}, now)
	if registered.EditorJoined {
		payload, err := appws.MarshalEvent(appws.EventEditorJoined, appws.EditorEventData{
			UserID: registered.JoinedEditor.UserID.String(),
			Name:   registered.JoinedEditor.Name,
		}, now)
		if err == nil {
			c.hub.BroadcastDocumentExcept(client.DocumentID, client, payload)
		}
	}
}

func (c *WebSocketController) handleClientMessage(client *appws.Client, raw []byte) {
	var message appws.ClientMessage
	if err := json.Unmarshal(raw, &message); err != nil {
		c.sendError(client, "INVALID_REQUEST", "メッセージ形式が不正です")
		return
	}

	switch message.Type {
	case appws.EventDocumentEdit:
		var data appws.DocumentEditData
		if err := json.Unmarshal(message.Data, &data); err != nil {
			c.sendError(client, "INVALID_REQUEST", "編集内容が不正です")
			return
		}
		ctx, cancel := c.backgroundContext()
		out, err := c.wsUseCase.ApplyDocumentEdit(ctx, usecase.ApplyDocumentEditInput{
			UserID: client.UserID, DocumentID: client.DocumentID, Content: data.Content,
		})
		cancel()
		if err != nil {
			c.sendUseCaseError(client, err)
			return
		}
		payload, err := appws.MarshalEvent(appws.EventDocumentUpdated, appws.DocumentUpdatedData{
			DocumentID: out.DocumentID.String(),
			Content:    out.Content,
			UpdatedBy:  out.UpdatedBy.String(),
			UpdatedAt:  out.UpdatedAt,
		}, c.now())
		if err != nil {
			return
		}
		c.hub.BroadcastDocumentExcept(client.DocumentID, client, payload)
	default:
		c.sendError(client, "UNSUPPORTED_EVENT", "未対応のイベントです")
	}
}

func (c *WebSocketController) cleanupClient(client *appws.Client) {
	ctx, cancel := c.backgroundContext()
	out, err := c.wsUseCase.UnregisterConnection(ctx, usecase.UnregisterWebSocketConnectionInput{
		UserID: client.UserID, DocumentID: client.DocumentID, ConnectionID: client.ConnectionID,
	})
	cancel()
	if err != nil {
		log.Printf("websocket: unregister connection failed document=%s user=%s connection=%s: %v",
			client.DocumentID, client.UserID, client.ConnectionID, err)
		return
	}
	now := c.now()
	if out.EditorLeft {
		payload, err := appws.MarshalEvent(appws.EventEditorLeft, appws.EditorEventData{
			UserID: out.LeftEditor.UserID.String(),
			Name:   out.LeftEditor.Name,
		}, now)
		if err == nil {
			c.hub.BroadcastDocument(client.DocumentID, payload)
		}
	}
}

func (c *WebSocketController) sendEvent(client *appws.Client, eventType string, data any, now time.Time) {
	payload, err := appws.MarshalEvent(eventType, data, now)
	if err != nil {
		return
	}
	client.TrySend(payload)
}

func (c *WebSocketController) sendError(client *appws.Client, code, message string) {
	c.sendEvent(client, appws.EventError, appws.ErrorData{Code: code, Message: message}, c.now())
}

func (c *WebSocketController) sendUseCaseError(client *appws.Client, err error) {
	switch {
	case errors.Is(err, usecase.ErrValidation):
		c.sendError(client, "VALIDATION_ERROR", "入力値が不正です")
	case errors.Is(err, usecase.ErrDocumentNotFound):
		c.sendError(client, "DOCUMENT_NOT_FOUND", "ドキュメントが見つかりません")
	case errors.Is(err, usecase.ErrDocumentDeleted):
		c.sendError(client, "DOCUMENT_DELETED", "ドキュメントは削除されています")
	case errors.Is(err, usecase.ErrDocumentContentTooLarge):
		c.sendError(client, "DOCUMENT_CONTENT_TOO_LARGE", "ドキュメント本文が上限を超えています")
	case errors.Is(err, usecase.ErrWorkspaceNotFound):
		c.sendError(client, "WORKSPACE_NOT_FOUND", "ワークスペースが見つかりません")
	case errors.Is(err, usecase.ErrWorkspaceAccessDenied):
		c.sendError(client, "WORKSPACE_ACCESS_DENIED", "このワークスペースへアクセスできません")
	case errors.Is(err, usecase.ErrWorkspaceHostSuspended):
		c.sendError(client, "WORKSPACE_HOST_SUSPENDED", "ホストが停止されているため利用できません")
	case errors.Is(err, usecase.ErrWorkspaceHostDeleted):
		c.sendError(client, "WORKSPACE_HOST_DELETED", "ホストが削除されているため利用できません")
	default:
		c.sendError(client, "INTERNAL_ERROR", "処理に失敗しました")
	}
}

func toEditorEventData(editors []usecase.DocumentEditorInfo) []appws.EditorEventData {
	out := make([]appws.EditorEventData, 0, len(editors))
	for _, editor := range editors {
		out = append(out, appws.EditorEventData{
			UserID: editor.UserID.String(),
			Name:   editor.Name,
		})
	}
	return out
}

func writeWebSocketHTTPError(e echo.Context, status int, code, message string) error {
	return e.JSON(status, dto.ErrorResponse{Error: dto.ErrorBody{Code: code, Message: message}})
}

func handleWebSocketUseCaseError(e echo.Context, err error) error {
	switch {
	case errors.Is(err, usecase.ErrWebSocketConnectionLimitExceeded):
		return writeWebSocketHTTPError(e, http.StatusConflict, "WEBSOCKET_CONNECTION_LIMIT_EXCEEDED", "WebSocket接続数の上限に達しています")
	case errors.Is(err, usecase.ErrValidation):
		return writeWebSocketHTTPError(e, http.StatusBadRequest, "VALIDATION_ERROR", "入力値が不正です")
	case errors.Is(err, usecase.ErrDocumentNotFound):
		return writeWebSocketHTTPError(e, http.StatusNotFound, "DOCUMENT_NOT_FOUND", "ドキュメントが見つかりません")
	case errors.Is(err, usecase.ErrDocumentDeleted):
		return writeWebSocketHTTPError(e, http.StatusNotFound, "DOCUMENT_DELETED", "ドキュメントは削除されています")
	default:
		return handleWorkspaceUseCaseError(e, err)
	}
}

func WebSocketControllerConfigFromApp(cfg *config.Config) WebSocketControllerConfig {
	origins := cfg.AllowedOrigins
	if len(origins) == 0 && cfg.Environment == config.EnvironmentDevelopment {
		origins = []string{"http://localhost:5173", "http://127.0.0.1:5173"}
	}
	return WebSocketControllerConfig{
		AllowedOrigins:   origins,
		Production:       cfg.Environment == config.EnvironmentProduction,
		OperationTimeout: cfg.RedisOperationTimeout,
	}
}
