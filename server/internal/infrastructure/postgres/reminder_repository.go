package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/zhouwu97/key-cabinet/server/internal/repository"
	"gorm.io/gorm"
)

type PostgresReminderRepository struct {
	db *gorm.DB
}

func NewReminderRepository(db *gorm.DB) repository.ReminderRepository {
	return &PostgresReminderRepository{db: db}
}

func (r *PostgresReminderRepository) Create(ctx context.Context, reminder *repository.Reminder) error {
	return r.db.WithContext(ctx).Create(reminder).Error
}

func (r *PostgresReminderRepository) Update(ctx context.Context, reminder *repository.Reminder) error {
	return r.db.WithContext(ctx).Save(reminder).Error
}

func (r *PostgresReminderRepository) ExistsByTypeAndRecord(ctx context.Context, borrowRecordID, reminderType string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&repository.Reminder{}).
		Where("borrow_record_id = ? AND type = ? AND status = ?", borrowRecordID, reminderType, "SENT").
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *PostgresReminderRepository) FindByRecordAndType(ctx context.Context, borrowRecordID, reminderType string) (*repository.Reminder, error) {
	var reminder repository.Reminder
	err := r.db.WithContext(ctx).
		Where("borrow_record_id = ? AND type = ?", borrowRecordID, reminderType).
		First(&reminder).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &reminder, nil
}

func (r *PostgresReminderRepository) FindPendingOrFailedRetries(ctx context.Context, now time.Time, maxAttempts int) ([]*repository.Reminder, error) {
	var reminders []*repository.Reminder
	err := r.db.WithContext(ctx).
		Where("status = ? AND attempt_count < ? AND (next_retry_at IS NULL OR next_retry_at <= ?)", "FAILED", maxAttempts, now).
		Order("created_at asc").
		Limit(50).
		Find(&reminders).Error
	if err != nil {
		return nil, err
	}
	return reminders, nil
}

func (r *PostgresReminderRepository) FindByUserID(ctx context.Context, userID string) ([]*repository.Reminder, error) {
	var reminders []*repository.Reminder
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at desc").
		Find(&reminders).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return reminders, err
}
