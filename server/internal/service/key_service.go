package service

import (
	"context"
	"strings"

	apperrors "github.com/zhouwu97/key-cabinet/server/internal/platform/errors"
	"github.com/zhouwu97/key-cabinet/server/internal/repository"
)

type KeyListFilter = repository.KeyListFilter

type KeyService interface {
	ListKeys(ctx context.Context, filter KeyListFilter) ([]*repository.Key, error)
	GetKey(ctx context.Context, id string) (*repository.Key, error)
	GetKeySlot(ctx context.Context, keyOrSlotID string) (*repository.Slot, error)
}

type keyService struct {
	keyRepo  repository.KeyRepository
	slotRepo repository.SlotRepository
}

func NewKeyService(keyRepo repository.KeyRepository, slotRepo repository.SlotRepository) KeyService {
	return &keyService{keyRepo: keyRepo, slotRepo: slotRepo}
}

func (s *keyService) ListKeys(ctx context.Context, filter KeyListFilter) ([]*repository.Key, error) {
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	filter.DeviceID = strings.TrimSpace(filter.DeviceID)
	filter.Status = strings.ToUpper(strings.TrimSpace(filter.Status))
	if filter.Status != "" && !isSupportedKeyStatus(filter.Status) {
		return nil, apperrors.Newf(apperrors.CodeInvalidInput, "unsupported key status: %s", filter.Status)
	}
	keys, err := s.keyRepo.List(ctx, filter)
	if err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query keys")
	}
	return keys, nil
}

func (s *keyService) GetKey(ctx context.Context, id string) (*repository.Key, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, apperrors.New(apperrors.CodeInvalidInput, "key id is required")
	}
	key, err := s.keyRepo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query key")
	}
	if key == nil {
		return nil, apperrors.New(apperrors.CodeNotFound, "key not found")
	}
	return key, nil
}

func (s *keyService) GetKeySlot(ctx context.Context, keyOrSlotID string) (*repository.Slot, error) {
	keyOrSlotID = strings.TrimSpace(keyOrSlotID)
	if keyOrSlotID == "" {
		return nil, apperrors.New(apperrors.CodeInvalidInput, "key or slot id is required")
	}

	slot, err := s.slotRepo.FindByID(ctx, keyOrSlotID)
	if err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query slot")
	}
	if slot == nil {
		slot, err = s.slotRepo.FindByKeyID(ctx, keyOrSlotID)
		if err != nil {
			return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query slot by key")
		}
	}
	if slot == nil {
		return nil, apperrors.New(apperrors.CodeNotFound, "slot not found")
	}

	slot.Presence = normalizeSlotPresence(slot.Presence)
	return slot, nil
}

func isSupportedKeyStatus(status string) bool {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "AVAILABLE", "RESERVED", "BORROWED", "OVERDUE", "MAINTENANCE", "DISABLED":
		return true
	default:
		return false
	}
}

func normalizeSlotPresence(presence string) string {
	switch strings.ToUpper(strings.TrimSpace(presence)) {
	case "PRESENT", "ABSENT", "UNKNOWN", "FAULT":
		return strings.ToUpper(strings.TrimSpace(presence))
	case "EMPTY":
		// 旧版 slots.status 的 EMPTY 表示槽位没有检测到钥匙。
		return "ABSENT"
	default:
		return "UNKNOWN"
	}
}
