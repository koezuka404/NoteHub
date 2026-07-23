package usecase

import (
	"net/mail"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	maxNameLength     = 50
	minPasswordLength = 8
	maxPasswordLength = 15
)

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
