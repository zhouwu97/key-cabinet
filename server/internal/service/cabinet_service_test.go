package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/zhouwu97/key-cabinet/server/internal/infrastructure/device"
	"github.com/zhouwu97/key-cabinet/server/internal/platform/jwt"
	"github.com/zhouwu97/key-cabinet/server/internal/repository"
)

type fakeCabinetUserRepository struct {
	user *repository.User
}

func (r *fakeCabinetUserRepository) Create(_ context.Context, _ *repository.User) error { return nil }
func (r *fakeCabinetUserRepository) FindByID(_ context.Context, id string) (*repository.User, error) {
	if r.user != nil && r.user.ID == id {
		return r.user, nil
	}
	return nil, nil
}
func (r *fakeCabinetUserRepository) FindByStudentNo(_ context.Context, no string) (*repository.User, error) {
	if r.user != nil && r.user.StudentNo == no {
		return r.user, nil
	}
	return nil, nil
}
func (r *fakeCabinetUserRepository) Update(_ context.Context, _ *repository.User) error { return nil }
func (r *fakeCabinetUserRepository) FindIdentity(_ context.Context, _, _ string) (*repository.UserIdentity, error) {
	return nil, nil
}
func (r *fakeCabinetUserRepository) FindIdentityByUserID(_ context.Context, _, _ string) (*repository.UserIdentity, error) {
	return nil, nil
}
func (r *fakeCabinetUserRepository) CreateIdentity(_ context.Context, _ *repository.UserIdentity) error {
	return nil
}

type fakeCabinetDeviceRepository struct{}

func (r *fakeCabinetDeviceRepository) FindByID(_ context.Context, id string) (*repository.Device, error) {
	return &repository.Device{ID: id, Status: "ONLINE"}, nil
}
func (r *fakeCabinetDeviceRepository) FindAll(_ context.Context) ([]*repository.Device, error) {
	return nil, nil
}

type fakeCabinetReservationRepository struct{}

func (r *fakeCabinetReservationRepository) Create(_ context.Context, _ *repository.Reservation) error {
	return nil
}
func (r *fakeCabinetReservationRepository) FindByID(_ context.Context, _ string) (*repository.Reservation, error) {
	return nil, nil
}
func (r *fakeCabinetReservationRepository) FindByUserID(_ context.Context, _ string) ([]*repository.Reservation, error) {
	return nil, nil
}
func (r *fakeCabinetReservationRepository) List(_ context.Context, _ repository.ReservationListFilter) ([]*repository.Reservation, error) {
	return nil, nil
}
func (r *fakeCabinetReservationRepository) FindConflicts(_ context.Context, _ string, _, _ time.Time) ([]*repository.Reservation, error) {
	return nil, nil
}
func (r *fakeCabinetReservationRepository) Update(_ context.Context, _ *repository.Reservation) error {
	return nil
}

type fakeCabinetBorrowRepository struct{}

func (r *fakeCabinetBorrowRepository) Create(_ context.Context, _ *repository.BorrowRecord) error {
	return nil
}
func (r *fakeCabinetBorrowRepository) FindByID(_ context.Context, _ string) (*repository.BorrowRecord, error) {
	return nil, nil
}
func (r *fakeCabinetBorrowRepository) FindByReservationID(_ context.Context, _ string) (*repository.BorrowRecord, error) {
	return nil, nil
}
func (r *fakeCabinetBorrowRepository) FindByUserID(_ context.Context, _ string) ([]*repository.BorrowRecord, error) {
	return nil, nil
}
func (r *fakeCabinetBorrowRepository) List(_ context.Context, _ repository.BorrowListFilter) ([]*repository.BorrowRecord, error) {
	return nil, nil
}
func (r *fakeCabinetBorrowRepository) FindActiveByKeyID(_ context.Context, _ string) (*repository.BorrowRecord, error) {
	return nil, nil
}
func (r *fakeCabinetBorrowRepository) Update(_ context.Context, _ *repository.BorrowRecord) error {
	return nil
}
func (r *fakeCabinetBorrowRepository) MarkOverdue(_ context.Context, _ time.Time) error {
	return nil
}

type fakeCabinetOperationRepository struct {
	operations map[string]*repository.DeviceOperation
}

