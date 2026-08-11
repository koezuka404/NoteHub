package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

type RandomTokenService struct{}

func NewRandomTokenService() *RandomTokenService { return &RandomTokenService{} }

func (s *RandomTokenService) GenerateRefreshToken() (string, error) {
	return generateRandomToken(64)
}

func (s *RandomTokenService) GenerateCSRFToken() (string, error) {
	return generateRandomToken(32)
}

func generateRandomToken(size int) (string, error) {
	value := make([]byte, size)
	readRandom := randomReader
	if readRandom == nil {
		readRandom = rand.Read
	}
	if _, err := readRandom(value); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

var randomReader = rand.Read
