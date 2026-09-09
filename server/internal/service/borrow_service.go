package service

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/zhouwu97/key-cabinet/server/internal/infrastructure/wechat"
	apperrors "github.com/zhouwu97/key-cabinet/server/internal/platform/errors"
	"github.com/zhouwu97/key-cabinet/server/internal/repository"
)

type BorrowService interface {
	ListUserBorrowRecords(ctx context.Context, userID, keyID, status string) ([]*repository.BorrowRecord, error)
	GetBorrowRecord(ctx context.Context, id string) (*repository.BorrowRecord, error)
	GetUserBorrowRecord(ctx context.Context, userID, id string) (*repository.BorrowRecord, error)
	GetActiveBorrowByKey(ctx context.Context, keyID string) (*repository.BorrowRecord, error)
	CreateBorrowing(ctx context.Context, userID, keyID, deviceID, slotID, reservationID, purpose string, expectedReturnAt time.Time) (*repository.BorrowRecord, error)
	MarkBorrowed(ctx context.Context, id string, borrowedAt time.Time) error
	BeginReturn(ctx context.Context, userID, id string) (*repository.BorrowRecord, error)
	CompleteReturn(ctx context.Context, id string, returnedAt time.Time) error
	MarkOperationFailed(ctx context.Context, id, note string) error
	CheckOverdue(ctx context.Context, now time.Time) error
}

type borrowService struct {
	borrowRepo   repository.BorrowRepository
	reminderRepo repository.ReminderRepository
	wechatClient wechat.Client
	userRepo     repository.UserRepository
}

func NewBorrowService(borrowRepo repository.BorrowRepository) BorrowService {
	return &borrowService{borrowRepo: borrowRepo}
}

func NewBorrowServiceWithReminders(
	borrowRepo repository.BorrowRepository,
	reminderRepo repository.ReminderRepository,
	wechatClient wechat.Client,
	userRepo repository.UserRepository,
) BorrowService {
	return &borrowService{
		borrowRepo:   borrowRepo,
		reminderRepo: reminderRepo,
		wechatClient: wechatClient,
		userRepo:     userRepo,
	}
}

func (s *borrowService) ListUserBorrowRecords(ctx context.Context, userID, keyID, status string) ([]*repository.BorrowRecord, error) {
	userID = strings.TrimSpace(userID)
	keyID = strings.TrimSpace(keyID)
	status = strings.ToUpper(strings.TrimSpace(status))
	if userID == "" {
		return nil, apperrors.New(apperrors.CodeUnauthorized, "user identity is required")
	}
	if status != "" && !isBorrowStatus(status) {
		return nil, apperrors.Newf(apperrors.CodeInvalidInput, "unsupported borrow status: %s", status)
	}
	records, err := s.borrowRepo.List(ctx, repository.BorrowListFilter{UserID: userID, KeyID: keyID, Status: status})
	if err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query borrow records")
	}
	return records, nil
}

func (s *borrowService) GetBorrowRecord(ctx context.Context, id string) (*repository.BorrowRecord, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, apperrors.New(apperrors.CodeInvalidInput, "borrow record id is required")
	}
	record, err := s.borrowRepo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query borrow record")
	}
	if record == nil {
		return nil, apperrors.New(apperrors.CodeNotFound, "borrow record not found")
	}
	return record, nil
}

func (s *borrowService) GetUserBorrowRecord(ctx context.Context, userID, id string) (*repository.BorrowRecord, error) {
	record, err := s.GetBorrowRecord(ctx, id)
	if err != nil {
		return nil, err
	}
	if record.UserID != strings.TrimSpace(userID) {
		return nil, apperrors.New(apperrors.CodeNotFound, "borrow record not found")
	}
	return record, nil
}

func (s *borrowService) GetActiveBorrowByKey(ctx context.Context, keyID string) (*repository.BorrowRecord, error) {
	keyID = strings.TrimSpace(keyID)
	if keyID == "" {
		return nil, apperrors.New(apperrors.CodeInvalidInput, "key id is required")
	}
	record, err := s.borrowRepo.FindActiveByKeyID(ctx, keyID)
	if err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query active borrow")
	}
	return record, nil
}

