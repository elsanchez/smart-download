package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/elsanchez/smart-download/internal/domain"
	"github.com/elsanchez/smart-download/internal/repository"
)

// AccountRepository implementa repository.AccountRepository usando SQLite
type AccountRepository struct {
	db *sqlx.DB
}

// Compiletime check: asegura que implementa la interfaz
var _ repository.AccountRepository = (*AccountRepository)(nil)

// NewAccountRepository crea un nuevo repositorio de cuentas
func NewAccountRepository(db *sqlx.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

// accountRow mapea la tabla SQL a struct Go
type accountRow struct {
	ID         int64         `db:"id"`
	Platform   string        `db:"platform"`
	Name       string        `db:"name"`
	CookiePath string        `db:"cookie_path"`
	IsActive   int           `db:"is_active"`
	LastUsed   sql.NullInt64 `db:"last_used"`
	CreatedAt  int64         `db:"created_at"`
}

// Create inserta una nueva cuenta
func (r *AccountRepository) Create(ctx context.Context, acc *domain.Account) (int64, error) {
	query := `
		INSERT INTO accounts (platform, name, cookie_path, is_active)
		VALUES (:platform, :name, :cookie_path, :is_active)
	`

	isActive := 0
	if acc.IsActive {
		isActive = 1
	}

	result, err := r.db.NamedExecContext(ctx, query, map[string]interface{}{
		"platform":    acc.Platform,
		"name":        acc.Name,
		"cookie_path": acc.CookiePath,
		"is_active":   isActive,
	})

	if err != nil {
		return 0, fmt.Errorf("insert account: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get last insert id: %w", err)
	}

	return id, nil
}

// GetByID obtiene una cuenta por ID
func (r *AccountRepository) GetByID(ctx context.Context, id int64) (*domain.Account, error) {
	var row accountRow

	query := `SELECT * FROM accounts WHERE id = ?`
	if err := r.db.GetContext(ctx, &row, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("account not found: %d", id)
		}
		return nil, fmt.Errorf("get account: %w", err)
	}

	return accountRowToDomain(&row), nil
}

// Update actualiza una cuenta
func (r *AccountRepository) Update(ctx context.Context, acc *domain.Account) error {
	isActive := 0
	if acc.IsActive {
		isActive = 1
	}

	var lastUsed interface{}
	if acc.LastUsed != nil {
		lastUsed = acc.LastUsed.Unix()
	}

	query := `
		UPDATE accounts
		SET platform = :platform, name = :name, cookie_path = :cookie_path,
		    is_active = :is_active, last_used = :last_used
		WHERE id = :id
	`

	_, err := r.db.NamedExecContext(ctx, query, map[string]interface{}{
		"id":          acc.ID,
		"platform":    acc.Platform,
		"name":        acc.Name,
		"cookie_path": acc.CookiePath,
		"is_active":   isActive,
		"last_used":   lastUsed,
	})

	return err
}

// Delete elimina una cuenta
func (r *AccountRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM accounts WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// GetActive obtiene la cuenta activa para una plataforma
func (r *AccountRepository) GetActive(ctx context.Context, platform string) (*domain.Account, error) {
	var row accountRow

	query := `
		SELECT * FROM accounts
		WHERE platform = ? AND is_active = 1
		LIMIT 1
	`

	if err := r.db.GetContext(ctx, &row, query, platform); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No hay cuenta activa (no es error)
		}
		return nil, fmt.Errorf("get active account: %w", err)
	}

	return accountRowToDomain(&row), nil
}

// GetAll obtiene todas las cuentas de una plataforma
func (r *AccountRepository) GetAll(ctx context.Context, platform string) ([]*domain.Account, error) {
	var rows []accountRow

	query := `
		SELECT * FROM accounts
		WHERE platform = ?
		ORDER BY is_active DESC, last_used DESC
	`

	if err := r.db.SelectContext(ctx, &rows, query, platform); err != nil {
		return nil, fmt.Errorf("get all accounts: %w", err)
	}

	return accountRowsToDomain(rows), nil
}

// ListPlatforms lista todas las plataformas con cuentas
func (r *AccountRepository) ListPlatforms(ctx context.Context) ([]string, error) {
	var platforms []string

	query := `SELECT DISTINCT platform FROM accounts ORDER BY platform`
	if err := r.db.SelectContext(ctx, &platforms, query); err != nil {
		return nil, fmt.Errorf("list platforms: %w", err)
	}

	return platforms, nil
}

// SetActive establece una cuenta como activa (desactiva las demás de la plataforma)
func (r *AccountRepository) SetActive(ctx context.Context, platform, name string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Desactivar todas las cuentas de la plataforma
	if _, err := tx.ExecContext(ctx, `
		UPDATE accounts SET is_active = 0 WHERE platform = ?
	`, platform); err != nil {
		return fmt.Errorf("deactivate accounts: %w", err)
	}

	// Activar la cuenta específica
	result, err := tx.ExecContext(ctx, `
		UPDATE accounts
		SET is_active = 1, last_used = ?
		WHERE platform = ? AND name = ?
	`, time.Now().Unix(), platform, name)

	if err != nil {
		return fmt.Errorf("activate account: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("account not found: %s/%s", platform, name)
	}

	return tx.Commit()
}

// UpdateLastUsed actualiza el timestamp de último uso
func (r *AccountRepository) UpdateLastUsed(ctx context.Context, id int64) error {
	query := `UPDATE accounts SET last_used = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, time.Now().Unix(), id)
	return err
}

// Helper: conversión row → domain
func accountRowToDomain(row *accountRow) *domain.Account {
	acc := &domain.Account{
		ID:         row.ID,
		Platform:   row.Platform,
		Name:       row.Name,
		CookiePath: row.CookiePath,
		IsActive:   row.IsActive == 1,
		CreatedAt:  time.Unix(row.CreatedAt, 0),
	}

	if row.LastUsed.Valid {
		t := time.Unix(row.LastUsed.Int64, 0)
		acc.LastUsed = &t
	}

	return acc
}

// Helper: conversión múltiples rows → domain
func accountRowsToDomain(rows []accountRow) []*domain.Account {
	accounts := make([]*domain.Account, 0, len(rows))

	for _, row := range rows {
		accounts = append(accounts, accountRowToDomain(&row))
	}

	return accounts
}
