package usecase

import (
	"net/mail"

	"github.com/koezuka404/notehub/entity"
)

const deletedName = "削除済みユーザー"

func validEmail(raw string) (string, error) {
	email := normalizeEmail(raw)
	if email == "" || len(email) > 255 {
		return "", ErrValidation
	}
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return "", ErrValidation
	}
	return email, nil
}

func listUser(u *entity.User) (name, email, status string) {
	if u.IsDeleted() {
		return deletedName, "", string(entity.UserStatusDeleted)
	}
	return u.Name, u.Email, string(u.Status)
}
