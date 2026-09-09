package postgres

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zhouwu97/key-cabinet/server/internal/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PostgresOperationRepository struct {
	db *gorm.DB
}

func NewOperationRepository(db *gorm.DB) repository.OperationRepository {
	return &PostgresOperationRepository{db: db}
}

func (r *PostgresOperationRepository) Create(ctx context.Context, operation *repository.DeviceOperation) error {
	return r.db.WithContext(ctx).Table(operation.TableName()).Create(operationPersistenceValues(operation)).Error
}

func (r *PostgresOperationRepository) PreparePickup(ctx context.Context, borrow *repository.BorrowRecord, operation *repository.DeviceOperation) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(borrow).Error; err != nil {
			return err
		}
		return prepareOperationTx(tx, operation)
	})
}

func (r *PostgresOperationRepository) PrepareReturn(ctx context.Context, borrowRecordID, userID string, operation *repository.DeviceOperation) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := operation.CreatedAt
		result := tx.Model(&repository.BorrowRecord{}).
			Where("id = ? AND user_id = ? AND status = ?", borrowRecordID, userID, "BORROWED").
			Updates(map[string]interface{}{"status": "RETURNING", "updated_at": now})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return repository.ErrOperationInvalidState
		}
		return prepareOperationTx(tx, operation)
	})
}

func prepareOperationTx(tx *gorm.DB, operation *repository.DeviceOperation) error {
	operation.Status = "AUTHORIZED"
	if err := tx.Table(operation.TableName()).Create(operationPersistenceValues(operation)).Error; err != nil {
		return err
	}
	if err := createEventTx(tx, operation.ID, "REQUEST_RECEIVED", operation.CreatedAt); err != nil {
		return err
	}
	return createEventTx(tx, operation.ID, "AUTH_CONFIRMED", operation.CreatedAt)
}

func (r *PostgresOperationRepository) BeginExecution(ctx context.Context, operationID string, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		operation, err := lockOperation(tx, operationID)
		if err != nil {
			return err
		}
		if isTerminalOperationStatus(operation.Status) {
			return repository.ErrOperationTerminal
		}
		if operation.Status != "AUTHORIZED" {
			return repository.ErrOperationInvalidState
		}
		now = normalizedNow(now)
		result := tx.Model(&repository.DeviceOperation{}).
			Where("id = ? AND status = ?", operationID, "AUTHORIZED").
			Updates(map[string]interface{}{"status": "EXECUTING", "sent_at": now, "updated_at": now})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return repository.ErrOperationInvalidState
		}
		return createEventTx(tx, operationID, "COMMAND_DISPATCHING", now)
	})
}

