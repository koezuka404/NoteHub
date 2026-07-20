package controller

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/dto"
	appmiddleware "github.com/koezuka404/notehub/middleware"
	"github.com/koezuka404/notehub/usecase"
	"github.com/labstack/echo/v4"
)

type AuthCookieConfig struct {
	RefreshName string
	CSRFName    string
	Domain      string
	SameSite    string
	Secure      bool
	RefreshTTL  time.Duration
}

type AuthController struct {
	auth    usecase.AuthInputPort
	cookies AuthCookieConfig
}

func NewAuthController(auth usecase.AuthInputPort, cookies AuthCookieConfig) *AuthController {
	return &AuthController{auth: auth, cookies: cookies}
}

func (c *AuthController) Register(ctx echo.Context) error {
	var r dto.RegisterRequest
	if err := ctx.Bind(&r); err != nil {
		return writeAuthError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "リクエスト形式が不正です")
	}
	o, err := c.auth.Register(ctx.Request().Context(), usecase.RegisterInput{Name: r.Name, Email: r.Email, Password: r.Password})
	if err != nil {
		return handleAuthUseCaseError(ctx, err)
	}
	return ctx.JSON(http.StatusCreated, dto.Response{Data: dto.RegisterResponse{User: dto.AuthUserResponse{ID: o.ID.String(), Name: o.Name, Email: o.Email, Status: string(o.Status)}, CreatedAt: o.CreatedAt}})
}

func (c *AuthController) Login(ctx echo.Context) error {
	var r dto.LoginRequest
	if err := ctx.Bind(&r); err != nil {
		return writeAuthError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "リクエスト形式が不正です")
	}
	o, err := c.auth.Login(ctx.Request().Context(), usecase.LoginInput{Email: r.Email, Password: r.Password, IPAddress: ctx.RealIP(), UserAgent: ctx.Request().UserAgent()})
	if err != nil {
		return handleAuthUseCaseError(ctx, err)
	}
	c.setRefreshTokenCookie(ctx, o.RefreshToken)
	c.setCSRFTokenCookie(ctx, o.CSRFToken)
	return ctx.JSON(http.StatusOK, dto.Response{Data: dto.LoginResponse{User: dto.AuthUserResponse{ID: o.User.ID.String(), Name: o.User.Name, Email: o.User.Email, Status: string(o.User.Status)}, AccessToken: o.AccessToken, TokenType: o.TokenType, ExpiresAt: o.ExpiresAt}})
}

func (c *AuthController) Refresh(ctx echo.Context) error {
	cookie, err := ctx.Cookie(c.cookies.RefreshName)
	if err != nil || cookie.Value == "" {
		return handleAuthUseCaseError(ctx, usecase.ErrRefreshTokenRequired)
	}
	o, err := c.auth.Refresh(ctx.Request().Context(), usecase.RefreshInput{RefreshToken: cookie.Value, IPAddress: ctx.RealIP(), UserAgent: ctx.Request().UserAgent()})
	if err != nil {
		return handleAuthUseCaseError(ctx, err)
	}
	c.setRefreshTokenCookie(ctx, o.RefreshToken)
	c.setCSRFTokenCookie(ctx, o.CSRFToken)
	return ctx.JSON(http.StatusOK, dto.Response{Data: dto.RefreshResponse{AccessToken: o.AccessToken, TokenType: o.TokenType, ExpiresAt: o.ExpiresAt}})
}

func (c *AuthController) Logout(ctx echo.Context) error {
	userID, ok := ctx.Get(appmiddleware.ContextUserID).(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return writeAuthError(ctx, http.StatusUnauthorized, "ACCESS_TOKEN_INVALID", "アクセストークンが不正です")
	}
	jti, ok := ctx.Get(appmiddleware.ContextAccessTokenJTI).(uuid.UUID)
	if !ok || jti == uuid.Nil {
		return writeAuthError(ctx, http.StatusUnauthorized, "ACCESS_TOKEN_INVALID", "アクセストークンが不正です")
	}
	expiresAt, ok := ctx.Get(appmiddleware.ContextAccessTokenExp).(time.Time)
	if !ok || expiresAt.IsZero() {
		return writeAuthError(ctx, http.StatusUnauthorized, "ACCESS_TOKEN_INVALID", "アクセストークンが不正です")
	}
	csrfValidated, _ := ctx.Get("csrf_validated").(bool)
	refreshToken := ""
	if cookie, err := ctx.Cookie(c.cookies.RefreshName); err == nil {
		refreshToken = cookie.Value
	}
	_, err := c.auth.Logout(ctx.Request().Context(), usecase.LogoutInput{
		UserID: userID, AccessTokenJTI: jti, AccessTokenExp: expiresAt,
		RefreshToken: refreshToken, IPAddress: ctx.RealIP(), UserAgent: ctx.Request().UserAgent(),
		CSRFValidated: csrfValidated,
	})
	if err != nil {
		return handleAuthUseCaseError(ctx, err)
	}
	c.clearAuthCookies(ctx)
	return ctx.JSON(http.StatusOK, dto.Response{Data: dto.LogoutResponse{Message: "ログアウトしました"}})
}

