package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/zhouwu97/key-cabinet/server/internal/infrastructure/device"
	apperrors "github.com/zhouwu97/key-cabinet/server/internal/platform/errors"
	"github.com/zhouwu97/key-cabinet/server/internal/platform/jwt"
	"github.com/zhouwu97/key-cabinet/server/internal/repository"
)

type MatchedKeyInfo struct {
	Key          *repository.Key
	Slot         *repository.Slot
	IsBorrowable bool
}

type CabinetDirectDispenseParams struct {
	RequestID string
	DeviceID  string
	RoomNo    string
	KeyID     string
	UserID    string
	StudentNo string
	Purpose   string
}

type FaceAuthParams struct {
	DeviceID       string
	StudentNo      string
	Confidence     float64
	LivenessPassed bool
}

type FaceAuthResult struct {
	User               *repository.User
	ActiveReservations []*repository.Reservation
	ActiveBorrows      []*repository.BorrowRecord
	FaceSessionToken   string
	CabinetToken       string
}

type CabinetService interface {
	MatchRoom(ctx context.Context, deviceID, roomNo string) ([]*MatchedKeyInfo, error)
	DirectDispense(ctx context.Context, params CabinetDirectDispenseParams) (*repository.DeviceOperation, *repository.BorrowRecord, *repository.Slot, *repository.Key, error)
	FaceAuth(ctx context.Context, params FaceAuthParams) (*FaceAuthResult, error)
}

type cabinetService struct {
	userRepo        repository.UserRepository
	keyRepo         repository.KeyRepository
	slotRepo        repository.SlotRepository
	deviceRepo      repository.DeviceRepository
	reservationRepo repository.ReservationRepository
	borrowRepo      repository.BorrowRepository
	operationRepo   repository.OperationRepository
	deviceGateway   device.DeviceGateway
	tokenService    *jwt.TokenService
}

func NewCabinetService(
	userRepo repository.UserRepository,
	keyRepo repository.KeyRepository,
	slotRepo repository.SlotRepository,
	deviceRepo repository.DeviceRepository,
	reservationRepo repository.ReservationRepository,
	borrowRepo repository.BorrowRepository,
	operationRepo repository.OperationRepository,
	deviceGateway device.DeviceGateway,
	tokenService *jwt.TokenService,
) CabinetService {
	return &cabinetService{
		userRepo:        userRepo,
		keyRepo:         keyRepo,
		slotRepo:        slotRepo,
		deviceRepo:      deviceRepo,
		reservationRepo: reservationRepo,
		borrowRepo:      borrowRepo,
		operationRepo:   operationRepo,
		deviceGateway:   deviceGateway,
		tokenService:    tokenService,
	}
}

func (s *cabinetService) MatchRoom(ctx context.Context, deviceID, roomNo string) ([]*MatchedKeyInfo, error) {
	roomNo = strings.TrimSpace(roomNo)
	if roomNo == "" {
		return nil, apperrors.New(apperrors.CodeInvalidInput, "roomNo is required")
	}

	filter := repository.KeyListFilter{
		DeviceID: strings.TrimSpace(deviceID),
	}
	keys, err := s.keyRepo.List(ctx, filter)
	if err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query keys")
	}

	var results []*MatchedKeyInfo
	for _, key := range keys {
		if strings.EqualFold(strings.TrimSpace(key.RoomNo), roomNo) {
			var slot *repository.Slot
			if key.SlotID != "" {
				slot, _ = s.slotRepo.FindByID(ctx, key.SlotID)
			}
			isBorrowable := key.Status == "AVAILABLE" && key.Enabled
			if slot != nil && slot.Presence == "ABSENT" {
				isBorrowable = false
			}
			results = append(results, &MatchedKeyInfo{
				Key:          key,
				Slot:         slot,
				IsBorrowable: isBorrowable,
			})
		}
	}
	return results, nil
}

