package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/elsanchez/smart-download/internal/domain"
)

func TestDatabase_CreateAndGetDownload(t *testing.T) {
	// Crear DB temporal
	tmpDir := t.TempDir()
	db, err := NewDatabase(tmpDir)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Crear una descarga
	dl := &domain.Download{
		URL:      "https://youtube.com/watch?v=test",
		Platform: "youtube",
		Username: "testuser",
		Status:   domain.StatusPending,
		Options: domain.DownloadOptions{
			Resolution: "1080p",
			AudioOnly:  false,
		},
	}

	id, err := db.DownloadRepo.Create(ctx, dl)
	if err != nil {
		t.Fatalf("failed to create download: %v", err)
	}

	if id == 0 {
		t.Fatal("expected non-zero ID")
	}

	// Obtener la descarga
	retrieved, err := db.DownloadRepo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("failed to get download: %v", err)
	}

	// Verificar datos
	if retrieved.URL != dl.URL {
		t.Errorf("expected URL %s, got %s", dl.URL, retrieved.URL)
	}

	if retrieved.Platform != dl.Platform {
		t.Errorf("expected platform %s, got %s", dl.Platform, retrieved.Platform)
	}

	if retrieved.Status != domain.StatusPending {
		t.Errorf("expected status pending, got %s", retrieved.Status)
	}

	if retrieved.Options.Resolution != "1080p" {
		t.Errorf("expected resolution 1080p, got %s", retrieved.Options.Resolution)
	}

	t.Logf("✅ Download created with ID: %d", id)
}

func TestDatabase_MigrationsApplied(t *testing.T) {
	// Crear DB temporal
	tmpDir := t.TempDir()
	db, err := NewDatabase(tmpDir)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer db.Close()

	// Verificar que existe el archivo de base de datos
	dbPath := filepath.Join(tmpDir, "downloads.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatal("database file was not created")
	}

	// Verificar que las tablas existen
	ctx := context.Background()

	var count int
	err = db.DB.GetContext(ctx, &count, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='downloads'")
	if err != nil {
		t.Fatalf("failed to query tables: %v", err)
	}

	if count != 1 {
		t.Error("downloads table was not created")
	}

	err = db.DB.GetContext(ctx, &count, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='accounts'")
	if err != nil {
		t.Fatalf("failed to query tables: %v", err)
	}

	if count != 1 {
		t.Error("accounts table was not created")
	}

	t.Log("✅ Migrations applied successfully")
}

func TestDatabase_AccountActiveSwitch(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := NewDatabase(tmpDir)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Crear dos cuentas de Twitter
	acc1 := &domain.Account{
		Platform:   domain.PlatformTwitter,
		Name:       "personal",
		CookiePath: "/path/to/cookies1.txt",
		IsActive:   true,
	}

	id1, err := db.AccountRepo.Create(ctx, acc1)
	if err != nil {
		t.Fatalf("failed to create account 1: %v", err)
	}

	acc2 := &domain.Account{
		Platform:   domain.PlatformTwitter,
		Name:       "work",
		CookiePath: "/path/to/cookies2.txt",
		IsActive:   false,
	}

	id2, err := db.AccountRepo.Create(ctx, acc2)
	if err != nil {
		t.Fatalf("failed to create account 2: %v", err)
	}

	// Verificar que personal está activa
	active, err := db.AccountRepo.GetActive(ctx, domain.PlatformTwitter)
	if err != nil {
		t.Fatalf("failed to get active account: %v", err)
	}

	if active.ID != id1 {
		t.Errorf("expected account %d to be active, got %d", id1, active.ID)
	}

	// Cambiar a work
	err = db.AccountRepo.SetActive(ctx, domain.PlatformTwitter, "work")
	if err != nil {
		t.Fatalf("failed to set active account: %v", err)
	}

	// Verificar que work está activa
	active, err = db.AccountRepo.GetActive(ctx, domain.PlatformTwitter)
	if err != nil {
		t.Fatalf("failed to get active account: %v", err)
	}

	if active.ID != id2 {
		t.Errorf("expected account %d to be active, got %d", id2, active.ID)
	}

	// Verificar que personal fue desactivada
	acc1Updated, err := db.AccountRepo.GetByID(ctx, id1)
	if err != nil {
		t.Fatalf("failed to get account 1: %v", err)
	}

	if acc1Updated.IsActive {
		t.Error("account 1 should be inactive after switch")
	}

	t.Log("✅ Account switching works correctly")
}
