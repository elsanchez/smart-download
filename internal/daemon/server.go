package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
)

// Server es el servidor Unix socket
type Server struct {
	socketPath string
	listener   net.Listener
	queue      *QueueManager
	handlers   *Handlers
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

// NewServer crea un nuevo servidor
func NewServer(socketPath string, queue *QueueManager, handlers *Handlers) *Server {
	return &Server{
		socketPath: socketPath,
		queue:      queue,
		handlers:   handlers,
	}
}

// Start inicia el servidor
func (s *Server) Start(ctx context.Context) error {
	// Crear directorio para socket
	dir := filepath.Dir(s.socketPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create socket dir: %w", err)
	}

	// Limpiar socket anterior si existe
	os.Remove(s.socketPath)

	// Crear listener
	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("listen on socket: %w", err)
	}
	s.listener = listener

	// Permisos del socket
	if err := os.Chmod(s.socketPath, 0600); err != nil {
		return fmt.Errorf("chmod socket: %w", err)
	}

	log.Printf("Server listening on %s", s.socketPath)

	// Accept loop
	go s.acceptLoop(ctx)

	return nil
}

// acceptLoop acepta conexiones entrantes
func (s *Server) acceptLoop(ctx context.Context) {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				log.Printf("Accept error: %v", err)
				continue
			}
		}

		go s.handleConnection(ctx, conn)
	}
}

// handleConnection maneja una conexión individual
func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	var req Request
	decoder := json.NewDecoder(conn)

	if err := decoder.Decode(&req); err != nil {
		s.sendError(conn, fmt.Errorf("decode request: %w", err))
		return
	}

	log.Printf("Received request: action=%s", req.Action)

	// Routing
	var resp Response
	switch req.Action {
	case "add":
		resp = s.handlers.HandleAdd(ctx, req.Payload)
	case "status":
		resp = s.handlers.HandleStatus(ctx, req.Payload)
	case "list":
		resp = s.handlers.HandleList(ctx, req.Payload)
	case "stats":
		resp = s.handlers.HandleStats(ctx)
	case "ping":
		resp = Response{Success: true, Data: json.RawMessage(`{"message":"pong"}`)}
	default:
		resp = Response{Success: false, Error: fmt.Sprintf("unknown action: %s", req.Action)}
	}

	// Enviar respuesta
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(resp); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

// sendError envía una respuesta de error
func (s *Server) sendError(conn net.Conn, err error) {
	resp := Response{
		Success: false,
		Error:   err.Error(),
	}
	json.NewEncoder(conn).Encode(resp)
}

// Stop detiene el servidor
func (s *Server) Stop() error {
	log.Println("Server stopping...")
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}
