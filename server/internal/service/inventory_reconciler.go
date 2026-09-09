package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/zhouwu97/key-cabinet/server/internal/infrastructure/device"
	"github.com/zhouwu97/key-cabinet/server/internal/repository"
)

type InventoryDiscrepancyType string

const (
	DiscrepancyWrongKeyInSlot InventoryDiscrepancyType = "WRONG_KEY_IN_SLOT"
	DiscrepancyMissingKey     InventoryDiscrepancyType = "MISSING_KEY"
	DiscrepancyUnexpectedKey  InventoryDiscrepancyType = "UNEXPECTED_KEY_IN_SLOT"
	DiscrepancyUnregisteredKey InventoryDiscrepancyType = "UNREGISTERED_KEY"
)

type InventoryDiscrepancy struct {
	DeviceID     string                   `json:"deviceId"`
	SlotNo       int                      `json:"slotNo"`
	Type         InventoryDiscrepancyType `json:"type"`
	ExpectedRFID string                   `json:"expectedRfid,omitempty"`
	ActualRFID   string                   `json:"actualRfid,omitempty"`
	KeyID        string                   `json:"keyId,omitempty"`
	Message      string                   `json:"message"`
	DetectedAt   time.Time                `json:"detectedAt"`
}

type InventoryReconciler struct {
	slotRepo   repository.SlotRepository
	keyRepo    repository.KeyRepository
	deviceRepo repository.DeviceRepository
}

func NewInventoryReconciler(
	slotRepo repository.SlotRepository,
	keyRepo repository.KeyRepository,
	deviceRepo repository.DeviceRepository,
) *InventoryReconciler {
	return &InventoryReconciler{
		slotRepo:   slotRepo,
		keyRepo:    keyRepo,
		deviceRepo: deviceRepo,
	}
}

