package usecase

import (
	"context"

	"github.com/koezuka404/notehub/repository"
	"gorm.io/gorm"
)

type ITransactionManager interface {
	WithinTransaction(ctx context.Context, fn func(context.Context) error) error
}

type TransactionManager struct {
	db *gorm.DB
}

func NewTransactionManager(db *gorm.DB) *TransactionManager {
	return &TransactionManager{db: db}
}

func (m *TransactionManager) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(repository.WithTransactionContext(ctx, tx))
	})
}
