package usecase

import (
	"strings"
	"unicode/utf8"
)

const maxDocumentTitleLength = 100
const maxDocumentContentBytes = 512 * 1024

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

func validDocumentContent(content string) error {
	if len([]byte(content)) > maxDocumentContentBytes {
		return ErrDocumentContentTooLarge
	}
	return nil
}
