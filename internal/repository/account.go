package repository

import (
	"context"

	"github.com/elsanchez/smart-download/internal/domain"
)

// AccountRepository define las operaciones sobre cuentas
type AccountRepository interface {
	// CRUD básico
	Create(ctx context.Context, acc *domain.Account) (int64, error)
	GetByID(ctx context.Context, id int64) (*domain.Account, error)
	Update(ctx context.Context, acc *domain.Account) error
	Delete(ctx context.Context, id int64) error

	// Queries especializadas
	GetActive(ctx context.Context, platform string) (*domain.Account, error)
	GetAll(ctx context.Context, platform string) ([]*domain.Account, error)
	ListPlatforms(ctx context.Context) ([]string, error)

	// Gestión de cuenta activa
	SetActive(ctx context.Context, platform, name string) error
	UpdateLastUsed(ctx context.Context, id int64) error
}
