package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/elsanchez/smart-download/internal/daemon"
	"github.com/elsanchez/smart-download/internal/downloader"
	"github.com/elsanchez/smart-download/internal/repository/sqlite"
	"github.com/elsanchez/smart-download/pkg/client"
)

const (
	version = "0.1.0"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("smart-downloadd v%s starting...", version)

	// Verificar dependencias
	if err := downloader.CheckDependencies(); err != nil {
		log.Fatalf("Dependency check failed: %v", err)
	}
	log.Println("✓ Dependencies check passed (yt-dlp, gallery-dl)")

	// Obtener directorios
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get home directory: %v", err)
	}

	dataDir := filepath.Join(homeDir, ".local", "share", "smart-download")
	outputDir := filepath.Join(homeDir, "Downloads", "download_video")
	cookiesDir := filepath.Join(homeDir, "Documents", "cookies")

	// Crear directorios
	for _, dir := range []string{dataDir, outputDir, cookiesDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	log.Printf("Data directory: %s", dataDir)
	log.Printf("Output directory: %s", outputDir)
	log.Printf("Cookies directory: %s", cookiesDir)

	// Inicializar base de datos
	db, err := sqlite.NewDatabase(dataDir)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	log.Println("✓ Database initialized")

	// Crear downloader manager
	downloaderMgr := downloader.NewManager(outputDir, cookiesDir, db.AccountRepo)
	log.Println("✓ Downloader manager initialized")

	// Crear queue manager
	workers := 3 // Configurable
	queueMgr := daemon.NewQueueManager(db.DownloadRepo, downloaderMgr, workers)
	queueMgr.Start()
	defer queueMgr.Stop()
	log.Printf("✓ Queue manager started (%d workers)", workers)

	// Crear handlers
	handlers := daemon.NewHandlers(db.DownloadRepo, db.AccountRepo, queueMgr)

	// Crear servidor
	socketPath := client.GetDefaultSocketPath()
	server := daemon.NewServer(socketPath, queueMgr, handlers)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := server.Start(ctx); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	log.Println("✓ Server started")
	log.Printf("Socket: %s", socketPath)
	log.Println("smart-downloadd is ready")

	// Esperar señal de terminación
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	log.Printf("Received signal: %v", sig)
	log.Println("Shutting down gracefully...")

	cancel()
}
