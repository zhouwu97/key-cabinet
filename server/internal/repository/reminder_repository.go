package repository

import (
	"context"
	"time"
)

type Reminder struct {
	ID             string     `gorm:"primaryKey;column:id" json:"id"`
	BorrowRecordID string     `gorm:"column:borrow_record_id;index" json:"borrowRecordId"`
	UserID         string     `gorm:"column:user_id;index" json:"userId"`
	Type           string     `gorm:"column:type" json:"type"` // APPROACHING_OVERDUE / OVERDUE
	Status         string     `gorm:"column:status" json:"status"` // PENDING / SENT / FAILED
	AttemptCount   int        `gorm:"column:attempt_count;default:1" json:"attemptCount"`
	NextRetryAt    *time.Time `gorm:"column:next_retry_at" json:"nextRetryAt,omitempty"`
	SentAt         *time.Time `gorm:"column:sent_at" json:"sentAt,omitempty"`
	Channel        string     `gorm:"column:channel" json:"channel"` // WECHAT_SUBSCRIBE
	ErrorMessage   string     `gorm:"column:error_message" json:"errorMessage,omitempty"`
	CreatedAt      time.Time  `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt      time.Time  `gorm:"column:updated_at" json:"updatedAt"`
}

func (Reminder) TableName() string {
	return "reminders"
}

type ReminderRepository interface {
	Create(ctx context.Context, reminder *Reminder) error
	Update(ctx context.Context, reminder *Reminder) error
	ExistsByTypeAndRecord(ctx context.Context, borrowRecordID, reminderType string) (bool, error)
	FindByRecordAndType(ctx context.Context, borrowRecordID, reminderType string) (*Reminder, error)
	FindPendingOrFailedRetries(ctx context.Context, now time.Time, maxAttempts int) ([]*Reminder, error)
	FindByUserID(ctx context.Context, userID string) ([]*Reminder, error)
}
