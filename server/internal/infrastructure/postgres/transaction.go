package postgres

import (
	"context"
	"gorm.io/gorm"
)

// TransactionFunc is a function that will be executed within a transaction
type TransactionFunc func(tx *gorm.DB) error

// WithTransaction executes a function within a database transaction
func WithTransaction(db *gorm.DB, fn TransactionFunc) error {
	return db.Transaction(fn)
}

// WithTransactionContext executes a function within a database transaction with context
func WithTransactionContext(ctx context.Context, db *gorm.DB, fn TransactionFunc) error {
	return db.WithContext(ctx).Transaction(fn)
}
