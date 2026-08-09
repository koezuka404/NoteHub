package crypto

import (
	"fmt"
	"testing"
)

func TestRandomTokenService_GeneratesTokens(t *testing.T) {
	svc := NewRandomTokenService()
	refresh, err := svc.GenerateRefreshToken()
	if err != nil || refresh == "" {
		t.Fatalf("GenerateRefreshToken: refresh=%q err=%v", refresh, err)
	}
	csrf, err := svc.GenerateCSRFToken()
	if err != nil || csrf == "" {
		t.Fatalf("GenerateCSRFToken: csrf=%q err=%v", csrf, err)
	}
	if refresh == csrf {
		t.Fatal("tokens should differ")
	}
}

func TestGenerateRandomToken_NilReaderFallback(t *testing.T) {
	original := randomReader
	randomReader = nil
	t.Cleanup(func() { randomReader = original })

	token, err := generateRandomToken(16)
	if err != nil || token == "" {
		t.Fatalf("expected success with nil reader fallback, got token=%q err=%v", token, err)
	}
}

func TestGenerateRandomToken_RandReadError(t *testing.T) {
	original := randomReader
	randomReader = func([]byte) (int, error) { return 0, fmt.Errorf("rand failed") }
	t.Cleanup(func() { randomReader = original })

	_, err := generateRandomToken(16)
	if err == nil {
		t.Fatal("expected rand read error")
	}
}