func (s *borrowService) CreateBorrowing(ctx context.Context, userID, keyID, deviceID, slotID, reservationID, purpose string, expectedReturnAt time.Time) (*repository.BorrowRecord, error) {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(keyID) == "" || strings.TrimSpace(deviceID) == "" || strings.TrimSpace(slotID) == "" || expectedReturnAt.IsZero() {
		return nil, apperrors.New(apperrors.CodeInvalidInput, "borrow record fields are incomplete")
	}
	if reservationID != "" {
		existing, err := s.borrowRepo.FindByReservationID(ctx, reservationID)
		if err != nil {
			return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query reservation borrow record")
		}
		if existing != nil {
			return existing, nil
		}
	}
	active, err := s.borrowRepo.FindActiveByKeyID(ctx, keyID)
	if err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query active borrow")
	}
	if active != nil {
		return nil, apperrors.New(apperrors.CodeConflict, "key already has an active borrow record")
	}
	now := time.Now().UTC()
	record := &repository.BorrowRecord{
		ID:               "bor_" + generateUUID()[:12],
		ReservationID:    strings.TrimSpace(reservationID),
		UserID:           strings.TrimSpace(userID),
		KeyID:            strings.TrimSpace(keyID),
		DeviceID:         strings.TrimSpace(deviceID),
		SlotID:           strings.TrimSpace(slotID),
		Status:           "BORROWING",
		ExpectedReturnAt: expectedReturnAt.UTC(),
		Purpose:          strings.TrimSpace(purpose),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.borrowRepo.Create(ctx, record); err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to create borrow record")
	}
	return record, nil
}

func (s *borrowService) MarkBorrowed(ctx context.Context, id string, borrowedAt time.Time) error {
	record, err := s.GetBorrowRecord(ctx, id)
	if err != nil {
		return err
	}
	if record.Status != "BORROWING" {
		return apperrors.New(apperrors.CodeInvalidState, "borrow record is not waiting for pickup")
	}
	if borrowedAt.IsZero() {
		borrowedAt = time.Now().UTC()
	}
	record.Status = "BORROWED"
	record.BorrowedAt = &borrowedAt
	record.UpdatedAt = time.Now().UTC()
	if err := s.borrowRepo.Update(ctx, record); err != nil {
		return apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to mark borrow record borrowed")
	}
	return nil
}

func (s *borrowService) BeginReturn(ctx context.Context, userID, id string) (*repository.BorrowRecord, error) {
	record, err := s.GetUserBorrowRecord(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if record.Status != "BORROWED" {
		return nil, apperrors.New(apperrors.CodeInvalidState, "borrow record cannot be returned in its current state")
	}
	record.Status = "RETURNING"
	record.UpdatedAt = time.Now().UTC()
	if err := s.borrowRepo.Update(ctx, record); err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to begin return")
	}
	return record, nil
}

func (s *borrowService) CompleteReturn(ctx context.Context, id string, returnedAt time.Time) error {
	record, err := s.GetBorrowRecord(ctx, id)
	if err != nil {
		return err
	}
	if record.Status != "RETURNING" {
		return apperrors.New(apperrors.CodeInvalidState, "borrow record is not being returned")
	}
	if returnedAt.IsZero() {
		returnedAt = time.Now().UTC()
	}
	record.Status = "COMPLETED"
	record.RFIDVerified = true
	record.ReturnedAt = &returnedAt
	record.UpdatedAt = time.Now().UTC()
	if err := s.borrowRepo.Update(ctx, record); err != nil {
		return apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to complete return")
	}
	return nil
}

