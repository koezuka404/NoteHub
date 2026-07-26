package usecase

import (
	"strings"
	"unicode/utf8"
)

const maxDocumentTitleLength = 100

func normalizeTitle(title string) string {
	return strings.TrimSpace(title)
}

func validTitle(title string) error {
	title = normalizeTitle(title)
	if title == "" {
		return ErrValidation
	}
	if utf8.RuneCountInString(title) > maxDocumentTitleLength {
		return ErrValidation
	}
	return nil
}
