package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/elsanchez/smart-download/internal/domain"
	"github.com/elsanchez/smart-download/internal/repository"
)

// DownloadRepository implementa repository.DownloadRepository usando SQLite
type DownloadRepository struct {
	db *sqlx.DB
}

// Compiletime check: asegura que implementa la interfaz
var _ repository.DownloadRepository = (*DownloadRepository)(nil)

// NewDownloadRepository crea un nuevo repositorio de descargas
func NewDownloadRepository(db *sqlx.DB) *DownloadRepository {
	return &DownloadRepository{db: db}
}

// downloadRow mapea la tabla SQL a struct Go
type downloadRow struct {
	ID           int64          `db:"id"`
	URL          string         `db:"url"`
	Platform     sql.NullString `db:"platform"`
	Username     sql.NullString `db:"username"`
	Status       string         `db:"status"`
	OutputPath   sql.NullString `db:"output_path"`
	OptionsJSON  string         `db:"options"`
	AccountID    sql.NullInt64  `db:"account_id"`
	CreatedAt    int64          `db:"created_at"`
	CompletedAt  sql.NullInt64  `db:"completed_at"`
	ErrorMessage sql.NullString `db:"error_message"`
}

// Create inserta una nueva descarga
func (r *DownloadRepository) Create(ctx context.Context, dl *domain.Download) (int64, error) {
	optJSON, err := json.Marshal(dl.Options)
	if err != nil {
		return 0, fmt.Errorf("marshal options: %w", err)
	}

	query := `
		INSERT INTO downloads (url, platform, username, status, options, account_id)
		VALUES (:url, :platform, :username, :status, :options, :account_id)
	`

	result, err := r.db.NamedExecContext(ctx, query, map[string]interface{}{
		"url":        dl.URL,
		"platform":   dl.Platform,
		"username":   dl.Username,
		"status":     string(dl.Status),
		"options":    string(optJSON),
		"account_id": dl.AccountID,
	})

	if err != nil {
		return 0, fmt.Errorf("insert download: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get last insert id: %w", err)
	}

	return id, nil
}

// GetByID obtiene una descarga por ID
func (r *DownloadRepository) GetByID(ctx context.Context, id int64) (*domain.Download, error) {
	var row downloadRow

	query := `SELECT * FROM downloads WHERE id = ?`
	if err := r.db.GetContext(ctx, &row, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("download not found: %d", id)
		}
		return nil, fmt.Errorf("get download: %w", err)
	}

	return rowToDomain(&row)
}

// Update actualiza una descarga completa
func (r *DownloadRepository) Update(ctx context.Context, dl *domain.Download) error {
	optJSON, err := json.Marshal(dl.Options)
	if err != nil {
		return fmt.Errorf("marshal options: %w", err)
	}

	var completedAt interface{}
	if dl.CompletedAt != nil {
		completedAt = dl.CompletedAt.Unix()
	}

	query := `
		UPDATE downloads
		SET url = :url, platform = :platform, username = :username,
		    status = :status, output_path = :output_path, options = :options,
		    account_id = :account_id, completed_at = :completed_at,
		    error_message = :error_message
		WHERE id = :id
	`

	_, err = r.db.NamedExecContext(ctx, query, map[string]interface{}{
		"id":            dl.ID,
		"url":           dl.URL,
		"platform":      dl.Platform,
		"username":      dl.Username,
		"status":        string(dl.Status),
		"output_path":   dl.OutputPath,
		"options":       string(optJSON),
		"account_id":    dl.AccountID,
		"completed_at":  completedAt,
		"error_message": dl.ErrorMessage,
	})

	return err
}

