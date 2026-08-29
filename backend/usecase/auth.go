package usecase

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
	"github.com/koezuka404/notehub/repository"
)

const (
	maxNameLength     = 50
	minPasswordLength = 8
	maxPasswordLength = 15
)

type IAuthService interface {

	HashPassword(password string) (string, error)
	ComparePassword(passwordHash, password string) error

	GenerateAccessToken(userID uuid.UUID, authVersion uint, now time.Time) (token string, expiresAt time.Time, err error)

	GenerateRefreshToken() (string, error)
	GenerateCSRFToken() (string, error)

	HashToken(token string) string

	RevokeAccessToken(ctx context.Context, jti uuid.UUID, ttl time.Duration) error
	IsAccessTokenRevoked(ctx context.Context, jti uuid.UUID) (bool, error)

	IsLoginLocked(ctx context.Context, email string) (locked bool, retryAfter time.Duration, err error)
	RecordLoginFailure(ctx context.Context, email string) (locked bool, retryAfter time.Duration, err error)
	ResetLoginFailures(ctx context.Context, email string) error
}

type IAuthUsecase interface {
	Register(ctx context.Context, input RegisterInput) (*RegisterOutput, error)
	Login(ctx context.Context, input LoginInput) (*LoginOutput, error)
	Refresh(ctx context.Context, input RefreshInput) (*RefreshOutput, error)
	Logout(ctx context.Context, input LogoutInput) (*LogoutOutput, error)
	GetCurrentUser(ctx context.Context, input GetCurrentUserInput) (*GetCurrentUserOutput, error)
	IssueCSRFToken(ctx context.Context) (string, error)
}

type AuthUseCase struct {
	users           repository.UserRepository
	refreshTokens   repository.RefreshTokenRepository
	auditLogs       repository.AuditLogRepository
	auth            IAuthService
	transactions    repository.TransactionManager
	refreshTokenTTL time.Duration
	now             func() time.Time
}

func NewAuthUseCase(
	users repository.UserRepository,
	refreshTokens repository.RefreshTokenRepository,
	auditLogs repository.AuditLogRepository,
	auth IAuthService,
	transactions repository.TransactionManager,
	refreshTokenTTL time.Duration,
) *AuthUseCase {
	return &AuthUseCase{
		users: users, refreshTokens: refreshTokens, auditLogs: auditLogs,
		auth: auth, transactions: transactions,
		refreshTokenTTL: refreshTokenTTL, now: time.Now,
	}
}


func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validatePassword(password string) bool {
	if utf8.RuneCountInString(password) < minPasswordLength || utf8.RuneCountInString(password) > maxPasswordLength {
		return false
	}
	if strings.TrimSpace(password) == "" {
		return false
	}
	hasLetter, hasDigit := false, false
	for _, r := range password {
		if unicode.IsLetter(r) {
			hasLetter = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
	}
	return hasLetter && hasDigit
}

func validateRegisterInput(input RegisterInput) error {
	name := strings.TrimSpace(input.Name)
	email := normalizeEmail(input.Email)

	if utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > maxNameLength {
		return ErrValidation
	}
	if email == "" || len(email) > 255 {
		return ErrValidation
	}
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return ErrValidation
	}
	if !validatePassword(input.Password) {
		return ErrPasswordInvalid
	}
	return nil
}

func validateLoginInput(input LoginInput) bool {
	return normalizeEmail(input.Email) != "" && input.Password != ""
}



const dummyPasswordHash = "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro4llC/.og/at2.uheWG/igi"

type LoginInput struct {
	Email, Password, IPAddress, UserAgent string
}

type LoginUserOutput struct {
	ID          uuid.UUID
	Name, Email string
	Status      entity.UserStatus
}

type LoginOutput struct {
	User         LoginUserOutput
	AccessToken  string
	RefreshToken string
	CSRFToken    string
	TokenType    string
	ExpiresAt    string
}