func (r *fakeCabinetOperationRepository) Create(_ context.Context, op *repository.DeviceOperation) error {
	r.operations[op.ID] = op
	return nil
}
func (r *fakeCabinetOperationRepository) PreparePickup(_ context.Context, _ *repository.BorrowRecord, op *repository.DeviceOperation) error {
	r.operations[op.ID] = op
	return nil
}
func (r *fakeCabinetOperationRepository) PrepareReturn(_ context.Context, _, _ string, op *repository.DeviceOperation) error {
	r.operations[op.ID] = op
	return nil
}
func (r *fakeCabinetOperationRepository) BeginExecution(_ context.Context, id string, _ time.Time) error {
	if op, ok := r.operations[id]; ok {
		op.Status = "EXECUTING"
	}
	return nil
}
func (r *fakeCabinetOperationRepository) FindByID(_ context.Context, id string) (*repository.DeviceOperation, error) {
	return r.operations[id], nil
}
func (r *fakeCabinetOperationRepository) FindByRequestID(_ context.Context, reqID string) (*repository.DeviceOperation, error) {
	for _, op := range r.operations {
		if op.RequestID == reqID {
			return op, nil
		}
	}
	return nil, nil
}
func (r *fakeCabinetOperationRepository) FindByReservationID(_ context.Context, _ string) ([]*repository.DeviceOperation, error) {
	return nil, nil
}
func (r *fakeCabinetOperationRepository) FindActiveByReservation(_ context.Context, _ string) (*repository.DeviceOperation, error) {
	return nil, nil
}
func (r *fakeCabinetOperationRepository) FindActiveByDeviceID(_ context.Context, _ string) (*repository.DeviceOperation, error) {
	return nil, nil
}
func (r *fakeCabinetOperationRepository) FindActiveByKeyID(_ context.Context, _ string) (*repository.DeviceOperation, error) {
	return nil, nil
}
func (r *fakeCabinetOperationRepository) FindActiveByUserID(_ context.Context, _ string) (*repository.DeviceOperation, error) {
	return nil, nil
}
func (r *fakeCabinetOperationRepository) FindExpired(_ context.Context, _ time.Time) ([]*repository.DeviceOperation, error) {
	return nil, nil
}
func (r *fakeCabinetOperationRepository) Update(_ context.Context, op *repository.DeviceOperation) error {
	r.operations[op.ID] = op
	return nil
}
func (r *fakeCabinetOperationRepository) CreateEvent(_ context.Context, _ *repository.OperationEvent) error {
	return nil
}
func (r *fakeCabinetOperationRepository) AppendEventIfActive(_ context.Context, _ *repository.OperationEvent) error {
	return nil
}
func (r *fakeCabinetOperationRepository) CompletePickup(_ context.Context, id string, _ time.Time) error {
	if op, ok := r.operations[id]; ok {
		op.Status = "SUCCESS"
	}
	return nil
}
func (r *fakeCabinetOperationRepository) CompleteReturn(_ context.Context, id string, _ time.Time) error {
	if op, ok := r.operations[id]; ok {
		op.Status = "SUCCESS"
	}
	return nil
}
func (r *fakeCabinetOperationRepository) Fail(_ context.Context, id, code, msg string, _ time.Time) error {
	if op, ok := r.operations[id]; ok {
		op.Status = "FAILED"
		op.ErrorCode = code
		op.ErrorMessage = msg
	}
	return nil
}
func (r *fakeCabinetOperationRepository) Cancel(_ context.Context, id string, _ time.Time) error {
	if op, ok := r.operations[id]; ok {
		op.Status = "CANCELLED"
	}
	return nil
}
func (r *fakeCabinetOperationRepository) Timeout(_ context.Context, id string, _ time.Time) error {
	if op, ok := r.operations[id]; ok {
		op.Status = "TIMEOUT"
	}
	return nil
}

func TestCabinetService_MatchRoom(t *testing.T) {
	keyRepo := &fakeKeyRepository{keys: []*repository.Key{
		{ID: "k1", Name: "101室主钥匙", RoomNo: "101", DeviceID: "CAB001", SlotID: "s1", Status: "AVAILABLE", Enabled: true},
		{ID: "k2", Name: "102室钥匙", RoomNo: "102", DeviceID: "CAB001", SlotID: "s2", Status: "BORROWED", Enabled: true},
	}}
	slotRepo := &fakeSlotRepository{slot: &repository.Slot{ID: "s1", DeviceID: "CAB001", SlotNo: 1, Presence: "PRESENT", Enabled: true}}

	svc := NewCabinetService(
		&fakeCabinetUserRepository{},
		keyRepo,
		slotRepo,
		&fakeCabinetDeviceRepository{},
		&fakeCabinetReservationRepository{},
		&fakeCabinetBorrowRepository{},
		&fakeCabinetOperationRepository{operations: make(map[string]*repository.DeviceOperation)},
		device.NewMockDeviceGateway(),
		nil,
	)

	results, err := svc.MatchRoom(context.Background(), "CAB001", "101")
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "k1", results[0].Key.ID)
	require.True(t, results[0].IsBorrowable)

	results102, err := svc.MatchRoom(context.Background(), "CAB001", "102")
	require.NoError(t, err)
	require.Len(t, results102, 1)
	require.False(t, results102[0].IsBorrowable)
}

