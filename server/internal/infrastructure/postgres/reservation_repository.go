package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/zhouwu97/key-cabinet/server/internal/repository"
	"gorm.io/gorm"
)

type PostgresReservationRepository struct {
	db *gorm.DB
}

func NewReservationRepository(db *gorm.DB) repository.ReservationRepository {
	return &PostgresReservationRepository{db: db}
}

func (r *PostgresReservationRepository) Create(ctx context.Context, reservation *repository.Reservation) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(reservation).Error; err != nil {
			return err
		}
		if !isReservationActiveStatus(reservation.Status) {
			return nil
		}
		return tx.Model(&repository.Key{}).
			Where("id = ? AND status = ?", reservation.KeyID, "AVAILABLE").
			Update("status", "RESERVED").Error
	})
}

func (r *PostgresReservationRepository) FindByID(ctx context.Context, id string) (*repository.Reservation, error) {
	var reservation repository.Reservation
	err := r.reservationQuery(ctx).Where("reservations.id = ?", id).First(&reservation).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &reservation, nil
}

func (r *PostgresReservationRepository) FindByUserID(ctx context.Context, userID string) ([]*repository.Reservation, error) {
	return r.List(ctx, repository.ReservationListFilter{UserID: userID})
}

func (r *PostgresReservationRepository) List(ctx context.Context, filter repository.ReservationListFilter) ([]*repository.Reservation, error) {
	var reservations []*repository.Reservation
	query := r.reservationQuery(ctx)
	if filter.UserID != "" {
		query = query.Where("reservations.user_id = ?", filter.UserID)
	}
	if filter.Status != "" {
		query = query.Where("reservations.status = ?", filter.Status)
	}
	if err := query.Order("reservations.created_at DESC").Find(&reservations).Error; err != nil {
		return nil, err
	}
	return reservations, nil
}

func (r *PostgresReservationRepository) FindConflicts(ctx context.Context, keyID string, startTime, endTime time.Time) ([]*repository.Reservation, error) {
	var reservations []*repository.Reservation
	err := r.db.WithContext(ctx).
		Where("key_id = ?", keyID).
		Where("status IN ?", []string{"PENDING", "APPROVED", "ACTIVE"}).
		Where("tstzrange(pickup_window_start, expected_return_at) && tstzrange(?, ?)", startTime.UTC(), endTime.UTC()).
		Order("created_at DESC").Find(&reservations).Error
	return reservations, err
}

func (r *PostgresReservationRepository) Update(ctx context.Context, reservation *repository.Reservation) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&repository.Reservation{}).Where("id = ?", reservation.ID).Updates(map[string]interface{}{
			"status":              reservation.Status,
			"pickup_window_start": reservation.PickupWindowStart,
			"pickup_window_end":   reservation.PickupWindowEnd,
			"expected_return_at":  reservation.ExpectedReturnAt,
			"purpose":             reservation.Purpose,
			"approved_at":         reservation.ApprovedAt,
			"used_at":             reservation.UsedAt,
			"cancelled_at":        reservation.CancelledAt,
			"updated_at":          reservation.UpdatedAt,
		}).Error; err != nil {
			return err
		}

		if reservation.Status != "CANCELLED" {
			return nil
		}
		var activeCount int64
		if err := tx.Model(&repository.Reservation{}).
			Where("key_id = ? AND status IN ?", reservation.KeyID, []string{"PENDING", "APPROVED", "ACTIVE"}).
			Count(&activeCount).Error; err != nil {
			return err
		}
		if activeCount == 0 {
			return tx.Model(&repository.Key{}).
				Where("id = ? AND status = ?", reservation.KeyID, "RESERVED").
				Update("status", "AVAILABLE").Error
		}
		return nil
	})
}

func (r *PostgresReservationRepository) reservationQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).
		Table("reservations").
		Select(`reservations.*, keys.name AS key_name, keys.room_no AS room_no,
			keys.device_id AS device_id, devices.name AS device_name`).
		Joins("LEFT JOIN keys ON keys.id = reservations.key_id").
		Joins("LEFT JOIN devices ON devices.id = keys.device_id")
}

func isReservationActiveStatus(status string) bool {
	return status == "PENDING" || status == "APPROVED" || status == "ACTIVE"
}