func (c *AuthController) clearAuthCookies(ctx echo.Context) {
	ctx.SetCookie(&http.Cookie{Name: c.cookies.RefreshName, Value: "", Path: "/api/auth", Domain: c.cookies.Domain, MaxAge: -1, Expires: time.Unix(0, 0), HttpOnly: true, Secure: c.cookies.Secure, SameSite: parseSameSite(c.cookies.SameSite)})
	ctx.SetCookie(&http.Cookie{Name: c.cookies.CSRFName, Value: "", Path: "/", Domain: c.cookies.Domain, MaxAge: -1, Expires: time.Unix(0, 0), HttpOnly: false, Secure: c.cookies.Secure, SameSite: parseSameSite(c.cookies.SameSite)})
}

func (c *AuthController) setRefreshTokenCookie(ctx echo.Context, refresh string) {
	ctx.SetCookie(&http.Cookie{Name: c.cookies.RefreshName, Value: refresh, Path: "/api/auth", Domain: c.cookies.Domain, MaxAge: int(c.cookies.RefreshTTL.Seconds()), HttpOnly: true, Secure: c.cookies.Secure, SameSite: parseSameSite(c.cookies.SameSite)})
}

func (c *AuthController) setCSRFTokenCookie(ctx echo.Context, token string) {
	ctx.SetCookie(&http.Cookie{Name: c.cookies.CSRFName, Value: token, Path: "/", Domain: c.cookies.Domain, MaxAge: int(c.cookies.RefreshTTL.Seconds()), HttpOnly: false, Secure: c.cookies.Secure, SameSite: parseSameSite(c.cookies.SameSite)})
}

func parseSameSite(v string) http.SameSite {
	switch v {
	case "Strict":
		return http.SameSiteStrictMode
	case "None":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

func handleAuthUseCaseError(ctx echo.Context, err error) error {
	switch {
	case errors.Is(err, usecase.ErrValidation):
		return writeAuthError(ctx, 400, "VALIDATION_ERROR", "入力値が不正です")
	case errors.Is(err, usecase.ErrEmailAlreadyExists):
		return writeAuthError(ctx, 409, "EMAIL_ALREADY_EXISTS", "このメールアドレスは既に登録されています")
	case errors.Is(err, usecase.ErrInvalidCredentials):
		return writeAuthError(ctx, 401, "INVALID_CREDENTIALS", "メールアドレスまたはパスワードが正しくありません")
	case errors.Is(err, usecase.ErrAccountSuspended), errors.Is(err, usecase.ErrAccountDeleted):
		return writeAuthError(ctx, 401, "ACCOUNT_UNAVAILABLE", "このアカウントは利用できません")
	case errors.Is(err, usecase.ErrRefreshTokenRequired):
		return writeAuthError(ctx, 401, "REFRESH_TOKEN_REQUIRED", "リフレッシュトークンが必要です")
	case errors.Is(err, usecase.ErrRefreshTokenExpired):
		return writeAuthError(ctx, 401, "REFRESH_TOKEN_EXPIRED", "リフレッシュトークンの有効期限が切れています")
	case errors.Is(err, usecase.ErrRefreshTokenInvalid):
		return writeAuthError(ctx, 401, "REFRESH_TOKEN_INVALID", "リフレッシュトークンが不正です")
	case errors.Is(err, usecase.ErrRefreshTokenRevoked), errors.Is(err, usecase.ErrRefreshTokenReused):
		return writeAuthError(ctx, 401, "REFRESH_TOKEN_REVOKED", "リフレッシュトークンは失効しています")
	case errors.Is(err, usecase.ErrTokenOwnerMismatch):
		return writeAuthError(ctx, 403, "TOKEN_OWNER_MISMATCH", "トークンの所有者が一致しません")
	case errors.Is(err, usecase.ErrCSRFTokenInvalid):
		return writeAuthError(ctx, 403, "CSRF_TOKEN_INVALID", "CSRFトークンが不正です")
	case errors.Is(err, usecase.ErrAuthServiceUnavailable):
		return writeAuthError(ctx, 503, "AUTH_SERVICE_UNAVAILABLE", "認証サービスを利用できません")
	case errors.Is(err, usecase.ErrAccessTokenInvalid):
		return writeAuthError(ctx, 401, "ACCESS_TOKEN_INVALID", "アクセストークンが不正です")
	default:
		return writeAuthError(ctx, 500, "INTERNAL_ERROR", "内部エラーが発生しました")
	}
}

func writeAuthError(ctx echo.Context, status int, code, message string) error {
	return ctx.JSON(status, dto.ErrorResponse{Error: dto.ErrorBody{Code: code, Message: message}})
}
