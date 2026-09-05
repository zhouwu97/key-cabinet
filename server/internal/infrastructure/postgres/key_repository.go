package postgres

import (
	"context"
	"errors"
	"strings"

	"github.com/zhouwu97/key-cabinet/server/internal/repository"
	"gorm.io/gorm"
)

type PostgresKeyRepository struct {
	db *gorm.DB
}

func NewKeyRepository(db *gorm.DB) repository.KeyRepository {
	return &PostgresKeyRepository{db: db}
}

func (r *PostgresKeyRepository) FindByID(ctx context.Context, id string) (*repository.Key, error) {
	var key repository.Key
	err := r.keyQuery(ctx).Where("keys.id = ?", id).First(&key).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &key, nil
}

func (r *PostgresKeyRepository) FindAll(ctx context.Context) ([]*repository.Key, error) {
	return r.List(ctx, repository.KeyListFilter{})
}

func (r *PostgresKeyRepository) FindByStatus(ctx context.Context, status string) ([]*repository.Key, error) {
	return r.List(ctx, repository.KeyListFilter{Status: status})
}

func (r *PostgresKeyRepository) List(ctx context.Context, filter repository.KeyListFilter) ([]*repository.Key, error) {
	var keys []*repository.Key
	query := r.keyQuery(ctx)

	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		pattern := "%" + escapeLike(keyword) + "%"
		query = query.Where(`(
			keys.name ILIKE ? ESCAPE '\' OR
			keys.key_number ILIKE ? ESCAPE '\' OR
			keys.room_no ILIKE ? ESCAPE '\' OR
			COALESCE(keys.building, '') ILIKE ? ESCAPE '\' OR
			COALESCE(keys.description, '') ILIKE ? ESCAPE '\'
		)`, pattern, pattern, pattern, pattern, pattern)
	}
	if filter.DeviceID != "" {
		query = query.Where("keys.device_id = ?", filter.DeviceID)
	}
	if filter.Status != "" {
		query = query.Where("keys.status = ?", filter.Status)
	}
	if filter.Enabled != nil {
		query = query.Where("keys.enabled = ?", *filter.Enabled)
	}

	if err := query.Order("keys.key_number ASC").Find(&keys).Error; err != nil {
		return nil, err
	}
	return keys, nil
}

func (r *PostgresKeyRepository) Update(ctx context.Context, key *repository.Key) error {
	return r.db.WithContext(ctx).Model(&repository.Key{}).Where("id = ?", key.ID).Updates(map[string]interface{}{
		"name":              key.Name,
		"rfid_tag":          key.RFIDTag,
		"device_id":         key.DeviceID,
		"slot_id":           key.SlotID,
		"category":          key.Category,
		"room_no":           key.RoomNo,
		"building":          key.Building,
		"description":       key.Description,
		"status":            key.Status,
		"enabled":           key.Enabled,
		"requires_approval": key.RequiresApproval,
	}).Error
}

func (r *PostgresKeyRepository) keyQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).
		Table("keys").
		Select("keys.*, devices.name AS device_name").
		Joins("LEFT JOIN devices ON devices.id = keys.device_id")
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\`+`\`)
	value = strings.ReplaceAll(value, "%", `\%`)
	return strings.ReplaceAll(value, "_", `\_`)
}
