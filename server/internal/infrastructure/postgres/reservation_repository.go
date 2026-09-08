package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/zhouwu97/key-cabinet/server/internal/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
			"reviewed_by":         reservation.ReviewedBy,
			"reviewed_at":         reservation.ReviewedAt,
			"rejection_reason":    reservation.RejectionReason,
			"updated_at":          reservation.UpdatedAt,
		}).Error; err != nil {
			return err
		}

		if reservation.Status != "CANCELLED" && reservation.Status != "REJECTED" && reservation.Status != "EXPIRED" {
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

func (r *PostgresReservationRepository) Review(ctx context.Context, id, adminID string, approved bool, reason string, now time.Time) error {
	expired := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var reservation repository.Reservation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&reservation, "id = ?", id).Error; err != nil {
			return err
		}
		if reservation.Status != "PENDING" {
			return repository.ErrOperationInvalidState
		}
		if !now.Before(reservation.PickupWindowEnd) {
			if err := tx.Model(&repository.Reservation{}).Where("id = ?", id).
				Updates(map[string]interface{}{"status": "EXPIRED", "updated_at": now.UTC()}).Error; err != nil {
				return err
			}
			if err := releaseKeyIfUnreserved(tx, reservation.KeyID); err != nil {
				return err
			}
			expired = true
			return nil
		}
		var user repository.User
		if err := tx.First(&user, "id = ?", reservation.UserID).Error; err != nil {
			return err
		}
		if user.Status != "ACTIVE" || !user.IdentityVerified {
			return repository.ErrOperationInvalidState
		}
		status := "REJECTED"
		values := map[string]interface{}{
			"status": status, "reviewed_by": adminID, "reviewed_at": now.UTC(),
			"rejection_reason": reason, "updated_at": now.UTC(),
		}
		if approved {
			status = "APPROVED"
			values["status"] = status
			values["approved_at"] = now.UTC()
			values["rejection_reason"] = ""
		}
		if err := tx.Model(&repository.Reservation{}).Where("id = ? AND status = ?", id, "PENDING").Updates(values).Error; err != nil {
			return err
		}
		if approved {
			return nil
		}
		return releaseKeyIfUnreserved(tx, reservation.KeyID)
	})
	if err == nil && expired {
		return repository.ErrOperationInvalidState
	}
	return err
}

func (r *PostgresReservationRepository) ExpireBefore(ctx context.Context, now time.Time) (int64, error) {
	var expiredCount int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var reservations []*repository.Reservation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("status IN ? AND pickup_window_end < ?", []string{"PENDING", "APPROVED", "ACTIVE"}, now.UTC()).
			Find(&reservations).Error; err != nil {
			return err
		}
		for _, reservation := range reservations {
			result := tx.Model(&repository.Reservation{}).
				Where("id = ? AND status IN ?", reservation.ID, []string{"PENDING", "APPROVED", "ACTIVE"}).
				Updates(map[string]interface{}{"status": "EXPIRED", "updated_at": now.UTC()})
			if result.Error != nil {
				return result.Error
			}
			expiredCount += result.RowsAffected
			if result.RowsAffected > 0 {
				if err := releaseKeyIfUnreserved(tx, reservation.KeyID); err != nil {
					return err
				}
			}
		}
		return nil
	})
	return expiredCount, err
}

func releaseKeyIfUnreserved(tx *gorm.DB, keyID string) error {
	var activeCount int64
	if err := tx.Model(&repository.Reservation{}).
		Where("key_id = ? AND status IN ?", keyID, []string{"PENDING", "APPROVED", "ACTIVE"}).
		Count(&activeCount).Error; err != nil {
		return err
	}
	if activeCount == 0 {
		return tx.Model(&repository.Key{}).
			Where("id = ? AND status = ?", keyID, "RESERVED").
			Update("status", "AVAILABLE").Error
	}
	return nil
}

func (r *PostgresReservationRepository) reservationQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).
		Table("reservations").
		Select(`reservations.*, keys.name AS key_name, keys.room_no AS room_no,
			keys.device_id AS device_id, devices.name AS device_name,
			users.name AS user_name, users.student_no AS student_no`).
		Joins("LEFT JOIN keys ON keys.id = reservations.key_id").
		Joins("LEFT JOIN devices ON devices.id = keys.device_id").
		Joins("LEFT JOIN users ON users.id = reservations.user_id")
}

func isReservationActiveStatus(status string) bool {
	return status == "PENDING" || status == "APPROVED" || status == "ACTIVE"
}
