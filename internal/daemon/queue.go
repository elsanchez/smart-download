package daemon

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/elsanchez/smart-download/internal/domain"
	"github.com/elsanchez/smart-download/internal/downloader"
	"github.com/elsanchez/smart-download/internal/postprocessor"
	"github.com/elsanchez/smart-download/internal/repository"
)

// QueueManager gestiona la cola de descargas con workers paralelos
type QueueManager struct {
	downloadRepo  repository.DownloadRepository
	downloader    *downloader.Manager
	postprocessor postprocessor.PostProcessor
	workers       int
	workerPool    chan struct{}
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
	pollInterval  time.Duration
}

// NewQueueManager crea un nuevo gestor de cola
func NewQueueManager(
	downloadRepo repository.DownloadRepository,
	downloaderMgr *downloader.Manager,
	postproc postprocessor.PostProcessor,
	workers int,
) *QueueManager {
	ctx, cancel := context.WithCancel(context.Background())

	if workers <= 0 {
		workers = 3 // Default: 3 descargas paralelas
	}

	return &QueueManager{
		downloadRepo:  downloadRepo,
		downloader:    downloaderMgr,
		postprocessor: postproc,
		workers:       workers,
		workerPool:    make(chan struct{}, workers),
		ctx:           ctx,
		cancel:        cancel,
		pollInterval:  5 * time.Second,
	}
}

// Start inicia el queue manager
func (q *QueueManager) Start() {
	log.Printf("Queue manager started with %d workers", q.workers)
	go q.processLoop()
}

// Stop detiene el queue manager
func (q *QueueManager) Stop() {
	log.Println("Queue manager stopping...")
	q.cancel()
	q.wg.Wait()
	log.Println("Queue manager stopped")
}

// processLoop es el loop principal que busca descargas pendientes
func (q *QueueManager) processLoop() {
	ticker := time.NewTicker(q.pollInterval)
	defer ticker.Stop()

	// Procesar inmediatamente al inicio
	q.checkPendingDownloads()

	for {
		select {
		case <-q.ctx.Done():
			log.Println("Process loop shutting down")
			return

		case <-ticker.C:
			q.checkPendingDownloads()
		}
	}
}

// checkPendingDownloads verifica descargas pendientes y las procesa
func (q *QueueManager) checkPendingDownloads() {
	pending, err := q.downloadRepo.GetPending(q.ctx)
	if err != nil {
		log.Printf("Error getting pending downloads: %v", err)
		return
	}

	if len(pending) == 0 {
		return
	}

	log.Printf("Found %d pending download(s)", len(pending))

	for _, dl := range pending {
		select {
		case <-q.ctx.Done():
			return
		case q.workerPool <- struct{}{}: // Obtener slot de worker
			q.wg.Add(1)
			go q.processDownload(dl)
		default:
			// Pool lleno, procesar en siguiente tick
			log.Printf("Worker pool full, download %d queued for next tick", dl.ID)
		}
	}
}