func (r *PostgresOperationRepository) FindByID(ctx context.Context, id string) (*repository.DeviceOperation, error) {
	var operation repository.DeviceOperation
	err := r.db.WithContext(ctx).First(&operation, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := r.loadEvents(ctx, &operation); err != nil {
		return nil, err
	}
	return &operation, nil
}

func (r *PostgresOperationRepository) FindByRequestID(ctx context.Context, requestID string) (*repository.DeviceOperation, error) {
	var operation repository.DeviceOperation
	err := r.db.WithContext(ctx).Where("request_id = ?", requestID).First(&operation).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := r.loadEvents(ctx, &operation); err != nil {
		return nil, err
	}
	return &operation, nil
}

func (r *PostgresOperationRepository) FindByReservationID(ctx context.Context, reservationID string) ([]*repository.DeviceOperation, error) {
	var operations []*repository.DeviceOperation
	if err := r.db.WithContext(ctx).Where("reservation_id = ?", reservationID).Order("created_at DESC").Find(&operations).Error; err != nil {
		return nil, err
	}
	for _, operation := range operations {
		if err := r.loadEvents(ctx, operation); err != nil {
			return nil, err
		}
	}
	return operations, nil
}

func (r *PostgresOperationRepository) FindActiveByReservation(ctx context.Context, reservationID string) (*repository.DeviceOperation, error) {
	return r.findActive(ctx, "reservation_id = ?", reservationID)
}

func (r *PostgresOperationRepository) FindActiveByDeviceID(ctx context.Context, deviceID string) (*repository.DeviceOperation, error) {
	return r.findActive(ctx, "device_id = ?", deviceID)
}

func (r *PostgresOperationRepository) FindActiveByKeyID(ctx context.Context, keyID string) (*repository.DeviceOperation, error) {
	return r.findActive(ctx, "key_id = ?", keyID)
}

func (r *PostgresOperationRepository) FindActiveByUserID(ctx context.Context, userID string) (*repository.DeviceOperation, error) {
	return r.findActive(ctx, "user_id = ?", userID)
}

func (r *PostgresOperationRepository) FindExpired(ctx context.Context, before time.Time) ([]*repository.DeviceOperation, error) {
	var operations []*repository.DeviceOperation
	err := r.db.WithContext(ctx).
		Where("status IN ? AND COALESCE(sent_at, initiated_at, created_at) < ?", []string{"CREATED", "AUTHORIZED", "SENT", "EXECUTING"}, before.UTC()).
		Order("created_at ASC").Find(&operations).Error
	return operations, err
}

func (r *PostgresOperationRepository) Update(ctx context.Context, operation *repository.DeviceOperation) error {
	return r.db.WithContext(ctx).Model(&repository.DeviceOperation{}).Where("id = ?", operation.ID).Updates(operationPersistenceValues(operation)).Error
}

// operationPersistenceValues 将可选关联的空 ID 转成 NULL，避免 PostgreSQL 把空字符串当成外键值。
func operationPersistenceValues(operation *repository.DeviceOperation) map[string]interface{} {
	return map[string]interface{}{
		"id":               operation.ID,
		"request_id":       operation.RequestID,
		"user_id":          operation.UserID,
		"reservation_id":   nullableOperationID(operation.ReservationID),
		"borrow_record_id": nullableOperationID(operation.BorrowRecordID),
		"device_id":        nullableOperationID(operation.DeviceID),
		"slot_id":          nullableOperationID(operation.SlotID),
		"key_id":           nullableOperationID(operation.KeyID),
		"operation_type":   operation.Action,
		"status":           operation.Status,
		"scanned_rfid":     operation.ScannedRFID,
		"created_at":       operation.CreatedAt,
		"initiated_at":     operation.StartedAt,
		"sent_at":          operation.SentAt,
		"ack_at":           operation.AckAt,
		"completed_at":     operation.FinishedAt,
		"error_code":       operation.ErrorCode,
		"error_message":    operation.ErrorMessage,
		"updated_at":       time.Now().UTC(),
	}
}

func nullableOperationID(id string) interface{} {
	if id == "" {
		return nil
	}
	return id
}

func (r *PostgresOperationRepository) CreateEvent(ctx context.Context, event *repository.OperationEvent) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if event.ID == "" {
			event.ID = newPostgresID("evt")
		}
		if event.OccurredAt.IsZero() {
			event.OccurredAt = time.Now().UTC()
		}
		if event.Seq == 0 {
			var maxSeq int
			if err := tx.Model(&repository.OperationEvent{}).Where("operation_id = ?", event.OperationID).Select("COALESCE(MAX(seq), 0)").Scan(&maxSeq).Error; err != nil {
				return err
			}
			event.Seq = maxSeq + 1
		}
		return tx.Create(event).Error
	})
}

