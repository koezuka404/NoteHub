package crypto

import "testing"

func TestTokenHashService_Hash(t *testing.T) {
	svc := NewTokenHashService()
	got := svc.Hash("hello")
	want := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if got != want {
		t.Fatalf("Hash() = %q, want %q", got, want)
	}
}
