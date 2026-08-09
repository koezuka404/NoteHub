package crypto

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testSecret = "01234567890123456789012345678901"

func TestNewJWTService_Validation(t *testing.T) {
	if _, err := NewJWTService("short", "issuer", "aud", time.Minute); err == nil {
		t.Fatal("expected secret length error")
	}
	if _, err := NewJWTService(testSecret, " ", "aud", time.Minute); err == nil {
		t.Fatal("expected issuer error")
	}
	if _, err := NewJWTService(testSecret, "issuer", " ", time.Minute); err == nil {
		t.Fatal("expected audience error")
	}
	if _, err := NewJWTService(testSecret, "issuer", "aud", 0); err == nil {
		t.Fatal("expected ttl error")
	}
}

func TestJWTService_GenerateAndValidate(t *testing.T) {
	svc, err := NewJWTService(testSecret, "notehub", "notehub-client", 15*time.Minute)
	if err != nil {
		t.Fatalf("NewJWTService: %v", err)
	}
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	userID := uuid.New()

	token, expiresAt, err := svc.GenerateAccessToken(userID, 3, now)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}
	if expiresAt.Before(now) {
		t.Fatal("expiresAt should be in the future")
	}

	claims, err := svc.ValidateAccessToken(token, now)
	if err != nil {
		t.Fatalf("ValidateAccessToken: %v", err)
	}
	if claims.UserID != userID || claims.AuthVersion != 3 {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func jwtBaseClaims(now time.Time, userID uuid.UUID) jwt.MapClaims {
	return jwt.MapClaims{
		"sub": userID.String(),
		"jti": uuid.NewString(),
		"iss": "notehub",
		"aud": "notehub-client",
		"iat": now.Unix(),
		"nbf": now.Unix(),
		"exp": now.Add(time.Hour).Unix(),
		"ver": float64(1),
	}
}

func signMapClaims(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("SignedString: %v", err)
	}
	return signed
}

func TestJWTService_ValidateAccessToken_Errors(t *testing.T) {
	svc, _ := NewJWTService(testSecret, "notehub", "notehub-client", time.Minute)
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)

	if _, err := svc.ValidateAccessToken(" ", now); err == nil {
		t.Fatal("expected empty token error")
	}
	if _, err := svc.ValidateAccessToken("not-a-jwt", now); err == nil {
		t.Fatal("expected parse error")
	}

	userID := uuid.New()
	expiredToken, _, _ := svc.GenerateAccessToken(userID, 1, now.Add(-2*time.Hour))
	if _, err := svc.ValidateAccessToken(expiredToken, now); err == nil {
		t.Fatal("expected expired token error")
	}

	wrongSecretSvc, _ := NewJWTService("98765432109876543210987654321098", "notehub", "notehub-client", time.Minute)
	token, _, _ := svc.GenerateAccessToken(userID, 1, now)
	if _, err := wrongSecretSvc.ValidateAccessToken(token, now); err == nil {
		t.Fatal("expected invalid signature")
	}

	badAlgToken := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"sub": userID.String(), "jti": uuid.NewString(), "iss": "notehub", "aud": "notehub-client",
		"iat": now.Unix(), "nbf": now.Unix(), "exp": now.Add(time.Hour).Unix(), "ver": float64(1),
	})
	unsigned, _ := badAlgToken.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if _, err := svc.ValidateAccessToken(unsigned, now); err == nil {
		t.Fatal("expected algorithm error")
	}
}

func TestJWTService_ValidateAccessToken_ClaimBranches(t *testing.T) {
	svc, _ := NewJWTService(testSecret, "notehub", "notehub-client", time.Minute)
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	userID := uuid.New()

	regToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "notehub",
		Audience:  jwt.ClaimStrings{"notehub-client"},
		Subject:   userID.String(),
		ID:        uuid.NewString(),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
	})
	regSigned, err := regToken.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign registered claims: %v", err)
	}
	if _, err := svc.ValidateAccessToken(regSigned, now); !errors.Is(err, errors.New("access token claims are invalid")) && err == nil {
		if err == nil {
			t.Fatal("expected invalid claims type error")
		}
	}

	claims := jwtBaseClaims(now, userID)
	delete(claims, "sub")
	if _, err := svc.ValidateAccessToken(signMapClaims(t, claims), now); err == nil {
		t.Fatal("expected subject error")
	}

	claims = jwtBaseClaims(now, userID)
	claims["sub"] = "not-a-uuid"
	if _, err := svc.ValidateAccessToken(signMapClaims(t, claims), now); err == nil {
		t.Fatal("expected invalid subject")
	}

	claims = jwtBaseClaims(now, uuid.Nil)
	if _, err := svc.ValidateAccessToken(signMapClaims(t, claims), now); err == nil {
		t.Fatal("expected nil uuid subject")
	}

	claims = jwtBaseClaims(now, userID)
	delete(claims, "jti")
	if _, err := svc.ValidateAccessToken(signMapClaims(t, claims), now); err == nil {
		t.Fatal("expected invalid jti")
	}

	claims = jwtBaseClaims(now, userID)
	claims["jti"] = float64(1)
	if _, err := svc.ValidateAccessToken(signMapClaims(t, claims), now); err == nil {
		t.Fatal("expected non-string jti")
	}

	claims = jwtBaseClaims(now, userID)
	claims["jti"] = "not-a-uuid"
	if _, err := svc.ValidateAccessToken(signMapClaims(t, claims), now); err == nil {
		t.Fatal("expected bad jti uuid")
	}

	claims = jwtBaseClaims(now, userID)
	claims["jti"] = uuid.Nil.String()
	if _, err := svc.ValidateAccessToken(signMapClaims(t, claims), now); err == nil {
		t.Fatal("expected nil jti uuid")
	}

	claims = jwtBaseClaims(now, userID)
	delete(claims, "exp")
	if _, err := svc.ValidateAccessToken(signMapClaims(t, claims), now); err == nil {
		t.Fatal("expected expiration error")
	}

	claims = jwtBaseClaims(now, userID)
	claims["exp"] = now.Add(-time.Hour).Unix()
	if _, err := svc.ValidateAccessToken(signMapClaims(t, claims), now); err == nil {
		t.Fatal("expected expired exp claim")
	}

	claims = jwtBaseClaims(now, userID)
	claims["ver"] = "bad"
	if _, err := svc.ValidateAccessToken(signMapClaims(t, claims), now); err == nil {
		t.Fatal("expected invalid auth version")
	}
}

