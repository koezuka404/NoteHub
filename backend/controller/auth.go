package controller

import (
	"errors"
	"net/http"
	"time"

	"github.com/koezuka404/notehub/dto"
	"github.com/koezuka404/notehub/usecase"
	"github.com/labstack/echo/v4"
)

type AuthCookieConfig struct {
	RefreshName string
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
	return ctx.JSON(http.StatusOK, dto.Response{Data: dto.RefreshResponse{AccessToken: o.AccessToken, TokenType: o.TokenType, ExpiresAt: o.ExpiresAt}})
}

func (c *AuthController) setRefreshTokenCookie(ctx echo.Context, refresh string) {
	ctx.SetCookie(&http.Cookie{Name: c.cookies.RefreshName, Value: refresh, Path: "/api/auth", Domain: c.cookies.Domain, MaxAge: int(c.cookies.RefreshTTL.Seconds()), HttpOnly: true, Secure: c.cookies.Secure, SameSite: parseSameSite(c.cookies.SameSite)})
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
	default:
		return writeAuthError(ctx, 500, "INTERNAL_ERROR", "内部エラーが発生しました")
	}
}

func writeAuthError(ctx echo.Context, status int, code, message string) error {
	return ctx.JSON(status, dto.ErrorResponse{Error: dto.ErrorBody{Code: code, Message: message}})
}
