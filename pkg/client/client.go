package client

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
)

// GetDefaultSocketPath retorna el path del socket usando XDG_RUNTIME_DIR
// Desktop Linux con systemd siempre tiene esta variable
func GetDefaultSocketPath() string {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		// Fallback: construir con UID (aunque no debería ocurrir en desktop Linux moderno)
		uid := os.Getuid()
		runtimeDir = fmt.Sprintf("/run/user/%d", uid)
	}

	return filepath.Join(runtimeDir, "smart-download.sock")
}

// Client representa un cliente del daemon
type Client struct {
	socketPath string
}

// NewClient crea un cliente con socket path personalizado
func NewClient(socketPath string) *Client {
	return &Client{socketPath: socketPath}
}

// NewDefaultClient crea un cliente con el socket path por defecto
func NewDefaultClient() *Client {
	return &Client{socketPath: GetDefaultSocketPath()}
}

// Request representa una petición al daemon
type Request struct {
	Action  string          `json:"action"`
	Payload json.RawMessage `json:"payload"`
}

// Response representa una respuesta del daemon
type Response struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// Send envía una petición al daemon y retorna la respuesta
func (c *Client) Send(req *Request) (*Response, error) {
	// Conectar al socket
	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return nil, fmt.Errorf("connect to daemon: %w (is daemon running?)", err)
	}
	defer conn.Close()

	// Enviar request
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	// Leer response
	var resp Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &resp, nil
}

// AddDownloadPayload representa el payload para añadir una descarga
type AddDownloadPayload struct {
	URL        string                 `json:"url"`
	Options    map[string]interface{} `json:"options,omitempty"`
	AccountID  *int64                 `json:"account_id,omitempty"`
	Background bool                   `json:"background,omitempty"`
}

// AddDownload añade una descarga a la cola
func (c *Client) AddDownload(payload *AddDownloadPayload) (int64, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("marshal payload: %w", err)
	}

	resp, err := c.Send(&Request{
		Action:  "add",
		Payload: payloadJSON,
	})
	if err != nil {
		return 0, err
	}

	if !resp.Success {
		return 0, fmt.Errorf("add download failed: %s", resp.Error)
	}

	var result struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return 0, fmt.Errorf("unmarshal response: %w", err)
	}

	return result.ID, nil
}

// GetDownloadStatus obtiene el status de una descarga
func (c *Client) GetDownloadStatus(id int64) (string, error) {
	payload, _ := json.Marshal(map[string]int64{"id": id})

	resp, err := c.Send(&Request{
		Action:  "status",
		Payload: payload,
	})
	if err != nil {
		return "", err
	}

	if !resp.Success {
		return "", fmt.Errorf("get status failed: %s", resp.Error)
	}

	var result struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	return result.Status, nil
}

// ListRecentDownloads lista las descargas recientes
func (c *Client) ListRecentDownloads(limit int) ([]map[string]interface{}, error) {
	payload, _ := json.Marshal(map[string]int{"limit": limit})

	resp, err := c.Send(&Request{
		Action:  "list",
		Payload: payload,
	})
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("list failed: %s", resp.Error)
	}

	var result struct {
		Downloads []map[string]interface{} `json:"downloads"`
	}
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return result.Downloads, nil
}
