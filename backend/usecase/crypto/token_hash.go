package crypto

import (
	"crypto/sha256"
	"encoding/hex"
)

type TokenHashService struct{}

func NewTokenHashService() *TokenHashService { return &TokenHashService{} }

func (s *TokenHashService) Hash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