// processDownload procesa una descarga individual
func (q *QueueManager) processDownload(dl *domain.Download) {
	defer q.wg.Done()
	defer func() { <-q.workerPool }() // Liberar slot

	log.Printf("Processing download %d: %s", dl.ID, dl.URL)

	// Actualizar status a downloading
	if err := q.downloadRepo.UpdateStatus(q.ctx, dl.ID, domain.StatusDownloading, ""); err != nil {
		log.Printf("Failed to update status for download %d: %v", dl.ID, err)
		return
	}

	// Ejecutar descarga
	outputPath, err := q.downloader.Download(q.ctx, dl)
	if err != nil {
		log.Printf("Download %d failed: %v", dl.ID, err)
		q.downloadRepo.UpdateStatus(q.ctx, dl.ID, domain.StatusFailed, err.Error())
		q.sendNotification("Download Failed", fmt.Sprintf("Failed to download: %s", dl.URL))
		return
	}

	log.Printf("Download %d downloaded to: %s", dl.ID, outputPath)

	// Post-procesamiento (si aplica)
	if q.postprocessor != nil && !dl.Options.AudioOnly {
		needsProcessing, err := q.postprocessor.NeedsProcessing(outputPath, &dl.Options)
		if err != nil {
			log.Printf("Failed to check processing needs for download %d: %v", dl.ID, err)
		}

		if needsProcessing || dl.Options.ClipStart != "" || dl.Options.ConvertToGIF {
			// Actualizar status a processing
			if err := q.downloadRepo.UpdateStatus(q.ctx, dl.ID, domain.StatusProcessing, ""); err != nil {
				log.Printf("Failed to update status for download %d: %v", dl.ID, err)
			}

			log.Printf("Post-processing download %d...", dl.ID)

			processedPath, err := q.postprocessor.Process(q.ctx, outputPath, &dl.Options)
			if err != nil {
				log.Printf("Post-processing %d failed: %v", dl.ID, err)
				q.downloadRepo.UpdateStatus(q.ctx, dl.ID, domain.StatusFailed, fmt.Sprintf("post-processing: %v", err))
				q.sendNotification("Processing Failed", fmt.Sprintf("Failed to process: %s", outputPath))
				return
			}

			outputPath = processedPath
			log.Printf("Download %d post-processed to: %s", dl.ID, outputPath)
		}
	}

	// Actualizar con path de salida final
	if err := q.downloadRepo.UpdateOutputPath(q.ctx, dl.ID, outputPath); err != nil {
		log.Printf("Failed to update output path for download %d: %v", dl.ID, err)
	}

	// Actualizar status a completed
	if err := q.downloadRepo.UpdateStatus(q.ctx, dl.ID, domain.StatusCompleted, ""); err != nil {
		log.Printf("Failed to update status for download %d: %v", dl.ID, err)
		return
	}

	log.Printf("Download %d completed: %s", dl.ID, outputPath)
	q.sendNotification("Download Complete", fmt.Sprintf("Ready: %s", outputPath))

	// Copiar path al clipboard
	q.copyToClipboard(outputPath)
}

// sendNotification envía una notificación al usuario
func (q *QueueManager) sendNotification(title, message string) {
	// Usar notify-send en Desktop Linux
	cmd := exec.Command("notify-send", title, message)
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to send notification: %v", err)
	}
}

// copyToClipboard copia texto al clipboard
func (q *QueueManager) copyToClipboard(text string) {
	// Intentar con xsel primero
	cmd := exec.Command("xsel", "-b", "-i")
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err == nil {
		log.Printf("Path copied to clipboard: %s", text)
		return
	}

	// Fallback a xclip
	cmd = exec.Command("xclip", "-selection", "clipboard")
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err == nil {
		log.Printf("Path copied to clipboard: %s", text)
		return
	}

	log.Printf("Failed to copy to clipboard (install xsel or xclip)")
}

// GetStats retorna estadísticas de la cola
func (q *QueueManager) GetStats(ctx context.Context) (map[string]int, error) {
	stats := make(map[string]int)

	pending, err := q.downloadRepo.CountByStatus(ctx, domain.StatusPending)
	if err != nil {
		return nil, err
	}
	stats["pending"] = pending

	active, err := q.downloadRepo.CountByStatus(ctx, domain.StatusDownloading)
	if err != nil {
		return nil, err
	}
	stats["downloading"] = active

	processing, err := q.downloadRepo.CountByStatus(ctx, domain.StatusProcessing)
	if err != nil {
		return nil, err
	}
	stats["processing"] = processing

	completed, err := q.downloadRepo.CountByStatus(ctx, domain.StatusCompleted)
	if err != nil {
		return nil, err
	}
	stats["completed"] = completed

	failed, err := q.downloadRepo.CountByStatus(ctx, domain.StatusFailed)
	if err != nil {
		return nil, err
	}
	stats["failed"] = failed

	stats["workers_total"] = q.workers
	stats["workers_busy"] = len(q.workerPool)

	return stats, nil
}
