package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/zhouwu97/key-cabinet/server/internal/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

func (r *PostgresUserRepository) ListPendingVerification(ctx context.Context) ([]*repository.User, error) {
	var users []*repository.User
	err := r.db.WithContext(ctx).
		Where("profile_completed = ? AND identity_verified = ? AND status = ?", true, false, "ACTIVE").
		Order("updated_at ASC").Find(&users).Error
	return users, err
}

func (r *PostgresUserRepository) VerifyIdentity(ctx context.Context, userID, adminID string, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user repository.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, "id = ?", userID).Error; err != nil {
			return err
		}
		if !user.ProfileCompleted || user.Status != "ACTIVE" || user.StudentNo == "" {
			return gorm.ErrRecordNotFound
		}
		var duplicateCount int64
		if err := tx.Model(&repository.User{}).
			Where("id <> ? AND student_no = ? AND identity_verified = ?", userID, user.StudentNo, true).
			Count(&duplicateCount).Error; err != nil {
			return err
		}
		if duplicateCount > 0 {
			return repository.ErrIdentityConflict
		}
		err := tx.Model(&repository.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
			"identity_verified": true, "identity_verified_at": now.UTC(),
			"identity_verified_by": adminID, "updated_at": now.UTC(),
		}).Error
		if err != nil && strings.Contains(strings.ToLower(err.Error()), "idx_users_verified_student_no") {
			return repository.ErrIdentityConflict
		}
		return err
	})
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
