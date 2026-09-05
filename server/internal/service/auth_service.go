package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zhouwu97/key-cabinet/server/internal/infrastructure/wechat"
	"github.com/zhouwu97/key-cabinet/server/internal/platform/errors"
	"github.com/zhouwu97/key-cabinet/server/internal/platform/jwt"
	"github.com/zhouwu97/key-cabinet/server/internal/repository"
)

type LoginResult struct {
	AccessToken string           `json:"accessToken"`
	ExpiresIn   int              `json:"expiresIn"`
	User        *repository.User `json:"user"`
}

type UpdateProfileRequest struct {
	Name       string `json:"name"`
	StudentNo  string `json:"studentNo"`
	Department string `json:"department"`
	Phone      string `json:"phone"`
}

type AuthService interface {
	WechatLogin(ctx context.Context, jsCode string) (*LoginResult, error)
	GetMe(ctx context.Context, userID string) (*repository.User, error)
	UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) (*repository.User, error)
}

type authService struct {
	userRepo     repository.UserRepository
	wechatClient wechat.Client
	tokenService *jwt.TokenService
	jwtExpireSec int
}

func NewAuthService(
	userRepo repository.UserRepository,
	wechatClient wechat.Client,
	tokenService *jwt.TokenService,
	jwtExpireSec int,
) AuthService {
	return &authService{
		userRepo:     userRepo,
		wechatClient: wechatClient,
		tokenService: tokenService,
		jwtExpireSec: jwtExpireSec,
	}
}

func generateUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant RFC4122
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func (s *authService) WechatLogin(ctx context.Context, jsCode string) (*LoginResult, error) {
	if jsCode == "" {
		return nil, errors.New(errors.CodeInvalidInput, "code is required")
	}

	session, err := s.wechatClient.Code2Session(ctx, jsCode)
	if err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeUnauthorized, "wechat login failed")
	}

	// Find or create UserIdentity
	identity, err := s.userRepo.FindIdentity(ctx, "WECHAT", session.OpenID)
	if err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeInternalError, "failed to query user identity")
	}

	var user *repository.User
	if identity != nil {
		// Existing user
		user, err = s.userRepo.FindByID(ctx, identity.UserID)
		if err != nil {
			return nil, errors.WrapWithCode(err, errors.CodeInternalError, "failed to query user profile")
		}
		if user == nil {
			return nil, errors.New(errors.CodeNotFound, "associated user profile not found")
		}
	} else {
		// New user registration
		newUserID := "usr_" + generateUUID()[:12]
		now := time.Now()
		user = &repository.User{
			ID:               newUserID,
			Name:             "微信用户",
			StudentNo:        "",
			Phone:            "",
			Department:       "",
			Role:             "USER",
			Status:           "ACTIVE",
			CreditScore:      100,
			ProfileCompleted: false,
			CreatedAt:        now,
			UpdatedAt:        now,
		}

		metadata := make(map[string]string)
		if session.UnionID != "" {
			metadata["unionid"] = session.UnionID
		}
		metaBytes, _ := json.Marshal(metadata)

		newIdentity := &repository.UserIdentity{
			ID:        "ident_" + generateUUID()[:12],
			UserID:    newUserID,
			Provider:  "WECHAT",
			Subject:   session.OpenID,
			Metadata:  string(metaBytes),
			CreatedAt: now,
		}

		if registrationRepo, ok := s.userRepo.(repository.UserRegistrationRepository); ok {
			if err := registrationRepo.CreateWithIdentity(ctx, user, newIdentity); err != nil {
				return nil, errors.WrapWithCode(err, errors.CodeInternalError, "failed to create user registration")
			}
		} else {
			// 兼容仅用于测试的旧仓储；生产 PostgreSQL 仓储走上面的事务路径。
			if err := s.userRepo.Create(ctx, user); err != nil {
				return nil, errors.WrapWithCode(err, errors.CodeInternalError, "failed to create user")
			}
			if err := s.userRepo.CreateIdentity(ctx, newIdentity); err != nil {
				return nil, errors.WrapWithCode(err, errors.CodeInternalError, "failed to create user identity")
			}
		}
	}

	token, err := s.tokenService.Generate(user.ID, user.Role)
	if err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeInternalError, "failed to generate access token")
	}

	return &LoginResult{
		AccessToken: token,
		ExpiresIn:   s.jwtExpireSec,
		User:        user,
	}, nil
}

func (s *authService) GetMe(ctx context.Context, userID string) (*repository.User, error) {
	if userID == "" {
		return nil, errors.New(errors.CodeUnauthorized, "user id not found in context")
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeInternalError, "failed to query user")
	}
	if user == nil {
		return nil, errors.New(errors.CodeNotFound, "user not found")
	}

	return user, nil
}

func (s *authService) UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) (*repository.User, error) {
	if userID == "" {
		return nil, errors.New(errors.CodeUnauthorized, "user id not found in context")
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeInternalError, "failed to query user")
	}
	if user == nil {
		return nil, errors.New(errors.CodeNotFound, "user not found")
	}

	if req.Name != "" {
		user.Name = req.Name
	}
	if req.StudentNo != "" {
		user.StudentNo = req.StudentNo
	}
	if req.Department != "" {
		user.Department = req.Department
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}

	// Mark profile completed if name and studentNo are both filled
	if user.Name != "" && user.Name != "微信用户" && user.StudentNo != "" {
		user.ProfileCompleted = true
	}

	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeInternalError, "failed to update user profile")
	}

	return user, nil
}
