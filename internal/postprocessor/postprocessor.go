package postprocessor

import (
	"context"

	"github.com/elsanchez/smart-download/internal/domain"
)

// PostProcessor define la interfaz para procesamiento post-descarga
type PostProcessor interface {
	// Process aplica el procesamiento configurado al archivo
	Process(ctx context.Context, inputPath string, options *domain.DownloadOptions) (outputPath string, err error)

	// NeedsProcessing verifica si el archivo necesita procesamiento
	NeedsProcessing(inputPath string, options *domain.DownloadOptions) (bool, error)
}

// ProcessingResult contiene el resultado del procesamiento
type ProcessingResult struct {
	OutputPath    string
	OriginalSize  int64
	ProcessedSize int64
	Codec         string
	Resolution    string
	Operations    []string // Lista de operaciones aplicadas
}
