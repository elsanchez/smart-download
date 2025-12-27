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

// YtDlp implementa Downloader usando yt-dlp
type YtDlp struct {
	outputDir   string
	cookiesDir  string
	accountRepo AccountGetter // Interfaz para obtener cuentas
}

// AccountGetter define la interfaz para obtener cuentas (evita dependencia circular)
type AccountGetter interface {
	GetActive(ctx context.Context, platform string) (*domain.Account, error)
}

// NewYtDlp crea un nuevo downloader de yt-dlp
func NewYtDlp(outputDir string, cookiesDir string, accountRepo AccountGetter) *YtDlp {
	return &YtDlp{
		outputDir:   outputDir,
		cookiesDir:  cookiesDir,
		accountRepo: accountRepo,
	}
}

// Download ejecuta la descarga usando yt-dlp
func (y *YtDlp) Download(ctx context.Context, dl *domain.Download) (string, error) {
	// Crear subdirectorio por plataforma
	platformDir := filepath.Join(y.outputDir, dl.Platform)
	if err := os.MkdirAll(platformDir, 0755); err != nil {
		return "", fmt.Errorf("create platform dir: %w", err)
	}

	// Generar filename base
	filenameBase := y.generateFilename(dl)

	// Construir argumentos
	args := []string{
		"-o", filepath.Join(platformDir, filenameBase+".%(ext)s"),
	}

	// Opciones según configuración
	if dl.Options.AudioOnly {
		args = append(args, "-x", "--audio-format", "mp3")
	} else {
		// Formato de video
		format := y.buildFormatString(dl.Options.Resolution)
		args = append(args, "-f", format)
		args = append(args, "--merge-output-format", "mp4")
	}

	// Cookies: siempre buscar cuenta activa para la plataforma
	if y.accountRepo != nil {
		account, err := y.accountRepo.GetActive(ctx, dl.Platform)
		if err == nil && account != nil && account.CookiePath != "" {
			args = append(args, "--cookies", account.CookiePath)
		}
	}

	// Opciones adicionales
	args = append(args,
		"--no-check-certificate",
		"--no-playlist", // Por defecto no descargar playlists
		"--restrict-filenames", // POSIX-compliant filenames
	)

	// URL al final
	args = append(args, dl.URL)

	// Ejecutar yt-dlp
	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return "", fmt.Errorf("yt-dlp failed: %w\nOutput: %s", err, output)
	}

	// Buscar el archivo descargado
	outputPath, err := y.findDownloadedFile(platformDir, filenameBase)
	if err != nil {
		return "", fmt.Errorf("find downloaded file: %w\nyt-dlp output: %s", err, output)
	}

	return outputPath, nil
}

// Supports verifica si yt-dlp soporta la URL
func (y *YtDlp) Supports(url string) bool {
	// yt-dlp soporta todo excepto las plataformas específicas de gallery-dl
	return !NeedsGalleryDL(url)
}

// generateFilename genera el nombre de archivo base
func (y *YtDlp) generateFilename(dl *domain.Download) string {
	// Formato: platform_username_DDMMYYYY_### para redes sociales
	// Formato: platform_DDMMYYYY_###_%(title)s para YouTube
	timestamp := time.Now().Format("02012006")

	if dl.Platform == "youtube" {
		// YouTube: incluir título del video
		return dl.Platform + "_" + timestamp + "_%(title)s"
	}

	// Otras plataformas: incluir username
	username := dl.Username
	if username == "" {
		username = "user"
	}

	return fmt.Sprintf("%s_%s_%s", dl.Platform, username, timestamp)
}

// buildFormatString construye el string de formato según opciones
func (y *YtDlp) buildFormatString(resolution string) string {
	switch resolution {
	case "1080p":
		return "bestvideo[height<=1080][ext=mp4]+bestaudio[ext=m4a]/best[height<=1080][ext=mp4]/best"
	case "720p":
		return "bestvideo[height<=720][ext=mp4]+bestaudio[ext=m4a]/best[height<=720][ext=mp4]/best"
	case "480p":
		return "bestvideo[height<=480][ext=mp4]+bestaudio[ext=m4a]/best[height<=480][ext=mp4]/best"
	default:
		// Best quality
		return "bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best"
	}
}

// findDownloadedFile busca el archivo descargado en el directorio
func (y *YtDlp) findDownloadedFile(dir, basePattern string) (string, error) {
	// Buscar archivos que coincidan con el patrón
	// yt-dlp puede haber agregado sufijos o modificado el nombre

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

// CheckInstalled verifica si yt-dlp está instalado
func CheckYtDlpInstalled() error {
	cmd := exec.Command("yt-dlp", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("yt-dlp not found: %w (install: pip install yt-dlp)", err)
	}
	return nil
}