func (uc *AuthUseCase) Login(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	if !validateLoginInput(input) {
		return nil, ErrValidation
	}

	email := normalizeEmail(input.Email)
	locked, _, err := uc.auth.IsLoginLocked(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("%w: check login lock: %v", ErrAuthServiceUnavailable, err)
	}
	if locked {
		uc.recordLoginRateLimitedAudit(ctx, nil, nil, input)
		return nil, ErrLoginTemporarilyLocked
	}

	user, found, err := uc.users.FindByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}
	if !found {
		_ = uc.auth.ComparePassword(dummyPasswordHash, input.Password)
		return nil, uc.handleAuthFailure(ctx, nil, nil, input, email, "invalid_credentials")
	}
	if !user.CanAuthenticate() {
		userID := user.ID
		return nil, uc.handleAuthFailure(ctx, &userID, &userID, input, email, "account_unavailable")
	}
	if err := uc.auth.ComparePassword(user.PasswordHash, input.Password); err != nil {
		userID := user.ID
		return nil, uc.handleAuthFailure(ctx, &userID, &userID, input, email, "invalid_credentials")
	}
	if err := uc.auth.ResetLoginFailures(ctx, email); err != nil {
		return nil, fmt.Errorf("%w: reset login failures: %v", ErrAuthServiceUnavailable, err)
	}

	now := uc.now().UTC()
	accessToken, expiresAt, err := uc.auth.GenerateAccessToken(user.ID, user.AuthVersion, now)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}
	rawRefresh, err := uc.auth.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}
	csrfToken, err := uc.auth.GenerateCSRFToken()
	if err != nil {
		return nil, fmt.Errorf("generate csrf token: %w", err)
	}

	var output *LoginOutput
	if err := uc.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		refresh, err := newRefreshTokenEntityFn(user.ID, uc.auth.HashToken(rawRefresh), uuid.New(), now.Add(uc.refreshTokenTTL), now)
		if err != nil {
			return fmt.Errorf("create refresh token entity: %w", err)
		}
		refresh.IPAddress = input.IPAddress
		refresh.UserAgent = input.UserAgent
		if err := uc.refreshTokens.Create(txCtx, &refresh); err != nil {
			return fmt.Errorf("save refresh token: %w", err)
		}
		if err := uc.recordLoginSuccessAudit(txCtx, user.ID, input, now); err != nil {
			return err
		}
		output = &LoginOutput{
			User:         LoginUserOutput{ID: user.ID, Name: user.Name, Email: user.Email, Status: user.Status},
			AccessToken:  accessToken,
			RefreshToken: rawRefresh,
			CSRFToken:    csrfToken,
			TokenType:    "Bearer",
			ExpiresAt:    expiresAt.Format(timeFormat),
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return output, nil
}

func (uc *AuthUseCase) handleAuthFailure(
	ctx context.Context,
	actorUserID *uuid.UUID,
	resourceUserID *uuid.UUID,
	input LoginInput,
	email string,
	reason string,
) error {
	locked, _, err := uc.auth.RecordLoginFailure(ctx, email)
	if err != nil {
		return fmt.Errorf("%w: record login failure: %v", ErrAuthServiceUnavailable, err)
	}
	if locked {
		uc.recordLoginRateLimitedAudit(ctx, actorUserID, resourceUserID, input)
		return ErrLoginTemporarilyLocked
	}
	uc.recordLoginFailedAudit(ctx, actorUserID, resourceUserID, input, reason)
	return ErrInvalidCredentials
}

func (uc *AuthUseCase) recordLoginSuccessAudit(ctx context.Context, userID uuid.UUID, input LoginInput, now time.Time) error {
	if uc.auditLogs == nil {
		return nil
	}
	audit, err := newAuditLogFn(&userID, "LOGIN", "user", &userID, nil, now)
	if err != nil {
		return fmt.Errorf("create login success audit log entity: %w", err)
	}
	audit.IPAddress = input.IPAddress
	audit.UserAgent = input.UserAgent
	if err := uc.auditLogs.Create(ctx, &audit); err != nil {
		return fmt.Errorf("save login success audit log: %w", err)
	}
	return nil
}

func (uc *AuthUseCase) recordLoginRateLimitedAudit(
	ctx context.Context,
	actorUserID *uuid.UUID,
	resourceUserID *uuid.UUID,
	input LoginInput,
) {
	if uc.auditLogs == nil {
		return
	}
	audit, err := newAuditLogFn(actorUserID, "LOGIN_RATE_LIMITED", "user", resourceUserID, nil, uc.now().UTC())
	if err != nil {
		return
	}
	audit.IPAddress = input.IPAddress
	audit.UserAgent = input.UserAgent
	_ = uc.auditLogs.Create(ctx, &audit)
}

func (uc *AuthUseCase) recordLoginFailedAudit(
	ctx context.Context,
	actorUserID *uuid.UUID,
	resourceUserID *uuid.UUID,
	input LoginInput,
	reason string,
) {
	if uc.auditLogs == nil {
		return
	}
	metadata := map[string]string{"reason": reason}
	audit, err := newAuditLogFn(actorUserID, "LOGIN_FAILED", "user", resourceUserID, metadata, uc.now().UTC())
	if err != nil {
		return
	}
	audit.IPAddress = input.IPAddress
	audit.UserAgent = input.UserAgent
	_ = uc.auditLogs.Create(ctx, &audit)
}