func (r *PostgresOperationRepository) AppendEventIfActive(ctx context.Context, event *repository.OperationEvent) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		operation, err := lockOperation(tx, event.OperationID)
		if err != nil {
			return err
		}
		if isTerminalOperationStatus(operation.Status) {
			return nil
		}
		if event.ID == "" {
			event.ID = newPostgresID("evt")
		}
		if event.OccurredAt.IsZero() {
			event.OccurredAt = time.Now().UTC()
		}
		var existingCount int64
		if err := tx.Model(&repository.OperationEvent{}).Where("id = ?", event.ID).Count(&existingCount).Error; err != nil {
			return err
		}
		if existingCount > 0 {
			return nil
		}
		var maxSeq int
		if err := tx.Model(&repository.OperationEvent{}).Where("operation_id = ?", event.OperationID).
			Select("COALESCE(MAX(seq), 0)").Scan(&maxSeq).Error; err != nil {
			return err
		}
		event.Seq = maxSeq + 1
		if err := tx.Create(event).Error; err != nil {
			return err
		}
		switch strings.ToUpper(event.Type) {
		case "ACK", "RECEIVED", "COMMAND_ACK":
			return tx.Model(&repository.DeviceOperation{}).Where("id = ? AND ack_at IS NULL", event.OperationID).Updates(map[string]interface{}{
				"ack_at": event.OccurredAt, "updated_at": time.Now().UTC(),
			}).Error
		default:
			return nil
		}
	})
}

func (r *PostgresOperationRepository) CompletePickup(ctx context.Context, operationID string, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		operation, err := lockOperation(tx, operationID)
		if err != nil {
			return err
		}
		if operation.Status == "SUCCESS" {
			return nil
		}
		if isTerminalOperationStatus(operation.Status) {
			return nil
		}
		if operation.Action != "PICKUP" {
			return fmt.Errorf("operation %s is not a pickup", operationID)
		}
		if operation.Status != "EXECUTING" && operation.Status != "SENT" {
			return repository.ErrOperationInvalidState
		}
		now = normalizedNow(now)

		if operation.ReservationID != "" {
			result := tx.Model(&repository.Reservation{}).
				Where("id = ? AND status IN ?", operation.ReservationID, []string{"PENDING", "APPROVED", "ACTIVE"}).
				Updates(map[string]interface{}{"status": "USED", "used_at": now, "updated_at": now})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return fmt.Errorf("reservation %s is not active", operation.ReservationID)
			}
		}
		if operation.BorrowRecordID != "" {
			result := tx.Model(&repository.BorrowRecord{}).
				Where("id = ? AND status = ?", operation.BorrowRecordID, "BORROWING").
				Updates(map[string]interface{}{"status": "BORROWED", "borrowed_at": now, "updated_at": now})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return fmt.Errorf("borrow record %s is not waiting for pickup", operation.BorrowRecordID)
			}
		}
		if err := updateKeyAndSlot(tx, operation.KeyID, operation.SlotID, "BORROWED", "ABSENT", now); err != nil {
			return err
		}
		if err := updateOperationStatus(tx, operationID, "SUCCESS", "", "", now); err != nil {
			return err
		}
		return createEventTx(tx, operationID, "SUCCESS", now)
	})
}

