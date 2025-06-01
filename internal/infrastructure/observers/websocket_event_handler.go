package observers

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/clientpc/events"
	sharedEvents "github.com/unikyri/escritorio-remoto-backend/internal/domain/shared/events"
)

// WebSocketEventHandler maneja eventos de dominio y los envía via WebSocket
type WebSocketEventHandler struct {
	connections map[string]*websocket.Conn
	mutex       sync.RWMutex
}

// NewWebSocketEventHandler crea un nuevo handler de WebSocket
func NewWebSocketEventHandler() *WebSocketEventHandler {
	return &WebSocketEventHandler{
		connections: make(map[string]*websocket.Conn),
		mutex:       sync.RWMutex{},
	}
}

// Handle procesa un evento de dominio
func (h *WebSocketEventHandler) Handle(ctx context.Context, event sharedEvents.DomainEvent) error {
	switch event.Type() {
	case events.PCConnectedEventType:
		return h.handlePCConnectedEvent(ctx, event)
	case events.PCDisconnectedEventType:
		return h.handlePCDisconnectedEvent(ctx, event)
	default:
		// Evento no manejado por este handler
		return nil
	}
}

// CanHandle verifica si puede manejar un tipo de evento
func (h *WebSocketEventHandler) CanHandle(eventType string) bool {
	supportedEvents := []string{
		events.PCConnectedEventType,
		events.PCDisconnectedEventType,
	}

	for _, supported := range supportedEvents {
		if eventType == supported {
			return true
		}
	}

	return false
}

// AddConnection agrega una nueva conexión WebSocket
func (h *WebSocketEventHandler) AddConnection(connectionID string, conn *websocket.Conn) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.connections[connectionID] = conn
}

// RemoveConnection remueve una conexión WebSocket
func (h *WebSocketEventHandler) RemoveConnection(connectionID string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if conn, exists := h.connections[connectionID]; exists {
		conn.Close()
		delete(h.connections, connectionID)
	}
}

// GetConnectionCount retorna el número de conexiones activas
func (h *WebSocketEventHandler) GetConnectionCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	return len(h.connections)
}

// handlePCConnectedEvent maneja eventos de PC conectado
func (h *WebSocketEventHandler) handlePCConnectedEvent(ctx context.Context, event sharedEvents.DomainEvent) error {
	pcEvent, ok := event.(*events.PCConnectedEvent)
	if !ok {
		return fmt.Errorf("invalid event type for PC connected")
	}

	message := map[string]interface{}{
		"type":      "PC_CONNECTED",
		"data":      pcEvent.GetPCData(),
		"timestamp": event.OccurredOn(),
	}

	return h.broadcastMessage(message)
}

// handlePCDisconnectedEvent maneja eventos de PC desconectado
func (h *WebSocketEventHandler) handlePCDisconnectedEvent(ctx context.Context, event sharedEvents.DomainEvent) error {
	pcEvent, ok := event.(*events.PCDisconnectedEvent)
	if !ok {
		return fmt.Errorf("invalid event type for PC disconnected")
	}

	message := map[string]interface{}{
		"type":      "PC_DISCONNECTED",
		"data":      pcEvent.GetPCData(),
		"timestamp": event.OccurredOn(),
	}

	return h.broadcastMessage(message)
}

// broadcastMessage envía un mensaje a todas las conexiones activas
func (h *WebSocketEventHandler) broadcastMessage(message interface{}) error {
	h.mutex.RLock()
	connections := make(map[string]*websocket.Conn)
	for id, conn := range h.connections {
		connections[id] = conn
	}
	h.mutex.RUnlock()

	messageBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("error marshaling message: %w", err)
	}

	// Enviar a todas las conexiones
	var failedConnections []string
	for connectionID, conn := range connections {
		if err := conn.WriteMessage(websocket.TextMessage, messageBytes); err != nil {
			// Marcar conexión como fallida para remover después
			failedConnections = append(failedConnections, connectionID)
		}
	}

	// Remover conexiones fallidas
	for _, connectionID := range failedConnections {
		h.RemoveConnection(connectionID)
	}

	return nil
}
