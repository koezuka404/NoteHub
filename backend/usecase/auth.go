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
)
const (
	maxNameLength     = 50
	minPasswordLength = 8
	maxPasswordLength = 15
)

type IUserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	FindByID(ctx context.Context, id uuid.UUID) (user *entity.User, found bool, err error)
	FindByEmail(ctx context.Context, email string) (user *entity.User, found bool, err error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	IncrementAuthVersion(ctx context.Context, userID uuid.UUID, now time.Time) error
}

type IRefreshTokenRepository interface {
	Create(ctx context.Context, token *entity.RefreshToken) error
	FindByHashForUpdate(ctx context.Context, tokenHash string) (token *entity.RefreshToken, found bool, err error)
	Update(ctx context.Context, token *entity.RefreshToken) error
	RevokeFamily(ctx context.Context, familyID uuid.UUID, now time.Time) error
}

type IAuditLogRepository interface {
	Create(ctx context.Context, log *entity.AuditLog) error
}

type IAccessTokenRevocationStore interface {
	Revoke(ctx context.Context, jti uuid.UUID, ttl time.Duration) error
	IsRevoked(ctx context.Context, jti uuid.UUID) (bool, error)
}

type ILoginFailureStore interface {
	IsLocked(ctx context.Context, email string) (locked bool, retryAfter time.Duration, err error)
	RecordFailure(ctx context.Context, email string) (locked bool, retryAfter time.Duration, err error)
	Reset(ctx context.Context, email string) error
}

type IPasswordService interface {
	Hash(password string) (string, error)
	Compare(passwordHash, password string) error
}

type IAccessTokenService interface {
	GenerateAccessToken(userID uuid.UUID, authVersion uint, now time.Time) (token string, expiresAt time.Time, err error)
}

type IRandomTokenService interface {
	GenerateRefreshToken() (string, error)
	GenerateCSRFToken() (string, error)
}

type ITokenHashService interface{ Hash(token string) string }

type IAuthUsecase interface {
	Register(ctx context.Context, input RegisterInput) (*RegisterOutput, error)
	Login(ctx context.Context, input LoginInput) (*LoginOutput, error)
	Refresh(ctx context.Context, input RefreshInput) (*RefreshOutput, error)
	Logout(ctx context.Context, input LogoutInput) (*LogoutOutput, error)
	GetCurrentUser(ctx context.Context, input GetCurrentUserInput) (*GetCurrentUserOutput, error)
}

type AuthUseCase struct {
	users           IUserRepository
	refreshTokens   IRefreshTokenRepository
	auditLogs       IAuditLogRepository
	revokedAccess   IAccessTokenRevocationStore
	loginFailures   ILoginFailureStore
	transactions    ITransactionManager
	passwords       IPasswordService
	accessTokens    IAccessTokenService
	randomTokens    IRandomTokenService
	tokenHashes     ITokenHashService
	refreshTokenTTL time.Duration
	now             func() time.Time
}

func NewAuthUseCase(
	users IUserRepository,
	refreshTokens IRefreshTokenRepository,
	auditLogs IAuditLogRepository,
	revokedAccess IAccessTokenRevocationStore,
	loginFailures ILoginFailureStore,
	transactions ITransactionManager,
	passwords IPasswordService,
	accessTokens IAccessTokenService,
	randomTokens IRandomTokenService,
	tokenHashes ITokenHashService,
	refreshTokenTTL time.Duration,
) *AuthUseCase {
	return &AuthUseCase{
		users: users, refreshTokens: refreshTokens, auditLogs: auditLogs,
		revokedAccess: revokedAccess, loginFailures: loginFailures, transactions: transactions,
		passwords: passwords, accessTokens: accessTokens, randomTokens: randomTokens,
		tokenHashes: tokenHashes, refreshTokenTTL: refreshTokenTTL, now: time.Now,
	}
}

// auth_helpers.go

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validatePassword(password string) bool {
	if len(password) < minPasswordLength || len(password) > maxPasswordLength {
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

// auth_login.go

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
	if uc.loginFailures != nil {
		locked, _, err := uc.loginFailures.IsLocked(ctx, email)
		if err != nil {
			return nil, fmt.Errorf("%w: check login lock: %v", ErrAuthServiceUnavailable, err)
		}
		if locked {
			return nil, ErrLoginTemporarilyLocked
		}
	}

	user, found, err := uc.users.FindByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}
	if !found {
		_ = uc.passwords.Compare(dummyPasswordHash, input.Password)
		return nil, uc.recordLoginFailure(ctx, email)
	}
	if !user.CanAuthenticate() {
		return nil, uc.recordLoginFailure(ctx, email)
	}
	if err := uc.passwords.Compare(user.PasswordHash, input.Password); err != nil {
		return nil, uc.recordLoginFailure(ctx, email)
	}
	if uc.loginFailures != nil {
		if err := uc.loginFailures.Reset(ctx, email); err != nil {
			return nil, fmt.Errorf("%w: reset login failures: %v", ErrAuthServiceUnavailable, err)
		}
	}

	now := uc.now()
	accessToken, expiresAt, err := uc.accessTokens.GenerateAccessToken(user.ID, user.AuthVersion, now)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}
	rawRefresh, err := uc.randomTokens.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}
	refresh, err := entity.NewRefreshToken(user.ID, uc.tokenHashes.Hash(rawRefresh), uuid.New(), now.Add(uc.refreshTokenTTL), now)
	if err != nil {
		return nil, fmt.Errorf("create refresh token entity: %w", err)
	}
	refresh.IPAddress = input.IPAddress
	refresh.UserAgent = input.UserAgent
	if err := uc.refreshTokens.Create(ctx, &refresh); err != nil {
		return nil, fmt.Errorf("save refresh token: %w", err)
	}
	csrfToken, err := uc.randomTokens.GenerateCSRFToken()
	if err != nil {
		return nil, fmt.Errorf("generate csrf token: %w", err)
	}

	return &LoginOutput{
		User:        LoginUserOutput{ID: user.ID, Name: user.Name, Email: user.Email, Status: user.Status},
		AccessToken: accessToken, RefreshToken: rawRefresh, CSRFToken: csrfToken, TokenType: "Bearer", ExpiresAt: expiresAt.Format(timeFormat),
	}, nil
}

