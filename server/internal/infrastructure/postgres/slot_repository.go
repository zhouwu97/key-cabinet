package postgres

import (
	"context"
	"errors"

	"github.com/zhouwu97/key-cabinet/server/internal/repository"
	"gorm.io/gorm"
)

type PostgresSlotRepository struct {
	db *gorm.DB
}

func NewSlotRepository(db *gorm.DB) repository.SlotRepository {
	return &PostgresSlotRepository{db: db}
}

func (r *PostgresSlotRepository) FindByID(ctx context.Context, id string) (*repository.Slot, error) {
	var slot repository.Slot
	err := r.slotQuery(ctx).Where("slots.id = ?", id).First(&slot).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &slot, nil
}

func (r *PostgresSlotRepository) FindByKeyID(ctx context.Context, keyID string) (*repository.Slot, error) {
	var slot repository.Slot
	err := r.slotQuery(ctx).Where("keys.id = ?", keyID).First(&slot).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &slot, nil
}

func (r *PostgresSlotRepository) FindByDeviceID(ctx context.Context, deviceID string) ([]*repository.Slot, error) {
	var slots []*repository.Slot
	if err := r.slotQuery(ctx).Where("slots.device_id = ?", deviceID).Order("slots.slot_number ASC").Find(&slots).Error; err != nil {
		return nil, err
	}
	return slots, nil
}

func (r *PostgresSlotRepository) slotQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).
		Table("slots").
		Select("slots.*, keys.id AS key_id").
		Joins("LEFT JOIN keys ON keys.slot_id = slots.id")
}
