package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/zhouwu97/key-cabinet/server/internal/repository"
	"gorm.io/gorm"
)

type PostgresDeviceRepository struct {
	db *gorm.DB
}

func NewDeviceRepository(db *gorm.DB) repository.DeviceRepository {
	return &PostgresDeviceRepository{db: db}
}

func (r *PostgresDeviceRepository) FindByID(ctx context.Context, id string) (*repository.Device, error) {
	var device repository.Device
	err := r.db.WithContext(ctx).First(&device, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &device, nil
}

func (r *PostgresDeviceRepository) FindAll(ctx context.Context) ([]*repository.Device, error) {
	var devices []*repository.Device
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&devices).Error; err != nil {
		return nil, err
	}
	return devices, nil
}

func (r *PostgresDeviceRepository) UpdateRuntimeStatus(ctx context.Context, deviceID, status string, lastSeen time.Time) error {
	return r.db.WithContext(ctx).Model(&repository.Device{}).
		Where("id = ?", deviceID).
		Updates(map[string]interface{}{
			"status": status, "last_heartbeat_at": lastSeen.UTC(), "updated_at": time.Now().UTC(),
		}).Error
}
