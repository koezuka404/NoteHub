package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/dto"
	"github.com/koezuka404/notehub/entity"
	appmiddleware "github.com/koezuka404/notehub/middleware"
	"github.com/koezuka404/notehub/usecase"
	"github.com/labstack/echo/v4"
)

func defaultAuthCookies() AuthCookieConfig {
	return AuthCookieConfig{
		RefreshName: "refresh_token",
		CSRFName:    "csrf_token",
		Domain:      "localhost",
		SameSite:    "Strict",
		Secure:      true,
		RefreshTTL:  time.Hour,
	}
}

func TestAuthController_IssueCSRF(t *testing.T) {
	ctrl := NewAuthController(&mockAuthUsecase{issueCSRFOut: "csrf-token"}, defaultAuthCookies())
	ctx, rec := newEchoContext(t, http.MethodGet, "/auth/csrf", nil)
	if err := ctrl.IssueCSRF(ctx); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var payload dto.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
}

func TestAuthController_Register(t *testing.T) {
	userID := uuid.New()
	ctrl := NewAuthController(&mockAuthUsecase{
		registerOut: &usecase.RegisterOutput{
			ID: userID, Name: "Alice", Email: "alice@example.com",
			Status: entity.UserStatusActive, CreatedAt: testFixedTime.Format(time.RFC3339),
		},
	}, defaultAuthCookies())

	e := echo.New()
	e.POST("/register", ctrl.Register)
	body := `{"name":"Alice","email":"alice@example.com","password":"Pass1234"}`
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}

	ctrl = NewAuthController(&mockAuthUsecase{registerErr: usecase.ErrValidation}, defaultAuthCookies())
	e = echo.New()
	e.POST("/register", ctrl.Register)
	req = httptest.NewRequest(http.MethodPost, "/register", strings.NewReader("not-json"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bind status = %d", rec.Code)
	}
}

func TestAuthController_Login(t *testing.T) {
	userID := uuid.New()
	ctrl := NewAuthController(&mockAuthUsecase{
		loginOut: &usecase.LoginOutput{
			User: usecase.LoginUserOutput{
				ID: userID, Name: "Alice", Email: "alice@example.com", Status: entity.UserStatusActive,
			},
			AccessToken: "access", RefreshToken: "refresh", CSRFToken: "csrf",
			TokenType: "Bearer", ExpiresAt: testFixedTime.Format(time.RFC3339),
		},
	}, defaultAuthCookies())

	e := echo.New()
	e.POST("/login", ctrl.Login)
	body := `{"email":"alice@example.com","password":"Pass1234"}`
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec.Result().Cookies()[0].Name != "refresh_token" {
		t.Fatal("expected refresh cookie")
	}

	ctrl = NewAuthController(&mockAuthUsecase{loginErr: usecase.ErrInvalidCredentials}, defaultAuthCookies())
	e = echo.New()
	e.POST("/login", ctrl.Login)
	req = httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("{bad"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bind status = %d", rec.Code)
	}
}

func TestAuthController_LoginSameSiteCookies(t *testing.T) {
	for _, sameSite := range []string{"Strict", "None", "Lax"} {
		ctrl := NewAuthController(&mockAuthUsecase{
			loginOut: &usecase.LoginOutput{
				User:         usecase.LoginUserOutput{ID: uuid.New(), Status: entity.UserStatusActive},
				RefreshToken: "refresh", CSRFToken: "csrf", TokenType: "Bearer", ExpiresAt: testFixedTime.Format(time.RFC3339),
			},
		}, AuthCookieConfig{
			RefreshName: "refresh_token", CSRFName: "csrf_token", SameSite: sameSite, RefreshTTL: time.Hour,
		})
		ctx, _ := newEchoContext(t, http.MethodPost, "/login", dto.LoginRequest{Email: "a@b.com", Password: "Pass1234"})
		_ = ctrl.Login(ctx)
	}
}

func TestAuthController_Refresh(t *testing.T) {
	ctrl := NewAuthController(&mockAuthUsecase{
		refreshOut: &usecase.RefreshOutput{
			AccessToken: "access", RefreshToken: "refresh", CSRFToken: "csrf",
			TokenType: "Bearer", ExpiresAt: testFixedTime.Format(time.RFC3339),
		},
	}, defaultAuthCookies())

	e := echo.New()
	e.POST("/refresh", ctrl.Refresh)
	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "token"})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	e = echo.New()
	e.POST("/refresh", ctrl.Refresh)
	req = httptest.NewRequest(http.MethodPost, "/refresh", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("missing cookie status = %d", rec.Code)
	}

	ctrl = NewAuthController(&mockAuthUsecase{refreshErr: usecase.ErrRefreshTokenExpired}, defaultAuthCookies())
	e = echo.New()
	e.POST("/refresh", ctrl.Refresh)
	req = httptest.NewRequest(http.MethodPost, "/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "token"})
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("refresh error status = %d", rec.Code)
	}
}

