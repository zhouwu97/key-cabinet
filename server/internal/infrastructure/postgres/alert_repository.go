package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/zhouwu97/key-cabinet/server/internal/repository"
	"gorm.io/gorm"
)

type PostgresAlertRepository struct {
	db *gorm.DB
}

func NewAlertRepository(db *gorm.DB) repository.AlertRepository {
	return &PostgresAlertRepository{db: db}
}

func (r *PostgresAlertRepository) Create(ctx context.Context, alert *repository.DeviceAlert) error {
	return r.db.WithContext(ctx).Table("device_alerts").Create(alert).Error
}

func (r *PostgresAlertRepository) ResolveBySlotAndType(ctx context.Context, deviceID string, slotNo int, alertType string, resolvedAt time.Time) error {
	return r.db.WithContext(ctx).Table("device_alerts").
		Where("device_id = ? AND slot_no = ? AND type = ? AND status = 'OPEN'", deviceID, slotNo, alertType).
		Updates(map[string]interface{}{
			"status":      "RESOLVED",
			"resolved_at": resolvedAt.UTC(),
		}).Error
}

func (r *PostgresAlertRepository) ResolveAllBySlot(ctx context.Context, deviceID string, slotNo int, resolvedAt time.Time) error {
	return r.db.WithContext(ctx).Table("device_alerts").
		Where("device_id = ? AND slot_no = ? AND status = 'OPEN'", deviceID, slotNo).
		Updates(map[string]interface{}{
			"status":      "RESOLVED",
			"resolved_at": resolvedAt.UTC(),
		}).Error
}

func (r *PostgresAlertRepository) FindOpenBySlotAndType(ctx context.Context, deviceID string, slotNo int, alertType string) (*repository.DeviceAlert, error) {
	var alert repository.DeviceAlert
	err := r.db.WithContext(ctx).Table("device_alerts").
		Where("device_id = ? AND slot_no = ? AND type = ? AND status = 'OPEN'", deviceID, slotNo, alertType).
		First(&alert).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &alert, nil
}

func (r *PostgresAlertRepository) ListOpen(ctx context.Context, deviceID string) ([]*repository.DeviceAlert, error) {
	var alerts []*repository.DeviceAlert
	q := r.db.WithContext(ctx).Table("device_alerts").Where("status = 'OPEN'")
	if deviceID != "" {
		q = q.Where("device_id = ?", deviceID)
	}
	if err := q.Order("detected_at DESC").Find(&alerts).Error; err != nil {
		return nil, err
	}
	return alerts, nil
}