func (s *cabinetService) DirectDispense(ctx context.Context, params CabinetDirectDispenseParams) (*repository.DeviceOperation, *repository.BorrowRecord, *repository.Slot, *repository.Key, error) {
	params.RequestID = strings.TrimSpace(params.RequestID)
	params.DeviceID = strings.TrimSpace(params.DeviceID)
	params.RoomNo = strings.TrimSpace(params.RoomNo)
	params.KeyID = strings.TrimSpace(params.KeyID)
	params.UserID = strings.TrimSpace(params.UserID)

	if params.RequestID == "" || params.DeviceID == "" {
		return nil, nil, nil, nil, apperrors.New(apperrors.CodeInvalidInput, "requestId and deviceId are required")
	}
	if params.UserID == "" {
		return nil, nil, nil, nil, apperrors.New(apperrors.CodeUnauthorized, "direct dispense requires an authenticated user from a valid face session")
	}
	if params.RoomNo == "" && params.KeyID == "" {
		return nil, nil, nil, nil, apperrors.New(apperrors.CodeInvalidInput, "either roomNo or keyId is required")
	}

	// 1. 验证设备存在且在线
	dev, err := s.deviceRepo.FindByID(ctx, params.DeviceID)
	if err != nil {
		return nil, nil, nil, nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query device")
	}
	if dev == nil {
		return nil, nil, nil, nil, apperrors.New(apperrors.CodeNotFound, "未找到指定的机柜设备")
	}
	if dev.Status != "ONLINE" {
		return nil, nil, nil, nil, apperrors.New(apperrors.CodeForbidden, "机柜设备当前离线或处于维护状态")
	}

	// 2. 查验用户身份 (严格根据已核验的 UserID)
	user, err := s.userRepo.FindByID(ctx, params.UserID)
	if err != nil {
		return nil, nil, nil, nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query user by id")
	}
	if user == nil {
		return nil, nil, nil, nil, apperrors.New(apperrors.CodeNotFound, "未找到该用户")
	}
	if user.Status != "ACTIVE" {
		return nil, nil, nil, nil, apperrors.New(apperrors.CodeForbidden, "用户账号状态不可借用钥匙")
	}
	if !user.IdentityVerified {
		return nil, nil, nil, nil, apperrors.New(apperrors.CodeForbidden, "用户未通过实名核验，不可出钥")
	}

	// 3. 幂等拦截
	existingOp, err := s.operationRepo.FindByRequestID(ctx, params.RequestID)
	if err != nil {
		return nil, nil, nil, nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to check idempotent request")
	}
	if existingOp != nil {
		return existingOp, nil, nil, nil, nil
	}

	// 4. 匹配钥匙
	var targetKey *repository.Key
	if params.KeyID != "" {
		k, err := s.keyRepo.FindByID(ctx, params.KeyID)
		if err != nil {
			return nil, nil, nil, nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query key")
		}
		if k == nil {
			return nil, nil, nil, nil, apperrors.New(apperrors.CodeNotFound, "钥匙不存在")
		}
		if k.DeviceID != params.DeviceID {
			return nil, nil, nil, nil, apperrors.New(apperrors.CodeForbidden, "钥匙不在指定机柜中")
		}
		targetKey = k
	} else {
		matched, err := s.MatchRoom(ctx, params.DeviceID, params.RoomNo)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		for _, m := range matched {
			if m.IsBorrowable {
				targetKey = m.Key
				break
			}
		}
		if targetKey == nil {
			return nil, nil, nil, nil, apperrors.New(apperrors.CodeNotFound, fmt.Sprintf("房间 %s 当前无在柜可用的钥匙", params.RoomNo))
		}
	}

	if !targetKey.Enabled {
		return nil, nil, nil, nil, apperrors.New(apperrors.CodeForbidden, "钥匙已被停用")
	}
	if targetKey.Status != "AVAILABLE" {
		return nil, nil, nil, nil, apperrors.New(apperrors.CodeConflict, "钥匙当前不可借出 (状态为: "+targetKey.Status+")")
	}

	// 5. 获取槽位并核验在位与可用
	slot, err := s.slotRepo.FindByID(ctx, targetKey.SlotID)
	if err != nil {
		return nil, nil, nil, nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query slot")
	}
	if slot == nil {
		return nil, nil, nil, nil, apperrors.New(apperrors.CodeNotFound, "钥匙未绑定有效槽位")
	}
	if !slot.Enabled {
		return nil, nil, nil, nil, apperrors.New(apperrors.CodeForbidden, "钥匙所在槽位已被禁用")
	}
	if slot.Presence != "PRESENT" {
		return nil, nil, nil, nil, apperrors.New(apperrors.CodeConflict, "钥匙当前不在槽位中 (状态为: "+slot.Presence+")")
	}

	// 6. 校验用户没有正在借用该钥匙
	activeBorrows, err := s.borrowRepo.FindByUserID(ctx, user.ID)
	if err == nil {
		for _, b := range activeBorrows {
			if (b.Status == "BORROWED" || b.Status == "BORROWING") && b.KeyID == targetKey.ID {
				return nil, nil, nil, nil, apperrors.New(apperrors.CodeConflict, "您当前已有该钥匙的借用流程")
			}
		}
	}

	// 7. 校验机柜当前没有正在执行中的冲突操作
	activeDevOp, err := s.operationRepo.FindActiveByDeviceID(ctx, params.DeviceID)
	if err != nil {
		return nil, nil, nil, nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to check active device operation")
	}
	if activeDevOp != nil {
		return nil, nil, nil, nil, apperrors.New(apperrors.CodeConflict, "机柜正在执行其它借还操作，请稍候")
	}

	// 8. 计算借用时长策略 (现场直借默认4小时，实验室钥匙默认8小时)
	borrowDuration := 4 * time.Hour
	if strings.Contains(strings.ToLower(targetKey.Category), "lab") {
		borrowDuration = 8 * time.Hour
	}

	// 9. 生成借用记录与操作
	// 审计物理一致性：在设备物理出钥前，BorrowedAt 必须为 nil，待设备确认 KEY_REMOVED 后由 CompletePickup 写入
	now := time.Now().UTC()
	purpose := params.Purpose
	if purpose == "" {
		purpose = "现场房间号自助出钥"
	}
	borrow := &repository.BorrowRecord{
		ID:               "bor_" + generateRandomID(8),
		UserID:           user.ID,
		KeyID:            targetKey.ID,
		DeviceID:         params.DeviceID,
		SlotID:           slot.ID,
		Status:           "BORROWING",
		BorrowedAt:       nil,
		ExpectedReturnAt: now.Add(borrowDuration),
		Purpose:          purpose,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	operation := &repository.DeviceOperation{
		ID:             "op_" + generateRandomID(8),
		RequestID:      params.RequestID,
		UserID:         user.ID,
		BorrowRecordID: borrow.ID,
		DeviceID:       params.DeviceID,
		SlotID:         slot.ID,
		KeyID:          targetKey.ID,
		Action:         "PICKUP",
		Status:         "AUTHORIZED",
		CreatedAt:      now,
		StartedAt:      &now,
	}

	if err := s.operationRepo.PreparePickup(ctx, borrow, operation); err != nil {
		return nil, nil, nil, nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to prepare direct dispense")
	}

	// 10. 发送出钥指令
	if err := s.operationRepo.BeginExecution(ctx, operation.ID, now); err != nil {
		return nil, nil, nil, nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to begin direct dispense execution")
	}
	operation.Status = "EXECUTING"
	operation.SentAt = &now

	cmd := device.DeviceCommand{
		OperationID:    operation.ID,
		DeviceID:       operation.DeviceID,
		SlotID:         operation.SlotID,
		SlotNo:         slot.SlotNo,
		KeyID:          targetKey.ID,
		Type:           operation.Action,
		TimeoutSeconds: 120,
	}
	if err := s.deviceGateway.StartPickup(ctx, cmd); err != nil {
		_ = s.operationRepo.Fail(ctx, operation.ID, "DEVICE_COMMAND_FAILED", err.Error(), time.Now().UTC())
		return nil, nil, nil, nil, apperrors.WrapWithCode(err, apperrors.CodeServiceUnavailable, "failed to dispatch pickup command to device")
	}

	return operation, borrow, slot, targetKey, nil
}

func (s *cabinetService) FaceAuth(ctx context.Context, params FaceAuthParams) (*FaceAuthResult, error) {
	params.DeviceID = strings.TrimSpace(params.DeviceID)
	if params.DeviceID == "" {
		return nil, apperrors.New(apperrors.CodeInvalidInput, "deviceId is required")
	}

	// 验证机柜设备是否存在且在线
	dev, err := s.deviceRepo.FindByID(ctx, params.DeviceID)
	if err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query device")
	}
	if dev == nil {
		return nil, apperrors.New(apperrors.CodeNotFound, "未找到指定的机柜设备")
	}
	if dev.Status != "ONLINE" {
		return nil, apperrors.New(apperrors.CodeForbidden, "机柜设备当前处于离线或异常状态，无法进行人脸认证")
	}

	// 活体检测强校验
	if !params.LivenessPassed {
		return nil, apperrors.New("ERR_LIVENESS_FAILED", "活体检测未通过，拒绝开柜")
	}

	// 置信度阈值校验：必须 >= 0.80，防止 0 或过低置信度绕过
	if params.Confidence < 0.80 {
		return nil, apperrors.New("ERR_FACE_CONFIDENCE_LOW", "人脸比对置信度过低 (需 >= 0.80)，请重新正对摄像头")
	}

	params.StudentNo = strings.TrimSpace(params.StudentNo)
	if params.StudentNo == "" {
		return nil, apperrors.New(apperrors.CodeInvalidInput, "studentNo is required")
	}

	user, err := s.userRepo.FindByStudentNo(ctx, params.StudentNo)
	if err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query user")
	}
	if user == nil {
		user, err = s.userRepo.FindByID(ctx, params.StudentNo)
		if err != nil {
			return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query user by id")
		}
	}
	if user == nil {
		return nil, apperrors.New(apperrors.CodeNotFound, "未匹配到实名认证用户，请先在小程序完成身份登记")
	}
	if user.Status != "ACTIVE" {
		return nil, apperrors.New(apperrors.CodeForbidden, "用户账号异常，无法开柜")
	}
	if !user.IdentityVerified {
		return nil, apperrors.New(apperrors.CodeForbidden, "用户未通过实名身份核验，无法现场人脸取钥")
	}

	// 查该用户待取预约 (本柜机相关)
	var activeReservations []*repository.Reservation
	allReservations, err := s.reservationRepo.FindByUserID(ctx, user.ID)
	if err == nil {
		for _, rsv := range allReservations {
			if rsv.Status == "APPROVED" || rsv.Status == "ACTIVE" {
				activeReservations = append(activeReservations, rsv)
			}
		}
	}

	// 查该用户借用中记录
	var activeBorrows []*repository.BorrowRecord
	allBorrows, err := s.borrowRepo.FindByUserID(ctx, user.ID)
	if err == nil {
		for _, b := range allBorrows {
			if b.Status == "BORROWED" || b.Status == "RETURNING" {
				activeBorrows = append(activeBorrows, b)
			}
		}
	}

	var sessionToken string
	if s.tokenService != nil {
		sessionToken, _ = s.tokenService.GenerateFaceSession(user.ID, user.Role, params.DeviceID, 5*time.Minute)
	}

	return &FaceAuthResult{
		User:               user,
		ActiveReservations: activeReservations,
		ActiveBorrows:      activeBorrows,
		FaceSessionToken:   sessionToken,
		CabinetToken:       sessionToken,
	}, nil
}

func generateRandomID(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}
