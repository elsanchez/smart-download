package downloader

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/elsanchez/smart-download/internal/domain"
)

// GalleryDl implementa Downloader usando gallery-dl
type GalleryDl struct {
	outputDir   string
	cookiesDir  string
	accountRepo AccountGetter
}

// NewGalleryDl crea un nuevo downloader de gallery-dl
func NewGalleryDl(outputDir string, cookiesDir string, accountRepo AccountGetter) *GalleryDl {
	return &GalleryDl{
		outputDir:   outputDir,
		cookiesDir:  cookiesDir,
		accountRepo: accountRepo,
	}
}

// Download ejecuta la descarga usando gallery-dl
func (g *GalleryDl) Download(ctx context.Context, dl *domain.Download) (string, error) {
	// Crear subdirectorio por plataforma
	platformDir := filepath.Join(g.outputDir, dl.Platform)
	if err := os.MkdirAll(platformDir, 0755); err != nil {
		return "", fmt.Errorf("create platform dir: %w", err)
	}

	// Generar filename base
	filenameBase := g.generateFilename(dl)

	// Construir argumentos
	args := []string{
		"-D", platformDir, // Destination directory
		"-o", filenameBase + ".{extension}", // Output template
	}

	// Cookies: siempre buscar cuenta activa para la plataforma
	if g.accountRepo != nil {
		account, err := g.accountRepo.GetActive(ctx, dl.Platform)
		if err == nil && account != nil && account.CookiePath != "" {
			args = append(args, "--cookies", account.CookiePath)
		}
	}

	// Opciones adicionales
	args = append(args,
		"--no-check-certificate",
	)

	// Modos especiales
	if dl.Options.AudioOnly {
		// gallery-dl no soporta audio-only directamente
		// Descargar normal y luego procesar con ffmpeg
	}

	// URL al final
	args = append(args, dl.URL)

	// Ejecutar gallery-dl
	cmd := exec.CommandContext(ctx, "gallery-dl", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return "", fmt.Errorf("gallery-dl failed: %w\nOutput: %s", err, output)
	}

	// Buscar el archivo descargado
	outputPath, err := g.findDownloadedFile(platformDir, filenameBase)
	if err != nil {
		// Para galleries (múltiples archivos), retornar el directorio
		if strings.Contains(string(output), "downloaded") {
			return platformDir, nil
		}
		return "", fmt.Errorf("find downloaded file: %w\ngallery-dl output: %s", err, output)
	}

	return outputPath, nil
}

// Supports verifica si gallery-dl soporta la URL
func (g *GalleryDl) Supports(url string) bool {
	return NeedsGalleryDL(url)
}

// generateFilename genera el nombre de archivo base
func (g *GalleryDl) generateFilename(dl *domain.Download) string {
	timestamp := time.Now().Format("02012006")

	username := dl.Username
	if username == "" {
		username = "user"
	}

	return fmt.Sprintf("%s_%s_%s", dl.Platform, username, timestamp)
}

// findDownloadedFile busca el archivo descargado en el directorio
func (g *GalleryDl) findDownloadedFile(dir, basePattern string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("read dir: %w", err)
	}

	// Buscar el archivo más reciente que coincida con el patrón
	var newestFile string
	var newestTime time.Time

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Verificar si el nombre contiene el patrón base
		if strings.Contains(entry.Name(), basePattern) ||
			strings.HasPrefix(entry.Name(), strings.Split(basePattern, "_")[0]) {

			info, err := entry.Info()
			if err != nil {
				continue
			}

			if info.ModTime().After(newestTime) {
				newestTime = info.ModTime()
				newestFile = filepath.Join(dir, entry.Name())
			}
		}
	}

	if newestFile == "" {
		return "", fmt.Errorf("no file found matching pattern: %s", basePattern)
	}

	return newestFile, nil
}

// CheckInstalled verifica si gallery-dl está instalado
func CheckGalleryDlInstalled() error {
	cmd := exec.Command("gallery-dl", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gallery-dl not found: %w (install: pip install gallery-dl)", err)
	}
	return nil
}