// OnInventorySnapshot implements device.DeviceInventorySink interface.
func (r *InventoryReconciler) OnInventorySnapshot(ctx context.Context, snapshot device.DeviceInventorySnapshot) error {
	if snapshot.DeviceID == "" {
		return fmt.Errorf("device id is empty in inventory snapshot")
	}

	// 1. 验证设备是否存在
	dev, err := r.deviceRepo.FindByID(ctx, snapshot.DeviceID)
	if err != nil {
		return fmt.Errorf("failed to query device %s: %w", snapshot.DeviceID, err)
	}
	if dev == nil {
		log.Printf("[InventoryReconciler] received inventory for unregistered device %s, skipping reconciliation", snapshot.DeviceID)
		return nil
	}

	// 2. 获取数据库中该设备配置的所有物理槽位
	dbSlots, err := r.slotRepo.FindByDeviceID(ctx, snapshot.DeviceID)
	if err != nil {
		return fmt.Errorf("failed to query slots for device %s: %w", snapshot.DeviceID, err)
	}

	slotMap := make(map[int]*repository.Slot, len(dbSlots))
	for _, s := range dbSlots {
		slotMap[s.SlotNo] = s
	}

	timestamp := snapshot.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}

	var discrepancies []InventoryDiscrepancy

	// 3. 对齐物理槽位上报与数据库状态
	for _, pSlot := range snapshot.Slots {
		dbSlot, exists := slotMap[pSlot.SlotNo]
		if !exists {
			log.Printf("[InventoryReconciler] device %s reported slot #%d not defined in database", snapshot.DeviceID, pSlot.SlotNo)
			continue
		}

		targetPresence := "ABSENT"
		if pSlot.Presence {
			targetPresence = "PRESENT"
		}

		// 3.1 若微动在位状态发生物理变化，同步更新槽位物理在位标记
		if dbSlot.Presence != targetPresence {
			if err := r.slotRepo.UpdatePresence(ctx, dbSlot.ID, targetPresence, timestamp); err != nil {
				log.Printf("[InventoryReconciler] failed to update presence for slot %s (#%d) on %s: %v",
					dbSlot.ID, pSlot.SlotNo, snapshot.DeviceID, err)
			} else {
				log.Printf("[InventoryReconciler] ✅ 已同步物理在位状态 设备=%s 槽位=#%d %s -> %s",
					snapshot.DeviceID, pSlot.SlotNo, dbSlot.Presence, targetPresence)
				dbSlot.Presence = targetPresence
			}
		}

		// 3.2 槽位绑定钥匙的 RFID 与在位逻辑对账检验
		if dbSlot.KeyID != nil && *dbSlot.KeyID != "" {
			key, err := r.keyRepo.FindByID(ctx, *dbSlot.KeyID)
			if err != nil {
				log.Printf("[InventoryReconciler] failed to query bound key %s for slot #%d: %v", *dbSlot.KeyID, pSlot.SlotNo, err)
				continue
			}
			if key == nil {
				continue
			}

			// (a) 物理有卡时进行 RFID 强比对（防错还 / 错插异常）
			if pSlot.Presence && pSlot.RFID != "" && key.RFIDTag != "" {
				if !strings.EqualFold(pSlot.RFID, key.RFIDTag) {
					disc := InventoryDiscrepancy{
						DeviceID:     snapshot.DeviceID,
						SlotNo:       pSlot.SlotNo,
						Type:         DiscrepancyWrongKeyInSlot,
						ExpectedRFID: key.RFIDTag,
						ActualRFID:   pSlot.RFID,
						KeyID:        key.ID,
						Message:      fmt.Sprintf("槽位 #%d 检测到错误钥匙 (实读 RFID=%s, 期望钥匙 [%s] RFID=%s)", pSlot.SlotNo, pSlot.RFID, key.Name, key.RFIDTag),
						DetectedAt:   timestamp,
					}
					discrepancies = append(discrepancies, disc)
					log.Printf("[InventoryReconciler] 🚨 错卡警告: %s", disc.Message)
				}
			}

			// (b) 数据库标记可用在柜，但物理槽位钥匙缺失
			if !pSlot.Presence && key.Status == "AVAILABLE" {
				disc := InventoryDiscrepancy{
					DeviceID:     snapshot.DeviceID,
					SlotNo:       pSlot.SlotNo,
					Type:         DiscrepancyMissingKey,
					ExpectedRFID: key.RFIDTag,
					KeyID:        key.ID,
					Message:      fmt.Sprintf("槽位 #%d 钥匙 [%s] 在数据库中标记可用在位，但物理微动检测为空缺", pSlot.SlotNo, key.Name),
					DetectedAt:   timestamp,
				}
				discrepancies = append(discrepancies, disc)
				log.Printf("[InventoryReconciler] ⚠️ 钥匙失位警告: %s", disc.Message)
			}

			// (c) 数据库标记已被借出，但物理槽位钥匙已被插入
			if pSlot.Presence && key.Status == "BORROWED" {
				disc := InventoryDiscrepancy{
					DeviceID:     snapshot.DeviceID,
					SlotNo:       pSlot.SlotNo,
					Type:         DiscrepancyUnexpectedKey,
					ExpectedRFID: key.RFIDTag,
					ActualRFID:   pSlot.RFID,
					KeyID:        key.ID,
					Message:      fmt.Sprintf("槽位 #%d 钥匙 [%s] 在数据库中标记借出中，但物理槽位存在钥匙", pSlot.SlotNo, key.Name),
					DetectedAt:   timestamp,
				}
				discrepancies = append(discrepancies, disc)
				log.Printf("[InventoryReconciler] ⚠️ 异常在位警告: %s", disc.Message)
			}
		} else {
			// 未绑定钥匙的空闲槽位却插有钥匙
			if pSlot.Presence && pSlot.RFID != "" {
				disc := InventoryDiscrepancy{
					DeviceID:   snapshot.DeviceID,
					SlotNo:     pSlot.SlotNo,
					Type:       DiscrepancyUnregisteredKey,
					ActualRFID: pSlot.RFID,
					Message:    fmt.Sprintf("未绑定钥匙的槽位 #%d 物理检测到插入钥匙 (RFID=%s)", pSlot.SlotNo, pSlot.RFID),
					DetectedAt: timestamp,
				}
				discrepancies = append(discrepancies, disc)
				log.Printf("[InventoryReconciler] ⚠️ 未注册钥匙警告: %s", disc.Message)
			}
		}
	}

	if len(discrepancies) > 0 {
		log.Printf("[InventoryReconciler] 设备 %s 对账完成: 发现 %d 处对账异常", snapshot.DeviceID, len(discrepancies))
	} else {
		log.Printf("[InventoryReconciler] 设备 %s 盘点对账完成: 物理与数据库一致 (DoorClosed=%v, Slots=%d)",
			snapshot.DeviceID, snapshot.DoorClosed, len(snapshot.Slots))
	}

	return nil
}
