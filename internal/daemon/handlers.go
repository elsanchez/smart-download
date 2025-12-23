package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/elsanchez/smart-download/internal/domain"
	"github.com/elsanchez/smart-download/internal/downloader"
	"github.com/elsanchez/smart-download/internal/repository"
)

// Handlers maneja las peticiones del servidor
type Handlers struct {
	downloadRepo repository.DownloadRepository
	accountRepo  repository.AccountRepository
	queue        *QueueManager
}

// NewHandlers crea un nuevo conjunto de handlers
func NewHandlers(
	downloadRepo repository.DownloadRepository,
	accountRepo repository.AccountRepository,
	queue *QueueManager,
) *Handlers {
	return &Handlers{
		downloadRepo: downloadRepo,
		accountRepo:  accountRepo,
		queue:        queue,
	}
}

// AddDownloadPayload es el payload para añadir una descarga
type AddDownloadPayload struct {
	URL       string                 `json:"url"`
	Options   *domain.DownloadOptions `json:"options,omitempty"`
	AccountID *int64                 `json:"account_id,omitempty"`
}

// HandleAdd maneja la petición de añadir una descarga
func (h *Handlers) HandleAdd(ctx context.Context, payload json.RawMessage) Response {
	var req AddDownloadPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		return Response{Success: false, Error: fmt.Sprintf("invalid payload: %v", err)}
	}

	if req.URL == "" {
		return Response{Success: false, Error: "url is required"}
	}

	// Detectar plataforma y username
	platform := downloader.DetectPlatform(req.URL)
	username := downloader.ExtractUsername(req.URL)

	// Crear descarga
	dl := &domain.Download{
		URL:       req.URL,
		Platform:  platform,
		Username:  username,
		Status:    domain.StatusPending,
		AccountID: req.AccountID,
		CreatedAt: time.Now(),
	}

	// Opciones (usar defaults si no se especifican)
	if req.Options != nil {
		dl.Options = *req.Options
	} else {
		dl.Options = domain.DownloadOptions{}
	}

	// Insertar en base de datos
	id, err := h.downloadRepo.Create(ctx, dl)
	if err != nil {
		return Response{Success: false, Error: fmt.Sprintf("create download: %v", err)}
	}

	// Respuesta
	data, _ := json.Marshal(map[string]interface{}{
		"id":       id,
		"platform": platform,
		"username": username,
		"status":   domain.StatusPending,
	})

	return Response{Success: true, Data: data}
}

// StatusPayload es el payload para consultar status
type StatusPayload struct {
	ID int64 `json:"id"`
}

// HandleStatus maneja la petición de status
func (h *Handlers) HandleStatus(ctx context.Context, payload json.RawMessage) Response {
	var req StatusPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		return Response{Success: false, Error: fmt.Sprintf("invalid payload: %v", err)}
	}

	if req.ID == 0 {
		return Response{Success: false, Error: "id is required"}
	}

	// Obtener descarga
	dl, err := h.downloadRepo.GetByID(ctx, req.ID)
	if err != nil {
		return Response{Success: false, Error: fmt.Sprintf("get download: %v", err)}
	}

	// Respuesta
	data, _ := json.Marshal(map[string]interface{}{
		"id":            dl.ID,
		"url":           dl.URL,
		"platform":      dl.Platform,
		"username":      dl.Username,
		"status":        dl.Status,
		"output_path":   dl.OutputPath,
		"created_at":    dl.CreatedAt,
		"completed_at":  dl.CompletedAt,
		"error_message": dl.ErrorMessage,
	})

	return Response{Success: true, Data: data}
}

// ListPayload es el payload para listar descargas
type ListPayload struct {
	Limit int `json:"limit"`
}

// HandleList maneja la petición de listar descargas
func (h *Handlers) HandleList(ctx context.Context, payload json.RawMessage) Response {
	var req ListPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		// Si no hay payload, usar default
		req.Limit = 50
	}

	if req.Limit <= 0 {
		req.Limit = 50
	}

	// Obtener descargas recientes
	downloads, err := h.downloadRepo.GetRecent(ctx, req.Limit)
	if err != nil {
		return Response{Success: false, Error: fmt.Sprintf("get downloads: %v", err)}
	}

	// Convertir a formato de respuesta
	items := make([]map[string]interface{}, 0, len(downloads))
	for _, dl := range downloads {
		items = append(items, map[string]interface{}{
			"id":            dl.ID,
			"url":           dl.URL,
			"platform":      dl.Platform,
			"username":      dl.Username,
			"status":        dl.Status,
			"output_path":   dl.OutputPath,
			"created_at":    dl.CreatedAt,
			"completed_at":  dl.CompletedAt,
			"error_message": dl.ErrorMessage,
		})
	}

	data, _ := json.Marshal(map[string]interface{}{
		"downloads": items,
		"count":     len(items),
	})

	return Response{Success: true, Data: data}
}

// HandleStats maneja la petición de estadísticas
func (h *Handlers) HandleStats(ctx context.Context) Response {
	stats, err := h.queue.GetStats(ctx)
	if err != nil {
		return Response{Success: false, Error: fmt.Sprintf("get stats: %v", err)}
	}

	data, _ := json.Marshal(stats)
	return Response{Success: true, Data: data}
}
