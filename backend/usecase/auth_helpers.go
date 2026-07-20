package usecase

import (
	"net/mail"
	"strings"
	"unicode/utf8"
)

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validateRegisterInput(input RegisterInput) bool {
	name := strings.TrimSpace(input.Name)
	email := normalizeEmail(input.Email)

	if utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 100 {
		return false
	}
	if email == "" || len(email) > 255 {
		return false
	}
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return false
	}
	return len(input.Password) >= 8 && len(input.Password) <= 72
}

func validateLoginInput(input LoginInput) bool {
	return normalizeEmail(input.Email) != "" && input.Password != ""
}