// Delete elimina una descarga
func (r *DownloadRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM downloads WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// GetPending obtiene todas las descargas pendientes
func (r *DownloadRepository) GetPending(ctx context.Context) ([]*domain.Download, error) {
	return r.GetByStatus(ctx, domain.StatusPending)
}

// GetActive obtiene descargas en proceso
func (r *DownloadRepository) GetActive(ctx context.Context) ([]*domain.Download, error) {
	var rows []downloadRow

	query := `
		SELECT * FROM downloads
		WHERE status IN ('downloading', 'processing')
		ORDER BY created_at ASC
	`

	if err := r.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("get active downloads: %w", err)
	}

	return rowsToDomain(rows)
}

// GetRecent obtiene las descargas recientes
func (r *DownloadRepository) GetRecent(ctx context.Context, limit int) ([]*domain.Download, error) {
	var rows []downloadRow

	query := `
		SELECT * FROM downloads
		ORDER BY created_at DESC
		LIMIT ?
	`

	if err := r.db.SelectContext(ctx, &rows, query, limit); err != nil {
		return nil, fmt.Errorf("get recent downloads: %w", err)
	}

	return rowsToDomain(rows)
}

// GetByStatus obtiene descargas por status
func (r *DownloadRepository) GetByStatus(ctx context.Context, status domain.DownloadStatus) ([]*domain.Download, error) {
	var rows []downloadRow

	query := `SELECT * FROM downloads WHERE status = ? ORDER BY created_at ASC`
	if err := r.db.SelectContext(ctx, &rows, query, string(status)); err != nil {
		return nil, fmt.Errorf("get downloads by status: %w", err)
	}

	return rowsToDomain(rows)
}

// UpdateStatus actualiza solo el status y mensaje de error
func (r *DownloadRepository) UpdateStatus(ctx context.Context, id int64, status domain.DownloadStatus, errMsg string) error {
	var completedAt interface{}
	if status == domain.StatusCompleted || status == domain.StatusFailed {
		completedAt = time.Now().Unix()
	}

	query := `
		UPDATE downloads
		SET status = ?, error_message = ?, completed_at = ?
		WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query, string(status), errMsg, completedAt, id)
	return err
}

// UpdateOutputPath actualiza solo el path de salida
func (r *DownloadRepository) UpdateOutputPath(ctx context.Context, id int64, path string) error {
	query := `UPDATE downloads SET output_path = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, path, id)
	return err
}

// CountByStatus cuenta descargas por status
func (r *DownloadRepository) CountByStatus(ctx context.Context, status domain.DownloadStatus) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM downloads WHERE status = ?`
	err := r.db.GetContext(ctx, &count, query, string(status))
	return count, err
}

// CountTotal cuenta todas las descargas
func (r *DownloadRepository) CountTotal(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM downloads`
	err := r.db.GetContext(ctx, &count, query)
	return count, err
}

// Helper: conversión row → domain
func rowToDomain(row *downloadRow) (*domain.Download, error) {
	var opts domain.DownloadOptions
	if err := json.Unmarshal([]byte(row.OptionsJSON), &opts); err != nil {
		return nil, fmt.Errorf("unmarshal options: %w", err)
	}

	dl := &domain.Download{
		ID:           row.ID,
		URL:          row.URL,
		Platform:     row.Platform.String,
		Username:     row.Username.String,
		Status:       domain.DownloadStatus(row.Status),
		OutputPath:   row.OutputPath.String,
		Options:      opts,
		ErrorMessage: row.ErrorMessage.String,
		CreatedAt:    time.Unix(row.CreatedAt, 0),
	}

	if row.AccountID.Valid {
		dl.AccountID = &row.AccountID.Int64
	}

	if row.CompletedAt.Valid {
		t := time.Unix(row.CompletedAt.Int64, 0)
		dl.CompletedAt = &t
	}

	return dl, nil
}

// Helper: conversión múltiples rows → domain
func rowsToDomain(rows []downloadRow) ([]*domain.Download, error) {
	downloads := make([]*domain.Download, 0, len(rows))

	for _, row := range rows {
		dl, err := rowToDomain(&row)
		if err != nil {
			return nil, err
		}
		downloads = append(downloads, dl)
	}

	return downloads, nil
}
