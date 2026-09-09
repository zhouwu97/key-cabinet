package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/zhouwu97/key-cabinet/server/internal/infrastructure/wechat"
	"github.com/zhouwu97/key-cabinet/server/internal/repository"
)

type fakeReminderRepository struct {
	reminders []*repository.Reminder
}

func (r *fakeReminderRepository) Create(_ context.Context, reminder *repository.Reminder) error {
	r.reminders = append(r.reminders, reminder)
	return nil
}

func (r *fakeReminderRepository) Update(_ context.Context, reminder *repository.Reminder) error {
	for i, rem := range r.reminders {
		if rem.ID == reminder.ID {
			r.reminders[i] = reminder
			return nil
		}
	}
	r.reminders = append(r.reminders, reminder)
	return nil
}

func (r *fakeReminderRepository) ExistsByTypeAndRecord(_ context.Context, recordID, reminderType string) (bool, error) {
	for _, rem := range r.reminders {
		if rem.BorrowRecordID == recordID && rem.Type == reminderType && rem.Status == "SENT" {
			return true, nil
		}
	}
	return false, nil
}

func (r *fakeReminderRepository) FindPendingOrFailedRetries(_ context.Context, now time.Time, maxAttempts int) ([]*repository.Reminder, error) {
	var retries []*repository.Reminder
	for _, rem := range r.reminders {
		if rem.Status == "FAILED" && rem.AttemptCount < maxAttempts {
			if rem.NextRetryAt == nil || rem.NextRetryAt.Before(now) || rem.NextRetryAt.Equal(now) {
				retries = append(retries, rem)
			}
		}
	}
	return retries, nil
}

func (r *fakeReminderRepository) FindByUserID(_ context.Context, userID string) ([]*repository.Reminder, error) {
	var results []*repository.Reminder
	for _, rem := range r.reminders {
		if rem.UserID == userID {
			results = append(results, rem)
		}
	}
	return results, nil
}

type fakeBorrowRepoForReminder struct {
	records []*repository.BorrowRecord
}

func (r *fakeBorrowRepoForReminder) Create(_ context.Context, _ *repository.BorrowRecord) error {
	return nil
}
func (r *fakeBorrowRepoForReminder) FindByID(_ context.Context, id string) (*repository.BorrowRecord, error) {
	for _, rec := range r.records {
		if rec.ID == id {
			return rec, nil
		}
	}
	return nil, nil
}
func (r *fakeBorrowRepoForReminder) FindByReservationID(_ context.Context, _ string) (*repository.BorrowRecord, error) {
	return nil, nil
}
func (r *fakeBorrowRepoForReminder) FindByUserID(_ context.Context, _ string) ([]*repository.BorrowRecord, error) {
	return nil, nil
}
func (r *fakeBorrowRepoForReminder) List(_ context.Context, _ repository.BorrowListFilter) ([]*repository.BorrowRecord, error) {
	return r.records, nil
}
func (r *fakeBorrowRepoForReminder) FindActiveByKeyID(_ context.Context, _ string) (*repository.BorrowRecord, error) {
	return nil, nil
}
func (r *fakeBorrowRepoForReminder) Update(_ context.Context, _ *repository.BorrowRecord) error {
	return nil
}
func (r *fakeBorrowRepoForReminder) MarkOverdue(_ context.Context, _ time.Time) error {
	return nil
}

func TestBorrowService_CheckOverdue_DispatchesOverdueReminder(t *testing.T) {
	now := time.Now().UTC()
	past := now.Add(-1 * time.Hour)

	borrowRepo := &fakeBorrowRepoForReminder{
		records: []*repository.BorrowRecord{
			{
				ID:               "bor_overdue_1",
				UserID:           "u1",
				KeyID:            "k1",
				KeyName:          "101主钥匙",
				Status:           "BORROWED",
				ExpectedReturnAt: past,
			},
		},
	}
	reminderRepo := &fakeReminderRepository{}
	wechatClient := wechat.NewClient("mock_appid", "mock_secret", true)

	svc := NewBorrowServiceWithReminders(borrowRepo, reminderRepo, wechatClient, nil)

	err := svc.CheckOverdue(context.Background(), now)
	require.NoError(t, err)

	require.Len(t, reminderRepo.reminders, 1)
	require.Equal(t, "OVERDUE", reminderRepo.reminders[0].Type)
	require.Equal(t, "bor_overdue_1", reminderRepo.reminders[0].BorrowRecordID)

	// 第二次检查防重发
	err = svc.CheckOverdue(context.Background(), now)
	require.NoError(t, err)
	require.Len(t, reminderRepo.reminders, 1, "应该防重发，不应追加重复提醒")
}

func TestBorrowService_CheckOverdue_DispatchesApproachingReminder(t *testing.T) {
	now := time.Now().UTC()
	soon := now.Add(15 * time.Minute) // 15分钟后到期

	borrowRepo := &fakeBorrowRepoForReminder{
		records: []*repository.BorrowRecord{
			{
				ID:               "bor_approaching_1",
				UserID:           "u2",
				KeyID:            "k2",
				KeyName:          "102室备用钥匙",
				Status:           "BORROWED",
				ExpectedReturnAt: soon,
			},
		},
	}
	reminderRepo := &fakeReminderRepository{}
	wechatClient := wechat.NewClient("mock_appid", "mock_secret", true)

	svc := NewBorrowServiceWithReminders(borrowRepo, reminderRepo, wechatClient, nil)

	err := svc.CheckOverdue(context.Background(), now)
	require.NoError(t, err)

	require.Len(t, reminderRepo.reminders, 1)
	require.Equal(t, "APPROACHING_OVERDUE", reminderRepo.reminders[0].Type)
	require.Equal(t, "bor_approaching_1", reminderRepo.reminders[0].BorrowRecordID)
}
