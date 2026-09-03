package repository

import "context"

type Key struct {
	ID        string
	Name      string
	KeyNumber string
	RFIDTag   string
	DeviceID  string
	SlotID    string
	Category  string
	Status    string
}

type KeyRepository interface {
	FindByID(ctx context.Context, id string) (*Key, error)
	FindAll(ctx context.Context) ([]*Key, error)
	FindByStatus(ctx context.Context, status string) ([]*Key, error)
	Update(ctx context.Context, key *Key) error
}
