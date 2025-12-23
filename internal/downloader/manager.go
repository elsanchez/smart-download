package downloader

import (
	"context"
	"fmt"

	"github.com/elsanchez/smart-download/internal/domain"
)

// Manager gestiona múltiples downloaders y selecciona el apropiado
type Manager struct {
	ytdlp     *YtDlp
	gallerydl *GalleryDl
}

// NewManager crea un nuevo manager de downloaders
func NewManager(outputDir string, cookiesDir string, accountRepo AccountGetter) *Manager {
	return &Manager{
		ytdlp:     NewYtDlp(outputDir, cookiesDir, accountRepo),
		gallerydl: NewGalleryDl(outputDir, cookiesDir, accountRepo),
	}
}

// Download selecciona el downloader apropiado y ejecuta la descarga
func (m *Manager) Download(ctx context.Context, dl *domain.Download) (string, error) {
	// Detectar plataforma si no está especificada
	if dl.Platform == "" {
		dl.Platform = DetectPlatform(dl.URL)
	}

	// Extraer username si no está especificado
	if dl.Username == "" {
		dl.Username = ExtractUsername(dl.URL)
	}

	// Seleccionar downloader
	var downloader Downloader
	if m.gallerydl.Supports(dl.URL) {
		downloader = m.gallerydl
	} else if m.ytdlp.Supports(dl.URL) {
		downloader = m.ytdlp
	} else {
		return "", fmt.Errorf("no downloader supports URL: %s", dl.URL)
	}

	// Ejecutar descarga
	return downloader.Download(ctx, dl)
}

// CheckDependencies verifica que los downloaders estén instalados
func CheckDependencies() error {
	if err := CheckYtDlpInstalled(); err != nil {
		return fmt.Errorf("yt-dlp check: %w", err)
	}

	if err := CheckGalleryDlInstalled(); err != nil {
		return fmt.Errorf("gallery-dl check: %w", err)
	}

	return nil
}
