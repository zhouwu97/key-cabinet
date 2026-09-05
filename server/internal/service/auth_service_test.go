package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zhouwu97/key-cabinet/server/internal/infrastructure/wechat"
	"github.com/zhouwu97/key-cabinet/server/internal/platform/jwt"
	"github.com/zhouwu97/key-cabinet/server/internal/repository"
	"github.com/zhouwu97/key-cabinet/server/internal/service"
)

// MockUserRepository implements in-memory repository for unit testing
type MockUserRepository struct {
	users                map[string]*repository.User
	identities           map[string]*repository.UserIdentity
	failIdentityCreation bool
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users:      make(map[string]*repository.User),
		identities: make(map[string]*repository.UserIdentity),
	}
}

func (m *MockUserRepository) Create(ctx context.Context, user *repository.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *MockUserRepository) FindByID(ctx context.Context, id string) (*repository.User, error) {
	return m.users[id], nil
}

func (m *MockUserRepository) FindByStudentNo(ctx context.Context, studentNo string) (*repository.User, error) {
	for _, u := range m.users {
		if u.StudentNo == studentNo {
			return u, nil
		}
	}
	return nil, nil
}

func (m *MockUserRepository) Update(ctx context.Context, user *repository.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *MockUserRepository) FindIdentity(ctx context.Context, provider, subject string) (*repository.UserIdentity, error) {
	key := provider + ":" + subject
	return m.identities[key], nil
}

func (m *MockUserRepository) CreateIdentity(ctx context.Context, identity *repository.UserIdentity) error {
	key := identity.Provider + ":" + identity.Subject
	m.identities[key] = identity
	return nil
}

func (m *MockUserRepository) CreateWithIdentity(
	ctx context.Context,
	user *repository.User,
	identity *repository.UserIdentity,
) error {
	if m.failIdentityCreation {
		return assert.AnError
	}
	if err := m.Create(ctx, user); err != nil {
		return err
	}
	return m.CreateIdentity(ctx, identity)
}

func TestAuthService_WechatLogin_NewAndExistingUser(t *testing.T) {
	repo := NewMockUserRepository()
	wechatClient := wechat.NewClient("", "", true) // 显式开启开发 Mock
	tokenService := jwt.NewTokenService("test-secret-with-sufficient-length!!", 3600)
	authSvc := service.NewAuthService(repo, wechatClient, tokenService, 3600)

	ctx := context.Background()

	// 1. First time login with code -> creates user
	res1, err := authSvc.WechatLogin(ctx, "mock_code_123")
	require.NoError(t, err)
	require.NotNil(t, res1)
	assert.NotEmpty(t, res1.AccessToken)
	assert.NotEmpty(t, res1.User.ID)
	assert.Equal(t, "USER", res1.User.Role)
	assert.Equal(t, false, res1.User.ProfileCompleted)
	for _, identity := range repo.identities {
		assert.NotContains(t, identity.Metadata, "session_key")
	}

	// Validate token
	claims, err := tokenService.Validate(res1.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, res1.User.ID, claims.UserID)
	assert.Equal(t, "USER", claims.Role)

	// 2. Second time login with same code -> returns existing user
	res2, err := authSvc.WechatLogin(ctx, "mock_code_123")
	require.NoError(t, err)
	assert.Equal(t, res1.User.ID, res2.User.ID)

	// 3. GetMe
	me, err := authSvc.GetMe(ctx, res1.User.ID)
	require.NoError(t, err)
	assert.Equal(t, res1.User.ID, me.ID)

	// 4. Update Profile
	updated, err := authSvc.UpdateProfile(ctx, res1.User.ID, service.UpdateProfileRequest{
		Name:       "张三",
		StudentNo:  "2023001",
		Department: "计算机与软件工程学院",
		Phone:      "13800000000",
	})
	require.NoError(t, err)
	assert.Equal(t, "张三", updated.Name)
	assert.Equal(t, "2023001", updated.StudentNo)
	assert.Equal(t, true, updated.ProfileCompleted)
}

func TestAuthService_WechatLogin_AtomicRegistrationFailure(t *testing.T) {
	repo := NewMockUserRepository()
	repo.failIdentityCreation = true
	wechatClient := wechat.NewClient("", "", true)
	tokenService := jwt.NewTokenService("test-secret-with-sufficient-length!!", 3600)
	authSvc := service.NewAuthService(repo, wechatClient, tokenService, 3600)

	_, err := authSvc.WechatLogin(context.Background(), "mock_atomic_failure")
	require.Error(t, err)
	assert.Empty(t, repo.users)
	assert.Empty(t, repo.identities)
}
