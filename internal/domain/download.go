package domain

import "time"

// DownloadStatus representa los estados posibles de una descarga
type DownloadStatus string

const (
	StatusPending     DownloadStatus = "pending"
	StatusDownloading DownloadStatus = "downloading"
	StatusProcessing  DownloadStatus = "processing"
	StatusCompleted   DownloadStatus = "completed"
	StatusFailed      DownloadStatus = "failed"
)

// Download representa una descarga en el sistema
type Download struct {
	ID           int64
	URL          string
	Platform     string
	Username     string
	Status       DownloadStatus
	OutputPath   string
	Options      DownloadOptions
	AccountID    *int64
	CreatedAt    time.Time
	CompletedAt  *time.Time
	ErrorMessage string
}

// DownloadOptions contiene las opciones de procesamiento
type DownloadOptions struct {
	// Descarga
	Resolution string `json:"resolution,omitempty"` // 1080p, 720p, 480p
	AudioOnly  bool   `json:"audio_only,omitempty"`

	// Clipping
	ClipStart string `json:"clip_start,omitempty"` // Formato: HH:MM:SS o SS
	ClipEnd   string `json:"clip_end,omitempty"`   // Formato: HH:MM:SS o SS

	// Conversión a GIF
	ConvertToGIF bool `json:"convert_to_gif,omitempty"`
	GIFWidth     int  `json:"gif_width,omitempty"` // Default: 480px

	// Post-procesamiento
	NoConvert bool `json:"no_convert,omitempty"` // Desactivar conversión automática a WhatsApp MP4
}

// IsCompleted retorna true si la descarga está completa o falló
func (d *Download) IsCompleted() bool {
	return d.Status == StatusCompleted || d.Status == StatusFailed
}

// IsActive retorna true si la descarga está en proceso
func (d *Download) IsActive() bool {
	return d.Status == StatusDownloading || d.Status == StatusProcessing
}
