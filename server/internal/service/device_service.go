package service

import (
	"context"
	"strings"
	"time"

	apperrors "github.com/zhouwu97/key-cabinet/server/internal/platform/errors"
	"github.com/zhouwu97/key-cabinet/server/internal/repository"
)

type DeviceView struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Location        string     `json:"location,omitempty"`
	Status          string     `json:"status"`
	TotalSlots      int        `json:"totalSlots"`
	AvailableSlots  int        `json:"availableSlots"`
	IPAddress       string     `json:"ipAddress,omitempty"`
	LastHeartbeatAt *time.Time `json:"lastHeartbeatAt"`
	CreatedAt       time.Time  `json:"createdAt,omitempty"`
	UpdatedAt       time.Time  `json:"updatedAt,omitempty"`
}

type DeviceService interface {
	ListDevices(ctx context.Context) ([]*DeviceView, error)
	GetDevice(ctx context.Context, id string) (*DeviceView, error)
	GetDeviceStatus(ctx context.Context, id string) (*DeviceView, error)
	GetDeviceSlots(ctx context.Context, deviceID string) ([]*repository.Slot, error)
}

type deviceService struct {
	deviceRepo repository.DeviceRepository
	slotRepo   repository.SlotRepository
}

func NewDeviceService(deviceRepo repository.DeviceRepository, slotRepo repository.SlotRepository) DeviceService {
	return &deviceService{deviceRepo: deviceRepo, slotRepo: slotRepo}
}

func (s *deviceService) ListDevices(ctx context.Context) ([]*DeviceView, error) {
	devices, err := s.deviceRepo.FindAll(ctx)
	if err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query devices")
	}

	views := make([]*DeviceView, 0, len(devices))
	for _, device := range devices {
		view, err := s.buildView(ctx, device)
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return views, nil
}

func (s *deviceService) GetDevice(ctx context.Context, id string) (*DeviceView, error) {
	return s.getDeviceView(ctx, id)
}

func (s *deviceService) GetDeviceStatus(ctx context.Context, id string) (*DeviceView, error) {
	return s.getDeviceView(ctx, id)
}

func (s *deviceService) GetDeviceSlots(ctx context.Context, deviceID string) ([]*repository.Slot, error) {
	if strings.TrimSpace(deviceID) == "" {
		return nil, apperrors.New(apperrors.CodeInvalidInput, "device id is required")
	}
	device, err := s.deviceRepo.FindByID(ctx, deviceID)
	if err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query device")
	}
	if device == nil {
		return nil, apperrors.New(apperrors.CodeNotFound, "device not found")
	}

	slots, err := s.slotRepo.FindByDeviceID(ctx, deviceID)
	if err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query device slots")
	}
	for _, slot := range slots {
		if slot == nil {
			continue
		}
		slot.Presence = normalizeSlotPresence(slot.Presence)
	}
	return slots, nil
}

func (s *deviceService) getDeviceView(ctx context.Context, id string) (*DeviceView, error) {
	if strings.TrimSpace(id) == "" {
		return nil, apperrors.New(apperrors.CodeInvalidInput, "device id is required")
	}
	device, err := s.deviceRepo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query device")
	}
	if device == nil {
		return nil, apperrors.New(apperrors.CodeNotFound, "device not found")
	}
	return s.buildView(ctx, device)
}

func (s *deviceService) buildView(ctx context.Context, device *repository.Device) (*DeviceView, error) {
	slots, err := s.slotRepo.FindByDeviceID(ctx, device.ID)
	if err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query device slots")
	}

	totalSlots := device.Capacity
	if totalSlots <= 0 {
		totalSlots = len(slots)
	}
	availableSlots := 0
	for _, slot := range slots {
		if slot == nil {
			continue
		}
		if slot.Enabled && normalizeSlotPresence(slot.Presence) == "PRESENT" {
			availableSlots++
		}
	}

	return &DeviceView{
		ID:              device.ID,
		Name:            device.Name,
		Location:        device.Location,
		Status:          device.Status,
		TotalSlots:      totalSlots,
		AvailableSlots:  availableSlots,
		IPAddress:       device.IPAddress,
		LastHeartbeatAt: device.LastHeartbeatAt,
		CreatedAt:       device.CreatedAt,
		UpdatedAt:       device.UpdatedAt,
	}, nil
}
