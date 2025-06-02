package handlers

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/unikyri/escritorio-remoto-backend/internal/application/userservice"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/user"
	"github.com/unikyri/escritorio-remoto-backend/internal/presentation/dto"
)

// AdminConnection representa una conexión WebSocket de administrador
type AdminConnection struct {
	ID       string
	UserID   string
	Username string
	Role     string
	IsAuth   bool
	Conn     *websocket.Conn
	LastSeen time.Time
}

// AdminWebSocketHandler maneja las conexiones WebSocket de administradores
type AdminWebSocketHandler struct {
	authService *userservice.AuthService
	upgrader    websocket.Upgrader

	// Mapa de conexiones de administradores
	adminConnections map[string]*AdminConnection
	mutex            sync.RWMutex
}

// NewAdminWebSocketHandler crea un nuevo handler de WebSocket para administradores
func NewAdminWebSocketHandler(authService *userservice.AuthService) *AdminWebSocketHandler {
	return &AdminWebSocketHandler{
		authService: authService,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // En producción, verificar origen
			},
		},
		adminConnections: make(map[string]*AdminConnection),
	}
}

// HandleAdminWebSocket maneja las conexiones WebSocket de administradores
func (h *AdminWebSocketHandler) HandleAdminWebSocket(c *gin.Context) {
	// Verificar autenticación JWT
	userInfo, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	userClaims, ok := userInfo.(*userservice.JWTClaims)
	if !ok || userClaims.Role != string(user.RoleAdministrator) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}

	// Actualizar WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Error upgrading to WebSocket: %v", err)
		return
	}
	defer conn.Close()

	// Crear conexión de administrador
	adminConn := &AdminConnection{
		ID:       generateConnectionID(),
		UserID:   userClaims.UserID,
		Username: userClaims.Username,
		Role:     userClaims.Role,
		IsAuth:   true,
		Conn:     conn,
		LastSeen: time.Now(),
	}

	// Registrar conexión
	h.mutex.Lock()
	h.adminConnections[adminConn.ID] = adminConn
	h.mutex.Unlock()

	log.Printf("Admin connected: %s (%s)", adminConn.Username, adminConn.ID)

	// Enviar mensaje de bienvenida
	welcomeMsg := dto.WebSocketMessage{
		Type: "admin_connected",
		Data: map[string]interface{}{
			"message": "Connected to admin notifications",
			"adminId": adminConn.ID,
		},
	}
	conn.WriteJSON(welcomeMsg)

	// Manejar mensajes
	defer func() {
		h.mutex.Lock()
		delete(h.adminConnections, adminConn.ID)
		h.mutex.Unlock()
		log.Printf("Admin disconnected: %s (%s)", adminConn.Username, adminConn.ID)
	}()

	// Configurar timeouts
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Loop de lectura de mensajes
	for {
		var message dto.WebSocketMessage
		err := conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Actualizar último visto
		adminConn.LastSeen = time.Now()

		// Procesar mensaje
		h.handleAdminMessage(adminConn, message)
	}
}

// handleAdminMessage procesa mensajes de administradores
func (h *AdminWebSocketHandler) handleAdminMessage(adminConn *AdminConnection, message dto.WebSocketMessage) {
	switch message.Type {
	case "ping":
		// Responder con pong
		pongMsg := dto.WebSocketMessage{
			Type: "pong",
			Data: map[string]interface{}{
				"timestamp": time.Now().Unix(),
			},
		}
		adminConn.Conn.WriteJSON(pongMsg)

	case "get_pc_list":
		// Solicitar lista actualizada de PCs (esto se puede implementar más tarde)
		log.Printf("Admin %s requested PC list", adminConn.Username)

	default:
		log.Printf("Unknown message type from admin %s: %s", adminConn.Username, message.Type)
	}
}

// BroadcastPCConnected notifica a todos los administradores que un PC se conectó
func (h *AdminWebSocketHandler) BroadcastPCConnected(pcID, identifier, ownerUserID, ip string) {
	notification := dto.WebSocketMessage{
		Type: "pc_connected",
		Data: map[string]interface{}{
			"pcId":        pcID,
			"identifier":  identifier,
			"ownerUserId": ownerUserID,
			"ip":          ip,
			"timestamp":   time.Now().Unix(),
		},
	}

	h.broadcastToAllAdmins(notification)
	log.Printf("Broadcasted PC connected: %s (%s)", identifier, pcID)
}

// BroadcastPCDisconnected notifica a todos los administradores que un PC se desconectó
func (h *AdminWebSocketHandler) BroadcastPCDisconnected(pcID, identifier, ownerUserID string) {
	notification := dto.WebSocketMessage{
		Type: "pc_disconnected",
		Data: map[string]interface{}{
			"pcId":        pcID,
			"identifier":  identifier,
			"ownerUserId": ownerUserID,
			"timestamp":   time.Now().Unix(),
		},
	}

	h.broadcastToAllAdmins(notification)
	log.Printf("Broadcasted PC disconnected: %s (%s)", identifier, pcID)
}

// BroadcastPCRegistered notifica a todos los administradores que un PC se registró
func (h *AdminWebSocketHandler) BroadcastPCRegistered(pcID, identifier, ownerUserID, ip string) {
	notification := dto.WebSocketMessage{
		Type: "pc_registered",
		Data: map[string]interface{}{
			"pcId":        pcID,
			"identifier":  identifier,
			"ownerUserId": ownerUserID,
			"ip":          ip,
			"timestamp":   time.Now().Unix(),
		},
	}

	h.broadcastToAllAdmins(notification)
	log.Printf("Broadcasted PC registered: %s (%s)", identifier, pcID)
}

// BroadcastPCStatusChanged notifica cambios de estado de PC
func (h *AdminWebSocketHandler) BroadcastPCStatusChanged(pcID, identifier, oldStatus, newStatus string) {
	notification := dto.WebSocketMessage{
		Type: "pc_status_changed",
		Data: map[string]interface{}{
			"pcId":       pcID,
			"identifier": identifier,
			"oldStatus":  oldStatus,
			"newStatus":  newStatus,
			"timestamp":  time.Now().Unix(),
		},
	}

	h.broadcastToAllAdmins(notification)
	log.Printf("Broadcasted PC status change: %s (%s) %s -> %s", identifier, pcID, oldStatus, newStatus)
}

// broadcastToAllAdmins envía un mensaje a todos los administradores conectados
func (h *AdminWebSocketHandler) broadcastToAllAdmins(message dto.WebSocketMessage) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for connID, adminConn := range h.adminConnections {
		err := adminConn.Conn.WriteJSON(message)
		if err != nil {
			log.Printf("Error sending message to admin %s (%s): %v", adminConn.Username, connID, err)
			// La conexión se limpiará en el defer del handler principal
		}
	}
}

// GetConnectedAdmins retorna la lista de administradores conectados
func (h *AdminWebSocketHandler) GetConnectedAdmins() map[string]*AdminConnection {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	result := make(map[string]*AdminConnection)
	for id, conn := range h.adminConnections {
		result[id] = conn
	}
	return result
}

// GetAdminCount retorna el número de administradores conectados
func (h *AdminWebSocketHandler) GetAdminCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.adminConnections)
}