func TestUintClaim(t *testing.T) {
	if v, err := uintClaim(float64(2)); err != nil || v != 2 {
		t.Fatalf("float64 claim: v=%d err=%v", v, err)
	}
	if v, err := uintClaim(int(4)); err != nil || v != 4 {
		t.Fatalf("int claim: v=%d err=%v", v, err)
	}
	if _, err := uintClaim(float64(-1)); err == nil {
		t.Fatal("expected negative float error")
	}
	if _, err := uintClaim(float64(1.5)); err == nil {
		t.Fatal("expected non-integer float error")
	}
	if _, err := uintClaim(int(-1)); err == nil {
		t.Fatal("expected negative int error")
	}
	if _, err := uintClaim("bad"); err == nil {
		t.Fatal("expected default type error")
	}
}

func TestJWTService_GenerateAccessToken_SignError(t *testing.T) {
	orig := accessTokenSigner
	accessTokenSigner = func(*jwt.Token, []byte) (string, error) {
		return "", fmt.Errorf("sign failed")
	}
	t.Cleanup(func() { accessTokenSigner = orig })

	svc, _ := NewJWTService(testSecret, "notehub", "notehub-client", time.Minute)
	_, _, err := svc.GenerateAccessToken(uuid.New(), 1, time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC))
	if err == nil || !strings.Contains(err.Error(), "sign access token") {
		t.Fatalf("expected sign error, got %v", err)
	}
}

func TestJWTService_ValidateAccessToken_HookBranches(t *testing.T) {
	svc, _ := NewJWTService(testSecret, "notehub", "notehub-client", time.Minute)
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	userID := uuid.New()

	t.Run("invalid token valid flag", func(t *testing.T) {
		orig := parseAccessTokenFn
		parseAccessTokenFn = func(string, jwt.Keyfunc, ...jwt.ParserOption) (*jwt.Token, error) {
			return &jwt.Token{Valid: false, Claims: jwt.MapClaims{}}, nil
		}
		t.Cleanup(func() { parseAccessTokenFn = orig })
		if _, err := svc.ValidateAccessToken("token", now); err == nil || err.Error() != "access token is invalid" {
			t.Fatalf("expected invalid token error, got %v", err)
		}
	})

	t.Run("get expiration error", func(t *testing.T) {
		claims := jwtBaseClaims(now, userID)
		claims["exp"] = "not-a-number"
		if _, err := svc.ValidateAccessToken(signMapClaims(t, claims), now); err == nil {
			t.Fatal("expected expiration parse error")
		}
	})

	t.Run("nil expiration time", func(t *testing.T) {
		orig := parseAccessTokenFn
		parseAccessTokenFn = func(string, jwt.Keyfunc, ...jwt.ParserOption) (*jwt.Token, error) {
			return &jwt.Token{
				Valid: true,
				Claims: jwt.MapClaims{
					"sub": userID.String(), "jti": uuid.NewString(), "ver": float64(1),
					"exp": nil,
				},
			}, nil
		}
		t.Cleanup(func() { parseAccessTokenFn = orig })
		if _, err := svc.ValidateAccessToken("token", now); err == nil || err.Error() != "access token expired" {
			t.Fatalf("expected expired error, got %v", err)
		}
	})

	t.Run("non map claims", func(t *testing.T) {
		orig := parseAccessTokenFn
		parseAccessTokenFn = func(string, jwt.Keyfunc, ...jwt.ParserOption) (*jwt.Token, error) {
			return &jwt.Token{Valid: true, Claims: jwt.RegisteredClaims{Subject: userID.String()}}, nil
		}
		t.Cleanup(func() { parseAccessTokenFn = orig })
		if _, err := svc.ValidateAccessToken("token", now); err == nil || err.Error() != "access token claims are invalid" {
			t.Fatalf("expected invalid claims type, got %v", err)
		}
	})

	t.Run("get subject error", func(t *testing.T) {
		orig := parseAccessTokenFn
		parseAccessTokenFn = func(string, jwt.Keyfunc, ...jwt.ParserOption) (*jwt.Token, error) {
			return &jwt.Token{Valid: true, Claims: jwt.MapClaims{"sub": []string{userID.String()}}}, nil
		}
		t.Cleanup(func() { parseAccessTokenFn = orig })
		if _, err := svc.ValidateAccessToken("token", now); err == nil {
			t.Fatal("expected get subject error")
		}
	})
}