type RegisterInput struct {
	Name      string
	Email     string
	Password  string
	IPAddress string
}

type RegisterOutput struct {
	ID        uuid.UUID
	Name      string
	Email     string
	Status    entity.UserStatus
	CreatedAt string
}

func (uc *AuthUseCase) Register(ctx context.Context, input RegisterInput) (*RegisterOutput, error) {
	if err := validateRegisterInput(input); err != nil {
		return nil, err
	}

	email := normalizeEmail(input.Email)
	exists, err := uc.users.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("check email existence: %w", err)
	}
	if exists {
		return nil, ErrEmailAlreadyExists
	}

	passwordHash, err := uc.auth.HashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := uc.now().UTC()
	user, err := newUserFn(strings.TrimSpace(input.Name), email, passwordHash, now)
	if err != nil {
		return nil, fmt.Errorf("create user entity: %w", err)
	}

	var output *RegisterOutput
	if err := uc.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.users.Create(txCtx, &user); err != nil {
			return fmt.Errorf("create user: %w", err)
		}
		actorID := user.ID
		auditLog, err := newAuditLogFn(&actorID, "USER_REGISTERED", "user", &user.ID, nil, now)
		if err != nil {
			return fmt.Errorf("create audit log entity: %w", err)
		}
		auditLog.IPAddress = input.IPAddress
		if err := uc.auditLogs.Create(txCtx, &auditLog); err != nil {
			return fmt.Errorf("create audit log: %w", err)
		}
		output = &RegisterOutput{
			ID:        user.ID,
			Name:      user.Name,
			Email:     user.Email,
			Status:    user.Status,
			CreatedAt: user.CreatedAt.Format(timeFormat),
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return output, nil
}


type RefreshInput struct {
	RefreshToken string
	IPAddress    string
	UserAgent    string
}

type RefreshOutput struct {
	AccessToken  string
	RefreshToken string
	CSRFToken    string
	TokenType    string
	ExpiresAt    string
}

func (uc *AuthUseCase) Refresh(ctx context.Context, input RefreshInput) (*RefreshOutput, error) {
	if strings.TrimSpace(input.RefreshToken) == "" {
		return nil, ErrRefreshTokenRequired
	}
	now := uc.now().UTC()
	var output *RefreshOutput
	var reused bool

	err := uc.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		current, found, err := uc.refreshTokens.FindByHashForUpdate(txCtx, uc.auth.HashToken(input.RefreshToken))
		if err != nil {
			return fmt.Errorf("find refresh token: %w", err)
		}
		if !found {
			return ErrRefreshTokenInvalid
		}
		if current.IsExpired(now) {
			return ErrRefreshTokenExpired
		}
		if current.Status != entity.RefreshTokenStatusActive || current.RotatedAt != nil || current.RevokedAt != nil {
			if err := uc.refreshTokens.RevokeFamily(txCtx, current.FamilyID, now); err != nil {
				return fmt.Errorf("revoke reused token family: %w", err)
			}
			if err := uc.users.IncrementAuthVersion(txCtx, current.UserID, now); err != nil {
				return fmt.Errorf("increment auth version: %w", err)
			}
			if uc.auditLogs != nil {
				userID := current.UserID
				audit, err := newAuditLogFn(&userID, "REFRESH_TOKEN_REUSED", "user", &userID, nil, now)
				if err != nil {
					return fmt.Errorf("create refresh token reused audit log entity: %w", err)
				}
				audit.IPAddress = input.IPAddress
				audit.UserAgent = input.UserAgent
				if err := uc.auditLogs.Create(txCtx, &audit); err != nil {
					return fmt.Errorf("save refresh token reused audit log: %w", err)
				}
			}
			reused = true
			return nil
		}

		user, found, err := uc.users.FindByID(txCtx, current.UserID)
		if err != nil {
			return fmt.Errorf("find refresh token user: %w", err)
		}
		if !found {
			return ErrRefreshTokenInvalid
		}
		if user.IsDeleted() {
			return ErrAccountDeleted
		}
		if user.IsSuspended() {
			return ErrAccountSuspended
		}
		if !user.CanAuthenticate() {
			return ErrRefreshTokenRevoked
		}

		rawNext, err := uc.auth.GenerateRefreshToken()
		if err != nil {
			return fmt.Errorf("generate next refresh token: %w", err)
		}
		next, err := newRefreshTokenEntityFn(user.ID, uc.auth.HashToken(rawNext), current.FamilyID, now.Add(uc.refreshTokenTTL), now)
		if err != nil {
			return fmt.Errorf("create next refresh token: %w", err)
		}
		next.IPAddress = input.IPAddress
		next.UserAgent = input.UserAgent
		if err := uc.refreshTokens.Create(txCtx, &next); err != nil {
			return fmt.Errorf("save next refresh token: %w", err)
		}
		if err := rotateRefreshTokenFn(current, next.ID, now); err != nil {
			return ErrRefreshTokenRevoked
		}
		if err := uc.refreshTokens.Update(txCtx, current); err != nil {
			return fmt.Errorf("rotate current refresh token: %w", err)
		}
		access, expiresAt, err := uc.auth.GenerateAccessToken(user.ID, user.AuthVersion, now)
		if err != nil {
			return fmt.Errorf("generate access token: %w", err)
		}
		csrfToken, err := uc.auth.GenerateCSRFToken()
		if err != nil {
			return fmt.Errorf("generate csrf token: %w", err)
		}
		output = &RefreshOutput{AccessToken: access, RefreshToken: rawNext, CSRFToken: csrfToken, TokenType: "Bearer", ExpiresAt: expiresAt.Format(timeFormat)}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if reused {
		return nil, ErrRefreshTokenReused
	}
	if output == nil {
		return nil, fmt.Errorf("refresh token transaction completed without output")
	}
	return output, nil
}



