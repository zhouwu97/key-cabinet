package repository

import (
	"context"
	"time"
)

type User struct {
	ID               string    `gorm:"primaryKey;column:id" json:"id"`
	Name             string    `gorm:"column:name" json:"name"`
	StudentNo        string    `gorm:"column:student_no" json:"studentNo"`
	Phone            string    `gorm:"column:phone" json:"phone"`
	Department       string    `gorm:"column:department" json:"department"`
	Role             string    `gorm:"column:role;default:USER" json:"role"`
	Status           string    `gorm:"column:status;default:ACTIVE" json:"status"`
	CreditScore      int       `gorm:"column:credit_score;default:100" json:"creditScore"`
	ProfileCompleted bool      `gorm:"column:profile_completed;default:false" json:"profileCompleted"`
	CreatedAt        time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt        time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (User) TableName() string {
	return "users"
}

type UserIdentity struct {
	ID        string    `gorm:"primaryKey;column:id" json:"id"`
	UserID    string    `gorm:"column:user_id;index" json:"userId"`
	Provider  string    `gorm:"column:provider" json:"provider"` // e.g. "WECHAT"
	Subject   string    `gorm:"column:subject" json:"subject"`   // e.g. openid
	Metadata  string    `gorm:"column:metadata;type:jsonb" json:"metadata"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
}

func (UserIdentity) TableName() string {
	return "user_identities"
}

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	FindByID(ctx context.Context, id string) (*User, error)
	FindByStudentNo(ctx context.Context, studentNo string) (*User, error)
	Update(ctx context.Context, user *User) error
	FindIdentity(ctx context.Context, provider, subject string) (*UserIdentity, error)
	CreateIdentity(ctx context.Context, identity *UserIdentity) error
}

// UserRegistrationRepository 为首次登录提供用户与身份的原子创建能力。
// 保留独立方法是为了兼容已有的查询/更新仓储实现；生产仓储应实现该接口。
type UserRegistrationRepository interface {
	UserRepository
	CreateWithIdentity(ctx context.Context, user *User, identity *UserIdentity) error
}