func (uc *AuthUseCase) recordLoginFailure(ctx context.Context, email string) error {
	if uc.loginFailures == nil {
		return ErrInvalidCredentials
	}
	locked, _, err := uc.loginFailures.RecordFailure(ctx, email)
	if err != nil {
		return fmt.Errorf("%w: record login failure: %v", ErrAuthServiceUnavailable, err)
	}
	if locked {
		return ErrLoginTemporarilyLocked
	}
	return ErrInvalidCredentials
}

// auth_register.go

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

	passwordHash, err := uc.passwords.Hash(input.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := uc.now().UTC()
	user, err := entity.NewUser(strings.TrimSpace(input.Name), email, passwordHash, now)
	if err != nil {
		return nil, fmt.Errorf("create user entity: %w", err)
	}

	if err := uc.users.Create(ctx, &user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	actorID := user.ID
	auditLog, err := entity.NewAuditLog(&actorID, "USER_REGISTERED", "user", &user.ID, nil, now)
	if err != nil {
		return nil, fmt.Errorf("create audit log entity: %w", err)
	}
	auditLog.IPAddress = input.IPAddress
	if err := uc.auditLogs.Create(ctx, &auditLog); err != nil {
		return nil, fmt.Errorf("create audit log: %w", err)
	}

	return &RegisterOutput{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Status:    user.Status,
		CreatedAt: user.CreatedAt.Format(timeFormat),
	}, nil
}

// auth_refresh.go

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
		current, found, err := uc.refreshTokens.FindByHashForUpdate(txCtx, uc.tokenHashes.Hash(input.RefreshToken))
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

		rawNext, err := uc.randomTokens.GenerateRefreshToken()
		if err != nil {
			return fmt.Errorf("generate next refresh token: %w", err)
		}
		next, err := entity.NewRefreshToken(user.ID, uc.tokenHashes.Hash(rawNext), current.FamilyID, now.Add(uc.refreshTokenTTL), now)
		if err != nil {
			return fmt.Errorf("create next refresh token: %w", err)
		}
		next.IPAddress = input.IPAddress
		next.UserAgent = input.UserAgent
		if err := uc.refreshTokens.Create(txCtx, &next); err != nil {
			return fmt.Errorf("save next refresh token: %w", err)
		}
		if err := current.Rotate(next.ID, now); err != nil {
			return ErrRefreshTokenRevoked
		}
		if err := uc.refreshTokens.Update(txCtx, current); err != nil {
			return fmt.Errorf("rotate current refresh token: %w", err)
		}
		access, expiresAt, err := uc.accessTokens.GenerateAccessToken(user.ID, user.AuthVersion, now)
		if err != nil {
			return fmt.Errorf("generate access token: %w", err)
		}
		csrfToken, err := uc.randomTokens.GenerateCSRFToken()
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

// auth_logout.go

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
			token, found, err := uc.refreshTokens.FindByHashForUpdate(txCtx, uc.tokenHashes.Hash(rawRefreshToken))
			if err != nil {
				return fmt.Errorf("find refresh token for logout: %w", err)
			}
			if found {
				if token.UserID != input.UserID {
					return ErrTokenOwnerMismatch
				}
				if token.Status == entity.RefreshTokenStatusActive || token.Status == entity.RefreshTokenStatusRotated {
					if err := token.Revoke(now); err != nil {
						return fmt.Errorf("revoke refresh token: %w", err)
					}
					if err := uc.refreshTokens.Update(txCtx, token); err != nil {
						return fmt.Errorf("save revoked refresh token: %w", err)
					}
				}
			}
		}

		if uc.auditLogs != nil {
			audit, err := entity.NewAuditLog(&input.UserID, "LOGOUT", "user", &input.UserID, nil, now)
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
		if err := uc.revokedAccess.Revoke(ctx, input.AccessTokenJTI, ttl); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrAuthServiceUnavailable, err)
		}
	}
	return &LogoutOutput{}, nil
}

// auth_me.go

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
