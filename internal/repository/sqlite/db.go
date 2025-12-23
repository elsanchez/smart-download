package sqlite

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Database encapsula la conexión a SQLite
type Database struct {
	DB               *sqlx.DB
	DownloadRepo     *DownloadRepository
	AccountRepo      *AccountRepository
	sqlDB            *sql.DB // Para migrations
}

// NewDatabase crea una nueva base de datos y ejecuta migrations
func NewDatabase(dataDir string) (*Database, error) {
	// Crear directorio de datos si no existe
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "downloads.db")

	// Abrir con database/sql (para migrations)
	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Ejecutar migrations
	if err := runMigrations(sqlDB); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	// Abrir con sqlx (para queries)
	db := sqlx.NewDb(sqlDB, "sqlite3")

	// Configuraciones SQLite
	db.SetMaxOpenConns(1) // SQLite no soporta concurrencia de escritura

	// Inicializar repositorios
	database := &Database{
		DB:           db,
		sqlDB:        sqlDB,
		DownloadRepo: NewDownloadRepository(db),
		AccountRepo:  NewAccountRepository(db),
	}

	return database, nil
}

// runMigrations ejecuta las migraciones usando golang-migrate
func runMigrations(db *sql.DB) error {
	// Driver para SQLite
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("create migration driver: %w", err)
	}

	// Source desde filesystem embebido
	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("create migration source: %w", err)
	}

	// Crear migrator
	m, err := migrate.NewWithInstance("iofs", source, "sqlite3", driver)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	// Ejecutar migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}

// Close cierra la conexión a la base de datos
func (d *Database) Close() error {
	return d.DB.Close()
}
