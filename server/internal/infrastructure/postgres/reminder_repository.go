package postgres

import (
	"context"
	"errors"

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

func (r *PostgresReminderRepository) ExistsByTypeAndRecord(ctx context.Context, borrowRecordID, reminderType string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&repository.Reminder{}).
		Where("borrow_record_id = ? AND type = ?", borrowRecordID, reminderType).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
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
