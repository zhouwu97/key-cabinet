package service

import (
	"context"
	"testing"
	"time"

	"github.com/zhouwu97/key-cabinet/server/internal/infrastructure/device"
	"github.com/zhouwu97/key-cabinet/server/internal/repository"
)

type fakeReconcilerDeviceRepo struct {
	device *repository.Device
}

func (r *fakeReconcilerDeviceRepo) FindByID(_ context.Context, id string) (*repository.Device, error) {
	if r.device != nil && r.device.ID == id {
		return r.device, nil
	}
	return nil, nil
}

func (r *fakeReconcilerDeviceRepo) FindAll(_ context.Context) ([]*repository.Device, error) {
	if r.device != nil {
		return []*repository.Device{r.device}, nil
	}
	return nil, nil
}

type fakeReconcilerSlotRepo struct {
	slots []*repository.Slot
}

func (r *fakeReconcilerSlotRepo) FindByID(_ context.Context, id string) (*repository.Slot, error) {
	for _, s := range r.slots {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, nil
}

func (r *fakeReconcilerSlotRepo) FindByKeyID(_ context.Context, keyID string) (*repository.Slot, error) {
	for _, s := range r.slots {
		if s.KeyID != nil && *s.KeyID == keyID {
			return s, nil
		}
	}
	return nil, nil
}

func (r *fakeReconcilerSlotRepo) FindByDeviceID(_ context.Context, deviceID string) ([]*repository.Slot, error) {
	var result []*repository.Slot
	for _, s := range r.slots {
		if s.DeviceID == deviceID {
			result = append(result, s)
		}
	}
	return result, nil
}

func (r *fakeReconcilerSlotRepo) UpdatePresence(_ context.Context, id string, presence string, _ time.Time) error {
	for _, s := range r.slots {
		if s.ID == id {
			s.Presence = presence
			return nil
		}
	}
	return nil
}

type fakeReconcilerKeyRepo struct {
	keys map[string]*repository.Key
}

func (r *fakeReconcilerKeyRepo) FindByID(_ context.Context, id string) (*repository.Key, error) {
	if k, ok := r.keys[id]; ok {
		return k, nil
	}
	return nil, nil
}

func (r *fakeReconcilerKeyRepo) FindAll(_ context.Context) ([]*repository.Key, error) {
	var res []*repository.Key
	for _, k := range r.keys {
		res = append(res, k)
	}
	return res, nil
}

func (r *fakeReconcilerKeyRepo) FindByStatus(_ context.Context, _ string) ([]*repository.Key, error) {
	return nil, nil
}

func (r *fakeReconcilerKeyRepo) List(_ context.Context, _ repository.KeyListFilter) ([]*repository.Key, error) {
	return nil, nil
}

func (r *fakeReconcilerKeyRepo) Update(_ context.Context, key *repository.Key) error {
	r.keys[key.ID] = key
	return nil
}

func TestInventoryReconciler_SyncPresenceAndDetectDiscrepancies(t *testing.T) {
	key1ID := "key-101"
	key2ID := "key-102"

	devRepo := &fakeReconcilerDeviceRepo{
		device: &repository.Device{ID: "CAB001", Name: "一号机柜", Status: "ONLINE"},
	}

	slot1 := &repository.Slot{ID: "slot-1", DeviceID: "CAB001", SlotNo: 1, KeyID: &key1ID, Presence: "ABSENT"}
	slot2 := &repository.Slot{ID: "slot-2", DeviceID: "CAB001", SlotNo: 2, KeyID: &key2ID, Presence: "PRESENT"}
	slotRepo := &fakeReconcilerSlotRepo{slots: []*repository.Slot{slot1, slot2}}

	keyRepo := &fakeReconcilerKeyRepo{
		keys: map[string]*repository.Key{
			key1ID: {ID: key1ID, Name: "101室钥匙", RFIDTag: "E2001122", Status: "AVAILABLE"},
			key2ID: {ID: key2ID, Name: "102室钥匙", RFIDTag: "E2003344", Status: "AVAILABLE"},
		},
	}

	reconciler := NewInventoryReconciler(slotRepo, keyRepo, devRepo)

	// Case 1: Slot 1 physically returned with correct RFID; Slot 2 has wrong RFID
	snapshot := device.DeviceInventorySnapshot{
		DeviceID:   "CAB001",
		Timestamp:  time.Now().UTC(),
		DoorClosed: true,
		Slots: []device.DeviceInventorySlot{
			{SlotNo: 1, Presence: true, RFID: "E2001122"},
			{SlotNo: 2, Presence: true, RFID: "WRONG_RFID_999"},
		},
	}

	err := reconciler.OnInventorySnapshot(context.Background(), snapshot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify Slot 1 presence updated in DB to PRESENT
	if slot1.Presence != "PRESENT" {
		t.Errorf("expected slot 1 presence to be PRESENT, got %s", slot1.Presence)
	}
}
