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
	auth    usecase.IAuthUsecase
	cookies AuthCookieConfig
}

func NewAuthController(auth usecase.IAuthUsecase, cookies AuthCookieConfig) *AuthController {
	return &AuthController{auth: auth, cookies: cookies}
}

// RegisterはPOST/api/auth/registerを処理して新規ユーザーを作成
func (c *AuthController) Register(e echo.Context) error {
	var r dto.RegisterRequest
	if err := e.Bind(&r); err != nil {
		return writeAuthError(e, http.StatusBadRequest, "INVALID_REQUEST", "リクエスト形式が不正です")
	}
	ctx := e.Request().Context()
	o, err := c.auth.Register(ctx, usecase.RegisterInput{Name: r.Name, Email: r.Email, Password: r.Password, IPAddress: e.RealIP()})
	if err != nil {
		return handleAuthUseCaseError(e, err)
	}
	return e.JSON(http.StatusCreated, dto.Response{Data: dto.RegisterResponse{User: dto.AuthUserResponse{ID: o.ID.String(), Name: o.Name, Email: o.Email, Status: string(o.Status)}, CreatedAt: o.CreatedAt}})
}

// LoginはPOST/api/auth/loginを処理してAccessTokenとCookieを返す
func (c *AuthController) Login(e echo.Context) error {
	var r dto.LoginRequest
	if err := e.Bind(&r); err != nil {
		return writeAuthError(e, http.StatusBadRequest, "INVALID_REQUEST", "リクエスト形式が不正です")
	}
	ctx := e.Request().Context()
	o, err := c.auth.Login(ctx, usecase.LoginInput{Email: r.Email, Password: r.Password, IPAddress: e.RealIP(), UserAgent: e.Request().UserAgent()})
	if err != nil {
		return handleAuthUseCaseError(e, err)
	}
	c.setRefreshTokenCookie(e, o.RefreshToken)
	c.setCSRFTokenCookie(e, o.CSRFToken)
	return e.JSON(http.StatusOK, dto.Response{Data: dto.LoginResponse{User: dto.AuthUserResponse{ID: o.User.ID.String(), Name: o.User.Name, Email: o.User.Email, Status: string(o.User.Status)}, AccessToken: o.AccessToken, TokenType: o.TokenType, ExpiresAt: o.ExpiresAt}})
}

// RefreshはPOST/api/auth/refreshを処理　RefreshTokenからAccessTokenを再発行する
func (c *AuthController) Refresh(e echo.Context) error {
	cookie, err := e.Cookie(c.cookies.RefreshName)
	if err != nil || cookie.Value == "" {
		return handleAuthUseCaseError(e, usecase.ErrRefreshTokenRequired)
	}
	ctx := e.Request().Context()
	o, err := c.auth.Refresh(ctx, usecase.RefreshInput{RefreshToken: cookie.Value, IPAddress: e.RealIP(), UserAgent: e.Request().UserAgent()})
	if err != nil {
		return handleAuthUseCaseError(e, err)
	}
	c.setRefreshTokenCookie(e, o.RefreshToken)
	c.setCSRFTokenCookie(e, o.CSRFToken)
	return e.JSON(http.StatusOK, dto.Response{Data: dto.RefreshResponse{AccessToken: o.AccessToken, TokenType: o.TokenType, ExpiresAt: o.ExpiresAt}})
}

// MeはGET/api/meを処理　認証済みユーザーの情報を返す
func (c *AuthController) Me(e echo.Context) error {
	userID, ok := e.Get(appmiddleware.ContextUserID).(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return writeAuthError(e, http.StatusUnauthorized, "ACCESS_TOKEN_INVALID", "アクセストークンが不正です")
	}
	ctx := e.Request().Context()
	o, err := c.auth.GetCurrentUser(ctx, usecase.GetCurrentUserInput{UserID: userID})
	if err != nil {
		return handleAuthUseCaseError(e, err)
	}
	return e.JSON(http.StatusOK, dto.Response{Data: dto.MeResponse{User: dto.AuthUserResponse{ID: o.ID.String(), Name: o.Name, Email: o.Email, Status: string(o.Status)}}})
}

// LogoutはPOST/api/auth/logoutを処理　トークンを失効させてCookieを削除
func (c *AuthController) Logout(e echo.Context) error {
	userID, ok := e.Get(appmiddleware.ContextUserID).(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return writeAuthError(e, http.StatusUnauthorized, "ACCESS_TOKEN_INVALID", "アクセストークンが不正です")
	}
	jti, ok := e.Get(appmiddleware.ContextAccessTokenJTI).(uuid.UUID)
	if !ok || jti == uuid.Nil {
		return writeAuthError(e, http.StatusUnauthorized, "ACCESS_TOKEN_INVALID", "アクセストークンが不正です")
	}
	expiresAt, ok := e.Get(appmiddleware.ContextAccessTokenExp).(time.Time)
	if !ok || expiresAt.IsZero() {
		return writeAuthError(e, http.StatusUnauthorized, "ACCESS_TOKEN_INVALID", "アクセストークンが不正です")
	}
	csrfValidated, _ := e.Get("csrf_validated").(bool)
	refreshToken := ""
	if cookie, err := e.Cookie(c.cookies.RefreshName); err == nil {
		refreshToken = cookie.Value
	}
	ctx := e.Request().Context()
	_, err := c.auth.Logout(ctx, usecase.LogoutInput{
		UserID: userID, AccessTokenJTI: jti, AccessTokenExp: expiresAt,
		RefreshToken: refreshToken, IPAddress: e.RealIP(), UserAgent: e.Request().UserAgent(),
		CSRFValidated: csrfValidated,
	})
	if err != nil {
		return handleAuthUseCaseError(e, err)
	}
	c.clearAuthCookies(e)
	return e.JSON(http.StatusOK, dto.Response{Data: dto.LogoutResponse{Message: "ログアウトしました"}})
}

