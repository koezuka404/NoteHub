package crypto

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type AccessTokenClaims struct {
	UserID      uuid.UUID
	JTI         uuid.UUID
	ExpiresAt   time.Time
	AuthVersion uint
}

type JWTService struct {
	secret    []byte
	issuer    string
	audience  string
	accessTTL time.Duration
}

func NewJWTService(secret, issuer, audience string, accessTTL time.Duration) (*JWTService, error) {
	if len([]byte(secret)) < 32 {
		return nil, errors.New("JWT secret must be at least 32 bytes")
	}
	if strings.TrimSpace(issuer) == "" {
		return nil, errors.New("JWT issuer is required")
	}
	if strings.TrimSpace(audience) == "" {
		return nil, errors.New("JWT audience is required")
	}
	if accessTTL <= 0 {
		return nil, errors.New("access token TTL must be greater than zero")
	}
	return &JWTService{secret: []byte(secret), issuer: strings.TrimSpace(issuer), audience: strings.TrimSpace(audience), accessTTL: accessTTL}, nil
}

func (s *JWTService) GenerateAccessToken(userID uuid.UUID, authVersion uint, now time.Time) (string, time.Time, error) {
	expiresAt := now.Add(s.accessTTL)
	claims := jwt.MapClaims{
		"sub": userID.String(), "jti": uuid.NewString(), "iss": s.issuer,
		"aud": s.audience, "iat": now.Unix(), "nbf": now.Unix(), "exp": expiresAt.Unix(),
		"ver": authVersion,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}
	return signedToken, expiresAt, nil
}

func (s *JWTService) ValidateAccessToken(raw string, now time.Time) (AccessTokenClaims, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return AccessTokenClaims{}, errors.New("access token is empty")
	}
	parsed, err := jwt.Parse(raw, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing algorithm")
		}
		return s.secret, nil
	}, jwt.WithIssuer(s.issuer), jwt.WithAudience(s.audience), jwt.WithExpirationRequired(), jwt.WithTimeFunc(func() time.Time { return now }))
	if err != nil {
		return AccessTokenClaims{}, err
	}
	if !parsed.Valid {
		return AccessTokenClaims{}, errors.New("access token is invalid")
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return AccessTokenClaims{}, errors.New("access token claims are invalid")
	}

	subject, err := claims.GetSubject()
	if err != nil {
		return AccessTokenClaims{}, err
	}
	userID, err := uuid.Parse(subject)
	if err != nil || userID == uuid.Nil {
		return AccessTokenClaims{}, errors.New("invalid subject")
	}
	jtiRaw, ok := claims["jti"].(string)
	if !ok {
		return AccessTokenClaims{}, errors.New("invalid jti")
	}
	jti, err := uuid.Parse(jtiRaw)
	if err != nil || jti == uuid.Nil {
		return AccessTokenClaims{}, errors.New("invalid jti")
	}
	exp, err := claims.GetExpirationTime()
	if err != nil || exp == nil || !exp.Time.After(now) {
		return AccessTokenClaims{}, errors.New("access token expired")
	}
	version, err := uintClaim(claims["ver"])
	if err != nil {
		return AccessTokenClaims{}, err
	}
	return AccessTokenClaims{UserID: userID, JTI: jti, ExpiresAt: exp.Time.UTC(), AuthVersion: version}, nil
}

func uintClaim(value any) (uint, error) {
	switch v := value.(type) {
	case float64:
		if v < 0 || v != float64(uint(v)) {
			return 0, errors.New("invalid auth version")
		}
		return uint(v), nil
	case int:
		if v < 0 {
			return 0, errors.New("invalid auth version")
		}
		return uint(v), nil
	default:
		return 0, errors.New("invalid auth version")
	}
}
