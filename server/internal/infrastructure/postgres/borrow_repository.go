package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/zhouwu97/key-cabinet/server/internal/repository"
	"gorm.io/gorm"
)

type PostgresBorrowRepository struct {
	db *gorm.DB
}

func NewBorrowRepository(db *gorm.DB) repository.BorrowRepository {
	return &PostgresBorrowRepository{db: db}
}

func (r *PostgresBorrowRepository) Create(ctx context.Context, record *repository.BorrowRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

func (r *PostgresBorrowRepository) FindByID(ctx context.Context, id string) (*repository.BorrowRecord, error) {
	var record repository.BorrowRecord
	err := r.borrowQuery(ctx).Where("borrow_records.id = ?", id).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *PostgresBorrowRepository) FindByReservationID(ctx context.Context, reservationID string) (*repository.BorrowRecord, error) {
	var record repository.BorrowRecord
	err := r.borrowQuery(ctx).Where("borrow_records.reservation_id = ?", reservationID).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *PostgresBorrowRepository) FindByUserID(ctx context.Context, userID string) ([]*repository.BorrowRecord, error) {
	return r.List(ctx, repository.BorrowListFilter{UserID: userID})
}

func (r *PostgresBorrowRepository) List(ctx context.Context, filter repository.BorrowListFilter) ([]*repository.BorrowRecord, error) {
	var records []*repository.BorrowRecord
	query := r.borrowQuery(ctx)
	if filter.UserID != "" {
		query = query.Where("borrow_records.user_id = ?", filter.UserID)
	}
	if filter.KeyID != "" {
		query = query.Where("borrow_records.key_id = ?", filter.KeyID)
	}
	if filter.Status != "" {
		query = query.Where("borrow_records.status = ?", filter.Status)
	}
	if err := query.Order("COALESCE(borrow_records.borrowed_at, borrow_records.created_at) DESC").Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func (r *PostgresBorrowRepository) FindActiveByKeyID(ctx context.Context, keyID string) (*repository.BorrowRecord, error) {
	var record repository.BorrowRecord
	err := r.borrowQuery(ctx).
		Where("borrow_records.key_id = ? AND borrow_records.status IN ?", keyID, []string{"BORROWING", "BORROWED", "RETURNING"}).
		Order("borrow_records.created_at DESC").First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *PostgresBorrowRepository) Update(ctx context.Context, record *repository.BorrowRecord) error {
	return r.db.WithContext(ctx).Model(&repository.BorrowRecord{}).Where("id = ?", record.ID).Updates(map[string]interface{}{
		"reservation_id":     record.ReservationID,
		"user_id":            record.UserID,
		"key_id":             record.KeyID,
		"device_id":          record.DeviceID,
		"slot_id":            record.SlotID,
		"status":             record.Status,
		"borrowed_at":        record.BorrowedAt,
		"expected_return_at": record.ExpectedReturnAt,
		"overdue_at":         record.OverdueAt,
		"returned_at":        record.ReturnedAt,
		"rfid_verified":      record.RFIDVerified,
		"purpose":            record.Purpose,
		"notes":              record.Notes,
		"updated_at":         record.UpdatedAt,
	}).Error
}

func (r *PostgresBorrowRepository) MarkOverdue(ctx context.Context, now time.Time) error {
	now = now.UTC()
	return r.db.WithContext(ctx).Model(&repository.BorrowRecord{}).
		Where("status = ? AND expected_return_at < ? AND overdue_at IS NULL", "BORROWED", now).
		Updates(map[string]interface{}{"overdue_at": now, "updated_at": now}).Error
}

func (r *PostgresBorrowRepository) borrowQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).
		Table("borrow_records").
		Select("borrow_records.*, keys.name AS key_name, keys.room_no AS room_no").
		Joins("LEFT JOIN keys ON keys.id = borrow_records.key_id")
}