// clearAuthCookies　Refresh/CSRFCookieを削除する
func (c *AuthController) clearAuthCookies(e echo.Context) {
	e.SetCookie(&http.Cookie{Name: c.cookies.RefreshName, Value: "", Path: "/api/auth", Domain: c.cookies.Domain, MaxAge: -1, Expires: time.Unix(0, 0), HttpOnly: true, Secure: c.cookies.Secure, SameSite: parseSameSite(c.cookies.SameSite)})
	e.SetCookie(&http.Cookie{Name: c.cookies.CSRFName, Value: "", Path: "/", Domain: c.cookies.Domain, MaxAge: -1, Expires: time.Unix(0, 0), HttpOnly: false, Secure: c.cookies.Secure, SameSite: parseSameSite(c.cookies.SameSite)})
}

// setRefreshTokenCookie　HttpOnlyのRefreshTokenCookieを設定する
func (c *AuthController) setRefreshTokenCookie(e echo.Context, refresh string) {
	e.SetCookie(&http.Cookie{Name: c.cookies.RefreshName, Value: refresh, Path: "/api/auth", Domain: c.cookies.Domain, MaxAge: int(c.cookies.RefreshTTL.Seconds()), HttpOnly: true, Secure: c.cookies.Secure, SameSite: parseSameSite(c.cookies.SameSite)})
}

// setCSRFTokenCookie　CSRF対策用Cookieを設定する
func (c *AuthController) setCSRFTokenCookie(e echo.Context, token string) {
	e.SetCookie(&http.Cookie{Name: c.cookies.CSRFName, Value: token, Path: "/", Domain: c.cookies.Domain, MaxAge: int(c.cookies.RefreshTTL.Seconds()), HttpOnly: false, Secure: c.cookies.Secure, SameSite: parseSameSite(c.cookies.SameSite)})
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

// handleAuthUseCaseError　は認証のUseCaseエラーをHTTPステータスとJSONレスポンスへ変換
func handleAuthUseCaseError(e echo.Context, err error) error {
	switch {
	case errors.Is(err, usecase.ErrValidation):
		return writeAuthError(e, 400, "VALIDATION_ERROR", "入力値が不正です")
	case errors.Is(err, usecase.ErrPasswordInvalid):
		return writeAuthError(e, 400, "PASSWORD_INVALID", "パスワードの入力内容を確認してください")
	case errors.Is(err, usecase.ErrEmailAlreadyExists):
		return writeAuthError(e, 409, "EMAIL_ALREADY_EXISTS", "このメールアドレスは既に登録されています")
	case errors.Is(err, usecase.ErrInvalidCredentials):
		return writeAuthError(e, 401, "INVALID_CREDENTIALS", "メールアドレスまたはパスワードが正しくありません")
	case errors.Is(err, usecase.ErrLoginTemporarilyLocked):
		return writeAuthError(e, 429, "LOGIN_RATE_LIMITED", "時間を空けて再度お試しください")
	case errors.Is(err, usecase.ErrAccountSuspended), errors.Is(err, usecase.ErrAccountDeleted):
		return writeAuthError(e, 401, "INVALID_CREDENTIALS", "メールアドレスまたはパスワードが正しくありません")
	case errors.Is(err, usecase.ErrRefreshTokenRequired):
		return writeAuthError(e, 401, "REFRESH_TOKEN_REQUIRED", "リフレッシュトークンが必要です")
	case errors.Is(err, usecase.ErrRefreshTokenExpired):
		return writeAuthError(e, 401, "REFRESH_TOKEN_EXPIRED", "リフレッシュトークンの有効期限が切れています")
	case errors.Is(err, usecase.ErrRefreshTokenInvalid):
		return writeAuthError(e, 401, "REFRESH_TOKEN_INVALID", "リフレッシュトークンが不正です")
	case errors.Is(err, usecase.ErrRefreshTokenRevoked), errors.Is(err, usecase.ErrRefreshTokenReused):
		return writeAuthError(e, 401, "REFRESH_TOKEN_REVOKED", "リフレッシュトークンは失効しています")
	case errors.Is(err, usecase.ErrTokenOwnerMismatch):
		return writeAuthError(e, 403, "TOKEN_OWNER_MISMATCH", "トークンの所有者が一致しません")
	case errors.Is(err, usecase.ErrCSRFTokenInvalid):
		return writeAuthError(e, 403, "CSRF_TOKEN_INVALID", "CSRFトークンが不正です")
	case errors.Is(err, usecase.ErrAuthServiceUnavailable):
		return writeAuthError(e, 503, "AUTH_SERVICE_UNAVAILABLE", "認証サービスを利用できません")
	case errors.Is(err, usecase.ErrAccessTokenInvalid):
		return writeAuthError(e, 401, "ACCESS_TOKEN_INVALID", "アクセストークンが不正です")
	default:
		return writeAuthError(e, 500, "INTERNAL_ERROR", "内部エラーが発生しました")
	}
}

// writeAuthError　認証APIのエラーレスポンスを返す eはHTTPレスポンス出力時に使用
func writeAuthError(e echo.Context, status int, code, message string) error {
	return e.JSON(status, dto.ErrorResponse{Error: dto.ErrorBody{Code: code, Message: message}})
}