func TestAuthController_Me(t *testing.T) {
	userID := uuid.New()
	ctrl := NewAuthController(&mockAuthUsecase{
		meOut: &usecase.GetCurrentUserOutput{
			ID: userID, Name: "Alice", Email: "alice@example.com", Status: entity.UserStatusActive,
		},
	}, defaultAuthCookies())

	e := echo.New()
	e.GET("/me", func(ctx echo.Context) error {
		setAuthenticatedUser(ctx, userID)
		return ctrl.Me(ctx)
	})
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	e = echo.New()
	e.GET("/me", ctrl.Me)
	req = httptest.NewRequest(http.MethodGet, "/me", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauth status = %d", rec.Code)
	}

	ctrl = NewAuthController(&mockAuthUsecase{meErr: usecase.ErrAccessTokenInvalid}, defaultAuthCookies())
	e = echo.New()
	e.GET("/me", func(ctx echo.Context) error {
		setAuthenticatedUser(ctx, userID)
		return ctrl.Me(ctx)
	})
	req = httptest.NewRequest(http.MethodGet, "/me", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("usecase error status = %d", rec.Code)
	}
}

func TestAuthController_Logout(t *testing.T) {
	userID := uuid.New()
	jti := uuid.New()
	exp := testFixedTime.Add(time.Hour)
	ctrl := NewAuthController(&mockAuthUsecase{logoutOut: &usecase.LogoutOutput{}}, defaultAuthCookies())

	e := echo.New()
	e.POST("/logout", func(ctx echo.Context) error {
		setLogoutContext(ctx, userID, jti, exp, true)
		ctx.Request().AddCookie(&http.Cookie{Name: "refresh_token", Value: "refresh"})
		return ctrl.Logout(ctx)
	})
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	tests := []struct {
		name  string
		setup func(echo.Context)
	}{
		{"missing user", func(ctx echo.Context) {}},
		{"missing jti", func(ctx echo.Context) { setAuthenticatedUser(ctx, userID) }},
		{"missing exp", func(ctx echo.Context) {
			setAuthenticatedUser(ctx, userID)
			ctx.Set(appmiddleware.ContextAccessTokenJTI, jti)
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e = echo.New()
			e.POST("/logout", func(ctx echo.Context) error {
				tc.setup(ctx)
				return ctrl.Logout(ctx)
			})
			req = httptest.NewRequest(http.MethodPost, "/logout", nil)
			rec = httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d", rec.Code)
			}
		})
	}

	ctrl = NewAuthController(&mockAuthUsecase{logoutErr: usecase.ErrCSRFTokenInvalid}, defaultAuthCookies())
	e = echo.New()
	e.POST("/logout", func(ctx echo.Context) error {
		setLogoutContext(ctx, userID, jti, exp, false)
		return ctrl.Logout(ctx)
	})
	req = httptest.NewRequest(http.MethodPost, "/logout", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("logout error status = %d", rec.Code)
	}
}

func TestAuthController_ClearAuthCookies(t *testing.T) {
	ctrl := NewAuthController(&mockAuthUsecase{}, defaultAuthCookies())
	ctx, rec := newEchoContext(t, http.MethodPost, "/logout", nil)
	ctrl.clearAuthCookies(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) < 2 {
		t.Fatalf("cookies = %v", cookies)
	}
}

func TestAuthController_RegisterResponseShape(t *testing.T) {
	userID := uuid.New()
	ctrl := NewAuthController(&mockAuthUsecase{
		registerOut: &usecase.RegisterOutput{
			ID: userID, Name: "Alice", Email: "alice@example.com",
			Status: entity.UserStatusActive, CreatedAt: testFixedTime.Format(time.RFC3339),
		},
	}, defaultAuthCookies())
	ctx, rec := newEchoContext(t, http.MethodPost, "/register", dto.RegisterRequest{
		Name: "Alice", Email: "alice@example.com", Password: "Pass1234",
	})
	if err := ctrl.Register(ctx); err != nil {
		t.Fatal(err)
	}
	var payload dto.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
}
