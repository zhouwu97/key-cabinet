package postgres

import (
	"context"
	"errors"

	"github.com/zhouwu97/key-cabinet/server/internal/repository"
	"gorm.io/gorm"
)

type PostgresUserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) repository.UserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) Create(ctx context.Context, user *repository.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *PostgresUserRepository) FindByID(ctx context.Context, id string) (*repository.User, error) {
	var user repository.User
	err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *PostgresUserRepository) FindByStudentNo(ctx context.Context, studentNo string) (*repository.User, error) {
	var user repository.User
	err := r.db.WithContext(ctx).First(&user, "student_no = ?", studentNo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *PostgresUserRepository) Update(ctx context.Context, user *repository.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *PostgresUserRepository) FindIdentity(ctx context.Context, provider, subject string) (*repository.UserIdentity, error) {
	var identity repository.UserIdentity
	err := r.db.WithContext(ctx).First(&identity, "provider = ? AND subject = ?", provider, subject).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &identity, nil
}

func (r *PostgresUserRepository) CreateIdentity(ctx context.Context, identity *repository.UserIdentity) error {
	return r.db.WithContext(ctx).Create(identity).Error
}

func (r *PostgresUserRepository) CreateWithIdentity(
	ctx context.Context,
	user *repository.User,
	identity *repository.UserIdentity,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		return tx.Create(identity).Error
	})
}