type LogoutInput struct {
	UserID         uuid.UUID
	AccessTokenJTI uuid.UUID
	AccessTokenExp time.Time
	RefreshToken   string
	IPAddress      string
	UserAgent      string
	CSRFValidated  bool
}

type LogoutOutput struct{}

func (uc *AuthUseCase) Logout(ctx context.Context, input LogoutInput) (*LogoutOutput, error) {
	if input.UserID == uuid.Nil || input.AccessTokenJTI == uuid.Nil || input.AccessTokenExp.IsZero() {
		return nil, ErrAccessTokenInvalid
	}
	if !input.CSRFValidated {
		return nil, ErrCSRFTokenInvalid
	}

	now := uc.now().UTC()
	rawRefreshToken := strings.TrimSpace(input.RefreshToken)

	err := uc.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		if rawRefreshToken != "" {
			token, found, err := uc.refreshTokens.FindByHashForUpdate(txCtx, uc.auth.HashToken(rawRefreshToken))
			if err != nil {
				return fmt.Errorf("find refresh token for logout: %w", err)
			}
			if found {
				if token.UserID != input.UserID {
					return ErrTokenOwnerMismatch
				}
				if token.Status == entity.RefreshTokenStatusActive || token.Status == entity.RefreshTokenStatusRotated {
					if err := revokeRefreshTokenFn(token, now); err != nil {
						return fmt.Errorf("revoke refresh token: %w", err)
					}
					if err := uc.refreshTokens.Update(txCtx, token); err != nil {
						return fmt.Errorf("save revoked refresh token: %w", err)
					}
				}
			}
		}

		if uc.auditLogs != nil {
			audit, err := newAuditLogFn(&input.UserID, "LOGOUT", "user", &input.UserID, nil, now)
			if err != nil {
				return fmt.Errorf("create logout audit log: %w", err)
			}
			audit.IPAddress = input.IPAddress
			audit.UserAgent = input.UserAgent
			if err := uc.auditLogs.Create(txCtx, &audit); err != nil {
				return fmt.Errorf("save logout audit log: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrTokenOwnerMismatch) {
			return nil, err
		}
		return nil, fmt.Errorf("logout transaction: %w", err)
	}

	ttl := input.AccessTokenExp.Sub(now)
	if ttl > 0 {
		if err := uc.auth.RevokeAccessToken(ctx, input.AccessTokenJTI, ttl); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrAuthServiceUnavailable, err)
		}
	}
	return &LogoutOutput{}, nil
}



type GetCurrentUserInput struct {
	UserID uuid.UUID
}

type GetCurrentUserOutput struct {
	ID     uuid.UUID
	Name   string
	Email  string
	Status entity.UserStatus
}

func (uc *AuthUseCase) GetCurrentUser(ctx context.Context, input GetCurrentUserInput) (*GetCurrentUserOutput, error) {
	user, found, err := uc.users.FindByID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	if !found || !user.CanAuthenticate() {
		return nil, ErrAccountSuspended
	}
	return &GetCurrentUserOutput{
		ID:     user.ID,
		Name:   user.Name,
		Email:  user.Email,
		Status: user.Status,
	}, nil
}

func (uc *AuthUseCase) IssueCSRFToken(ctx context.Context) (string, error) {
	_ = ctx
	token, err := uc.auth.GenerateCSRFToken()
	if err != nil {
		return "", fmt.Errorf("generate csrf token: %w", err)
	}
	return token, nil
}
