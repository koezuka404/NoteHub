package crypto

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTService struct {
	secret    []byte
	issuer    string
	audience  string
	accessTTL time.Duration
}

func NewJWTService(
	secret string,
	issuer string,
	audience string,
	accessTTL time.Duration,
) (*JWTService, error) {
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
	return &JWTService{
		secret:    []byte(secret),
		issuer:    strings.TrimSpace(issuer),
		audience:  strings.TrimSpace(audience),
		accessTTL: accessTTL,
	}, nil
}

func (s *JWTService) GenerateAccessToken(
	userID uuid.UUID,
	authVersion uint,
	now time.Time,
) (string, time.Time, error) {
	expiresAt := now.Add(s.accessTTL)
	claims := jwt.MapClaims{
		"sub":          userID.String(),
		"jti":          uuid.NewString(),
		"iss":          s.issuer,
		"aud":          s.audience,
		"iat":          now.Unix(),
		"nbf":          now.Unix(),
		"exp":          expiresAt.Unix(),
		"auth_version": authVersion,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}
	return signedToken, expiresAt, nil
}
