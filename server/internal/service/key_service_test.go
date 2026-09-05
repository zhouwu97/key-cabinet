package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zhouwu97/key-cabinet/server/internal/repository"
)

type fakeKeyRepository struct {
	keys []*repository.Key
}

func (r *fakeKeyRepository) FindByID(_ context.Context, id string) (*repository.Key, error) {
	for _, key := range r.keys {
		if key.ID == id {
			return key, nil
		}
	}
	return nil, nil
}

func (r *fakeKeyRepository) FindAll(_ context.Context) ([]*repository.Key, error) {
	return r.keys, nil
}

func (r *fakeKeyRepository) FindByStatus(_ context.Context, status string) ([]*repository.Key, error) {
	var result []*repository.Key
	for _, key := range r.keys {
		if key.Status == status {
			result = append(result, key)
		}
	}
	return result, nil
}

func (r *fakeKeyRepository) List(_ context.Context, filter repository.KeyListFilter) ([]*repository.Key, error) {
	var result []*repository.Key
	for _, key := range r.keys {
		if filter.Status != "" && key.Status != filter.Status {
			continue
		}
		if filter.DeviceID != "" && key.DeviceID != filter.DeviceID {
			continue
		}
		result = append(result, key)
	}
	return result, nil
}

func (r *fakeKeyRepository) Update(_ context.Context, _ *repository.Key) error {
	return nil
}

type fakeSlotRepository struct {
	slot *repository.Slot
}

func (r *fakeSlotRepository) FindByID(_ context.Context, id string) (*repository.Slot, error) {
	if r.slot != nil && r.slot.ID == id {
		return r.slot, nil
	}
	return nil, nil
}

func (r *fakeSlotRepository) FindByKeyID(_ context.Context, keyID string) (*repository.Slot, error) {
	if r.slot != nil && r.slot.KeyID != nil && *r.slot.KeyID == keyID {
		return r.slot, nil
	}
	return nil, nil
}

func (r *fakeSlotRepository) FindByDeviceID(_ context.Context, _ string) ([]*repository.Slot, error) {
	if r.slot == nil {
		return nil, nil
	}
	return []*repository.Slot{r.slot}, nil
}

func TestKeyServiceListsKeysWithSupportedFilters(t *testing.T) {
	service := NewKeyService(&fakeKeyRepository{keys: []*repository.Key{
		{ID: "KEY-1", DeviceID: "CAB-1", Status: "AVAILABLE"},
		{ID: "KEY-2", DeviceID: "CAB-2", Status: "BORROWED"},
	}}, &fakeSlotRepository{})

	keys, err := service.ListKeys(context.Background(), KeyListFilter{
		DeviceID: "CAB-1",
		Status:   "AVAILABLE",
	})

	require.NoError(t, err)
	require.Len(t, keys, 1)
	require.Equal(t, "KEY-1", keys[0].ID)
}

func TestKeyServiceResolvesSlotByKeyID(t *testing.T) {
	keyID := "KEY-1"
	service := NewKeyService(&fakeKeyRepository{}, &fakeSlotRepository{
		slot: &repository.Slot{ID: "SLOT-1", KeyID: &keyID, Presence: "PRESENT"},
	})

	slot, err := service.GetKeySlot(context.Background(), keyID)

	require.NoError(t, err)
	require.NotNil(t, slot)
	require.Equal(t, "SLOT-1", slot.ID)
}

func TestKeyServiceRejectsUnsupportedStatus(t *testing.T) {
	service := NewKeyService(&fakeKeyRepository{}, &fakeSlotRepository{})

	_, err := service.ListKeys(context.Background(), KeyListFilter{Status: "UNKNOWN_STATUS"})

	require.Error(t, err)
}
