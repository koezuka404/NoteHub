package crypto

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestPasswordService_HashAndCompare(t *testing.T) {
	svc := NewPasswordService(bcrypt.MinCost)
	hash, err := svc.Hash("Pass1234")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if err := svc.Compare(hash, "Pass1234"); err != nil {
		t.Fatalf("Compare match: %v", err)
	}
	if err := svc.Compare(hash, "wrong"); err == nil {
		t.Fatal("expected compare mismatch error")
	}
}

func TestPasswordService_HashError(t *testing.T) {
	svc := NewPasswordService(999)
	if _, err := svc.Hash("Pass1234"); err == nil {
		t.Fatal("expected hash error for invalid cost")
	}
}