func (s *borrowService) MarkOperationFailed(ctx context.Context, id, note string) error {
	record, err := s.GetBorrowRecord(ctx, id)
	if err != nil {
		return err
	}
	if record.Status == "COMPLETED" {
		return apperrors.New(apperrors.CodeInvalidState, "completed borrow record cannot be failed")
	}
	if record.Status == "RETURNING" {
		record.Status = "BORROWED"
	} else {
		record.Status = "EXCEPTION"
	}
	record.Notes = strings.TrimSpace(note)
	record.UpdatedAt = time.Now().UTC()
	if err := s.borrowRepo.Update(ctx, record); err != nil {
		return apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to update failed borrow record")
	}
	return nil
}

func (s *borrowService) CheckOverdue(ctx context.Context, now time.Time) error {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if err := s.borrowRepo.MarkOverdue(ctx, now); err != nil {
		return apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to mark overdue borrow records")
	}

	if s.reminderRepo != nil {
		s.dispatchReminders(ctx, now)
	}

	return nil
}

func (s *borrowService) dispatchReminders(ctx context.Context, now time.Time) {
	borrowed, err := s.borrowRepo.List(ctx, repository.BorrowListFilter{Status: "BORROWED"})
	if err != nil {
		log.Printf("[BorrowService] failed to list active borrows for reminders: %v", err)
		return
	}

	for _, record := range borrowed {
		if record == nil {
			continue
		}

		// 场景 A: 已经逾期催还 (expected_return_at < now)
		if record.ExpectedReturnAt.Before(now) {
			exists, _ := s.reminderRepo.ExistsByTypeAndRecord(ctx, record.ID, "OVERDUE")
			if !exists {
				s.sendAndRecordReminder(ctx, record, "OVERDUE", now)
			}
			continue
		}

		// 场景 B: 临期 30 分钟预警 (now <= expected_return_at <= now + 30m)
		if record.ExpectedReturnAt.Before(now.Add(30*time.Minute)) && !record.ExpectedReturnAt.Before(now) {
			exists, _ := s.reminderRepo.ExistsByTypeAndRecord(ctx, record.ID, "APPROACHING_OVERDUE")
			if !exists {
				s.sendAndRecordReminder(ctx, record, "APPROACHING_OVERDUE", now)
			}
		}
	}
}

func (s *borrowService) sendAndRecordReminder(ctx context.Context, record *repository.BorrowRecord, reminderType string, now time.Time) {
	reminder := &repository.Reminder{
		ID:             "rem_" + generateRandomID(8),
		BorrowRecordID: record.ID,
		UserID:         record.UserID,
		Type:           reminderType,
		Status:         "SENT",
		Channel:        "WECHAT_SUBSCRIBE",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if s.wechatClient != nil {
		openID := record.UserID
		if s.userRepo != nil {
			if ident, err := s.userRepo.FindIdentity(ctx, "WECHAT", record.UserID); err == nil && ident != nil {
				openID = ident.Subject
			}
		}

		keyName := record.KeyName
		if keyName == "" {
			keyName = record.KeyID
		}

		msgReq := wechat.SubscribeMessageRequest{
			ToUser:     openID,
			TemplateID: reminderType + "_TEMPLATE_ID",
			Page:       "pages/borrow-detail/borrow-detail?id=" + record.ID,
			Data: map[string]interface{}{
				"thing1": map[string]string{"value": keyName},
				"time2":  map[string]string{"value": record.ExpectedReturnAt.Format("2006-01-02 15:04")},
				"phrase3": map[string]string{"value": reminderType},
			},
		}
		if err := s.wechatClient.SendSubscribeMessage(ctx, msgReq); err != nil {
			reminder.Status = "FAILED"
			reminder.ErrorMessage = err.Error()
		}
	}

	if err := s.reminderRepo.Create(ctx, reminder); err != nil {
		log.Printf("[BorrowService] failed to save reminder: %v", err)
	}
}

func isBorrowStatus(status string) bool {
	switch status {
	case "BORROWING", "BORROWED", "RETURNING", "COMPLETED", "EXCEPTION":
		return true
	default:
		return false
	}
}
