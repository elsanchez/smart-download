package downloader

import (
	"context"

	"github.com/elsanchez/smart-download/internal/domain"
)

// Downloader define la interfaz para descargar contenido
type Downloader interface {
	// Download ejecuta la descarga y retorna el path del archivo descargado
	Download(ctx context.Context, dl *domain.Download) (outputPath string, err error)

	// Supports verifica si el downloader soporta la URL
	Supports(url string) bool
}

// Result representa el resultado de una descarga
type Result struct {
	OutputPath string
	FileSize   int64
	Duration   float64 // segundos
}
