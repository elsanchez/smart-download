package repository

import (
	"context"

	"github.com/elsanchez/smart-download/internal/domain"
)

// DownloadRepository define las operaciones sobre descargas
type DownloadRepository interface {
	// CRUD básico
	Create(ctx context.Context, dl *domain.Download) (int64, error)
	GetByID(ctx context.Context, id int64) (*domain.Download, error)
	Update(ctx context.Context, dl *domain.Download) error
	Delete(ctx context.Context, id int64) error

	// Queries especializadas
	GetPending(ctx context.Context) ([]*domain.Download, error)
	GetActive(ctx context.Context) ([]*domain.Download, error)
	GetRecent(ctx context.Context, limit int) ([]*domain.Download, error)
	GetByStatus(ctx context.Context, status domain.DownloadStatus) ([]*domain.Download, error)

	// Updates parciales
	UpdateStatus(ctx context.Context, id int64, status domain.DownloadStatus, errMsg string) error
	UpdateOutputPath(ctx context.Context, id int64, path string) error

	// Estadísticas
	CountByStatus(ctx context.Context, status domain.DownloadStatus) (int, error)
	CountTotal(ctx context.Context) (int, error)
}