func (r *PostgresOperationRepository) CompleteReturn(ctx context.Context, operationID string, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		operation, err := lockOperation(tx, operationID)
		if err != nil {
			return err
		}
		if operation.Status == "SUCCESS" {
			return nil
		}
		if isTerminalOperationStatus(operation.Status) {
			return nil
		}
		if operation.Action != "RETURN" {
			return fmt.Errorf("operation %s is not a return", operationID)
		}
		if operation.Status != "EXECUTING" && operation.Status != "SENT" {
			return repository.ErrOperationInvalidState
		}
		now = normalizedNow(now)
		if operation.BorrowRecordID == "" {
			return errors.New("return operation has no borrow record")
		}

		// 严格校验 RFID 是否已确认且匹配
		var rfidEvents []repository.OperationEvent
		if err := tx.Where("operation_id = ? AND event_type = ?", operationID, "RFID_CONFIRMED").
			Order("seq desc").
			Find(&rfidEvents).Error; err != nil {
			return err
		}

		scannedRFID := strings.TrimSpace(operation.ScannedRFID)
		if len(rfidEvents) == 0 && scannedRFID == "" {
			return repository.ErrRFIDNotVerified
		}

		if scannedRFID == "" && len(rfidEvents) > 0 {
			if uid, ok := rfidEvents[0].Data["scannedRfid"].(string); ok && uid != "" {
				scannedRFID = uid
			} else if uid, ok := rfidEvents[0].Data["uid"].(string); ok && uid != "" {
				scannedRFID = uid
			} else if uid, ok := rfidEvents[0].Data["rfidTag"].(string); ok && uid != "" {
				scannedRFID = uid
			}
		}

		// 如果钥匙定义了预期 RFIDTag，校验是否匹配
		if operation.KeyID != "" {
			var key repository.Key
			if err := tx.First(&key, "id = ?", operation.KeyID).Error; err == nil {
				if key.RFIDTag != "" && scannedRFID != "" && !strings.EqualFold(scannedRFID, key.RFIDTag) {
					return repository.ErrRFIDMismatch
				}
				if scannedRFID == "" {
					scannedRFID = key.RFIDTag
				}
			}
		}

		result := tx.Model(&repository.BorrowRecord{}).
			Where("id = ? AND status = ?", operation.BorrowRecordID, "RETURNING").
			Updates(map[string]interface{}{
				"status":        "COMPLETED",
				"returned_at":   now,
				"rfid_verified": true,
				"scanned_rfid":  scannedRFID,
				"notes":         "",
				"updated_at":    now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("borrow record %s is not being returned", operation.BorrowRecordID)
		}
		if err := updateKeyAndSlot(tx, operation.KeyID, operation.SlotID, "AVAILABLE", "PRESENT", now); err != nil {
			return err
		}
		if err := tx.Model(&repository.DeviceOperation{}).Where("id = ?", operationID).Updates(map[string]interface{}{
			"status":        "SUCCESS",
			"scanned_rfid":  scannedRFID,
			"error_code":    "",
			"error_message": "",
			"ack_at":        now,
			"completed_at":  now,
			"updated_at":    now,
		}).Error; err != nil {
			return err
		}
		return createEventTx(tx, operationID, "SUCCESS", now)
	})
}

func (r *PostgresOperationRepository) Fail(ctx context.Context, operationID, errorCode, errorMessage string, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		operation, err := lockOperation(tx, operationID)
		if err != nil {
			return err
		}
		if isTerminalOperationStatus(operation.Status) {
			return nil
		}
		now = normalizedNow(now)
		if operation.BorrowRecordID != "" {
			borrowStatus := "EXCEPTION"
			if operation.Action == "RETURN" {
				borrowStatus = "BORROWED"
			}
			if err := tx.Model(&repository.BorrowRecord{}).Where("id = ?", operation.BorrowRecordID).Updates(map[string]interface{}{
				"status":     borrowStatus,
				"notes":      errorMessage,
				"updated_at": now,
			}).Error; err != nil {
				return err
			}
		}
		if err := updateOperationStatus(tx, operationID, "FAILED", errorCode, errorMessage, now); err != nil {
			return err
		}
		return createEventTx(tx, operationID, "FAILED", now)
	})
}

func (r *PostgresOperationRepository) Timeout(ctx context.Context, operationID string, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		operation, err := lockOperation(tx, operationID)
		if err != nil {
			return err
		}
		if isTerminalOperationStatus(operation.Status) {
			return nil
		}
		now = normalizedNow(now)
		if operation.BorrowRecordID != "" {
			borrowStatus := "EXCEPTION"
			if operation.Action == "RETURN" {
				borrowStatus = "BORROWED"
			}
			if err := tx.Model(&repository.BorrowRecord{}).
				Where("id = ? AND status IN ?", operation.BorrowRecordID, []string{"BORROWING", "RETURNING"}).
				Updates(map[string]interface{}{
					"status": borrowStatus, "notes": "设备操作超时", "updated_at": now,
				}).Error; err != nil {
				return err
			}
		}
		if err := updateOperationStatus(tx, operationID, "TIMEOUT", "OPERATION_TIMEOUT", "device operation timed out", now); err != nil {
			return err
		}
		return createEventTx(tx, operationID, "TIMEOUT", now)
	})
}