func TestCabinetService_DirectDispense(t *testing.T) {
	keyRepo := &fakeKeyRepository{keys: []*repository.Key{
		{ID: "k1", Name: "101室主钥匙", RoomNo: "101", DeviceID: "CAB001", SlotID: "s1", Status: "AVAILABLE", Enabled: true},
	}}
	slotRepo := &fakeSlotRepository{slot: &repository.Slot{ID: "s1", DeviceID: "CAB001", SlotNo: 1, Presence: "PRESENT", Enabled: true}}
	userRepo := &fakeCabinetUserRepository{user: &repository.User{ID: "u1", StudentNo: "20230001", Name: "张三", Status: "ACTIVE", Role: "USER", IdentityVerified: true}}
	opRepo := &fakeCabinetOperationRepository{operations: make(map[string]*repository.DeviceOperation)}

	svc := NewCabinetService(
		userRepo,
		keyRepo,
		slotRepo,
		&fakeCabinetDeviceRepository{},
		&fakeCabinetReservationRepository{},
		&fakeCabinetBorrowRepository{},
		opRepo,
		device.NewMockDeviceGateway(),
		nil,
	)

	// 1. 缺少有效 FaceSession UserID 拦截
	_, _, _, _, err := svc.DirectDispense(context.Background(), CabinetDirectDispenseParams{
		RequestID: "req_dispense_0",
		DeviceID:  "CAB001",
		RoomNo:    "101",
		StudentNo: "20230001",
	})
	require.Error(t, err)

	// 2. 正常经过 FaceSession 授权出钥
	op, borrow, slot, key, err := svc.DirectDispense(context.Background(), CabinetDirectDispenseParams{
		RequestID: "req_dispense_1",
		DeviceID:  "CAB001",
		RoomNo:    "101",
		UserID:    "u1",
		StudentNo: "20230001",
	})
	require.NoError(t, err)
	require.NotNil(t, op)
	require.NotNil(t, borrow)
	require.Equal(t, "k1", key.ID)
	require.Equal(t, 1, slot.SlotNo)
	require.Equal(t, "EXECUTING", op.Status)
	// 关键审计检查：物理出钥前 BorrowedAt 必须为 nil
	require.Nil(t, borrow.BorrowedAt)

	// 3. 需审批钥匙 (RequiresApproval=true) 无审批预约拦截
	keyRepo.keys[0].RequiresApproval = true
	_, _, _, _, err = svc.DirectDispense(context.Background(), CabinetDirectDispenseParams{
		RequestID: "req_dispense_2",
		DeviceID:  "CAB001",
		RoomNo:    "101",
		UserID:    "u1",
		StudentNo: "20230001",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "需要管理员审批")
}

func TestCabinetService_FaceAuth(t *testing.T) {
	userRepo := &fakeCabinetUserRepository{user: &repository.User{ID: "u1", StudentNo: "20230001", Name: "李四", Status: "ACTIVE", Role: "USER", IdentityVerified: true}}
	tokenSvc := jwt.NewTokenService("test_secret_key_long_enough_32_bytes", 86400)

	svc := NewCabinetService(
		userRepo,
		&fakeKeyRepository{},
		&fakeSlotRepository{},
		&fakeCabinetDeviceRepository{},
		&fakeCabinetReservationRepository{},
		&fakeCabinetBorrowRepository{},
		&fakeCabinetOperationRepository{operations: make(map[string]*repository.DeviceOperation)},
		device.NewMockDeviceGateway(),
		tokenSvc,
	)

	// 1. 活体失败拦截
	_, err := svc.FaceAuth(context.Background(), FaceAuthParams{
		DeviceID:       "CAB001",
		StudentNo:      "20230001",
		Confidence:     0.95,
		LivenessPassed: false,
	})
	require.Error(t, err)

	// 2. 置信度过低拦截 (< 0.80)
	_, err = svc.FaceAuth(context.Background(), FaceAuthParams{
		DeviceID:       "CAB001",
		StudentNo:      "20230001",
		Confidence:     0.75,
		LivenessPassed: true,
	})
	require.Error(t, err)

	// 3. 正常刷脸通过，生成短期绑定柜机的 FaceSessionToken
	res, err := svc.FaceAuth(context.Background(), FaceAuthParams{
		DeviceID:       "CAB001",
		StudentNo:      "20230001",
		Confidence:     0.95,
		LivenessPassed: true,
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, "李四", res.User.Name)
	require.NotEmpty(t, res.FaceSessionToken)

	claims, err := tokenSvc.Validate(res.FaceSessionToken)
	require.NoError(t, err)
	require.Equal(t, "FACE_SESSION", claims.TokenType)
	require.Equal(t, "CAB001", claims.DeviceID)
	require.Equal(t, "u1", claims.UserID)
}
