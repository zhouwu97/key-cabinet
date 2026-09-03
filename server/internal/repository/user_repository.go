package repository

import (
	"context"
)

type User struct {
	ID        string
	Name      string
	StudentID string
	Phone     string
	Role      string
	Status    string
}

type UserIdentity struct {
	ID             string
	UserID         string
	IdentityType   string
	ProviderUserID string
}

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	FindByID(ctx context.Context, id string) (*User, error)
	FindByStudentID(ctx context.Context, studentID string) (*User, error)
	Update(ctx context.Context, user *User) error
	FindIdentity(ctx context.Context, identityType, providerUserID string) (*UserIdentity, error)
	CreateIdentity(ctx context.Context, identity *UserIdentity) error
}