func (r *PostgresOperationRepository) Cancel(ctx context.Context, operationID string, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		operation, err := lockOperation(tx, operationID)
		if err != nil {
			return err
		}
		if isTerminalOperationStatus(operation.Status) {
			return repository.ErrOperationTerminal
		}
		now = normalizedNow(now)
		if operation.BorrowRecordID != "" {
			borrowStatus := "BORROWED"
			if operation.Action == "PICKUP" {
				borrowStatus = "EXCEPTION"
			}
			if err := tx.Model(&repository.BorrowRecord{}).
				Where("id = ? AND status IN ?", operation.BorrowRecordID, []string{"BORROWING", "RETURNING"}).
				Update("status", borrowStatus).Error; err != nil {
				return err
			}
		}
		if err := updateOperationStatus(tx, operationID, "CANCELLED", "", "", now); err != nil {
			return err
		}
		return createEventTx(tx, operationID, "CANCELLED", now)
	})
}

func (r *PostgresOperationRepository) findActive(ctx context.Context, condition string, value string) (*repository.DeviceOperation, error) {
	var operation repository.DeviceOperation
	err := r.db.WithContext(ctx).
		Where(condition+" AND status IN ?", value, []string{"CREATED", "AUTHORIZED", "SENT", "EXECUTING"}).
		Order("created_at DESC").First(&operation).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := r.loadEvents(ctx, &operation); err != nil {
		return nil, err
	}
	return &operation, nil
}

func (r *PostgresOperationRepository) loadEvents(ctx context.Context, operation *repository.DeviceOperation) error {
	var events []*repository.OperationEvent
	if err := r.db.WithContext(ctx).Where("operation_id = ?", operation.ID).Order("seq ASC").Find(&events).Error; err != nil {
		return err
	}
	operation.Events = events
	return nil
}

func lockOperation(tx *gorm.DB, operationID string) (*repository.DeviceOperation, error) {
	var operation repository.DeviceOperation
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&operation, "id = ?", operationID).Error; err != nil {
		return nil, err
	}
	return &operation, nil
}

func updateOperationStatus(tx *gorm.DB, operationID, status, errorCode, errorMessage string, now time.Time) error {
	values := map[string]interface{}{
		"status":        status,
		"error_code":    errorCode,
		"error_message": errorMessage,
		"ack_at":        now,
		"completed_at":  now,
		"updated_at":    now,
	}
	return tx.Model(&repository.DeviceOperation{}).Where("id = ?", operationID).Updates(values).Error
}

func updateKeyAndSlot(tx *gorm.DB, keyID, slotID, keyStatus, presence string, now time.Time) error {
	if keyID != "" {
		result := tx.Model(&repository.Key{}).Where("id = ?", keyID).Updates(map[string]interface{}{"status": keyStatus, "updated_at": now})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("key %s not found", keyID)
		}
	}
	if slotID != "" {
		result := tx.Model(&repository.Slot{}).Where("id = ?", slotID).Updates(map[string]interface{}{"presence": presence, "updated_at": now})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("slot %s not found", slotID)
		}
	}
	return nil
}

func createEventTx(tx *gorm.DB, operationID, eventType string, occurredAt time.Time) error {
	var maxSeq int
	if err := tx.Model(&repository.OperationEvent{}).Where("operation_id = ?", operationID).Select("COALESCE(MAX(seq), 0)").Scan(&maxSeq).Error; err != nil {
		return err
	}
	return tx.Create(&repository.OperationEvent{
		ID:          newPostgresID("evt"),
		OperationID: operationID,
		Seq:         maxSeq + 1,
		Type:        eventType,
		OccurredAt:  occurredAt,
	}).Error
}

func normalizedNow(now time.Time) time.Time {
	if now.IsZero() {
		return time.Now().UTC()
	}
	return now.UTC()
}

func isTerminalOperationStatus(status string) bool {
	return status == "SUCCESS" || status == "FAILED" || status == "TIMEOUT" || status == "CANCELLED"
}

func newPostgresID(prefix string) string {
	var bytes [8]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s_%x", prefix, bytes)
}
