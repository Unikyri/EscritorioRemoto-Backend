package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/unikyri/escritorio-remoto-backend/internal/application/remotesessionservice"
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
	authService     *userservice.AuthService
	sessionService  *remotesessionservice.RemoteSessionService
	clientWSHandler *WebSocketHandler // Referencia circular manejada con puntero
	upgrader        websocket.Upgrader

	// Mapa de conexiones de administradores
	adminConnections map[string]*AdminConnection
	mutex            sync.RWMutex
}

// NewAdminWebSocketHandler crea un nuevo handler de WebSocket para administradores
func NewAdminWebSocketHandler(
	authService *userservice.AuthService,
	sessionService *remotesessionservice.RemoteSessionService,
) *AdminWebSocketHandler {
	return &AdminWebSocketHandler{
		authService:    authService,
		sessionService: sessionService,
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
	// Obtener token desde query parameter o header
	var token string

	// Intentar desde query parameter primero
	token = c.Query("token")
	if token == "" {
		// Intentar desde header Authorization
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication token required"})
		return
	}

	// Validar token directamente
	userClaims, err := h.authService.ValidateToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
		return
	}

	if userClaims.Role != string(user.RoleAdministrator) {
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

	case dto.MessageTypeInputCommand:
		// Manejar comando de input del administrador
		h.handleInputCommand(adminConn, message.Data)

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
			"status":      "ONLINE",
			"timestamp":   time.Now().Unix(),
			"event":       "connection",
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
			"status":      "OFFLINE",
			"timestamp":   time.Now().Unix(),
			"event":       "disconnection",
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
			"status":      "ONLINE",
			"timestamp":   time.Now().Unix(),
			"event":       "registration",
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
			"event":      "status_change",
		},
	}

	h.broadcastToAllAdmins(notification)
	log.Printf("Broadcasted PC status change: %s (%s) %s -> %s", identifier, pcID, oldStatus, newStatus)
}

// BroadcastPCListUpdate notifica que la lista de PCs debe actualizarse
func (h *AdminWebSocketHandler) BroadcastPCListUpdate() {
	notification := dto.WebSocketMessage{
		Type: "pc_list_update",
		Data: map[string]interface{}{
			"timestamp": time.Now().Unix(),
			"event":     "list_refresh",
			"message":   "PC list has been updated, please refresh",
		},
	}

	h.broadcastToAllAdmins(notification)
	log.Printf("Broadcasted PC list update notification")
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

// SetClientWSHandler establece la referencia al handler de clientes (para evitar dependencia circular)
func (h *AdminWebSocketHandler) SetClientWSHandler(clientHandler *WebSocketHandler) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.clientWSHandler = clientHandler
}

// handleInputCommand maneja comandos de input de administradores
func (h *AdminWebSocketHandler) handleInputCommand(adminConn *AdminConnection, data interface{}) {
	// Parse input command data
	commandData, err := json.Marshal(data)
	if err != nil {
		log.Printf("❌ INPUT COMMAND: Error marshalling command data: %v", err)
		return
	}

	var inputCommand dto.InputCommand
	if err := json.Unmarshal(commandData, &inputCommand); err != nil {
		log.Printf("❌ INPUT COMMAND: Error unmarshalling input command: %v", err)
		return
	}

	log.Printf("🎮 INPUT COMMAND: Received from admin %s: type=%s, action=%s, session=%s",
		adminConn.Username, inputCommand.EventType, inputCommand.Action, inputCommand.SessionID)

	// Validar permisos del administrador para enviar comandos
	err = h.sessionService.ValidateInputCommandPermission(inputCommand.SessionID, adminConn.UserID)
	if err != nil {
		log.Printf("❌ INPUT COMMAND: Invalid permission for admin %s: %v", adminConn.Username, err)
		return
	}

	// Obtener el PC cliente objetivo
	clientPCID, err := h.sessionService.GetClientPCIDForActiveSession(inputCommand.SessionID)
	if err != nil {
		log.Printf("❌ INPUT COMMAND: Error getting client PC for session: %v", err)
		return
	}

	// Reenviar comando al cliente a través del ClientWebSocketHandler
	if h.clientWSHandler != nil {
		err := h.clientWSHandler.SendInputCommandToClient(clientPCID, inputCommand)
		if err != nil {
			log.Printf("❌ INPUT COMMAND: Error forwarding command to client %s: %v", clientPCID, err)
		} else {
			log.Printf("✅ INPUT COMMAND: Command forwarded to client %s", clientPCID)
		}
	} else {
		log.Printf("⚠️ INPUT COMMAND: No client WebSocket handler available")
	}
}

// ForwardScreenFrameToAdmin reenvía un frame de pantalla a un administrador específico
func (h *AdminWebSocketHandler) ForwardScreenFrameToAdmin(adminUserID string, screenFrame dto.ScreenFrame) error {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	// Buscar la conexión del administrador
	var targetAdmin *AdminConnection
	for _, adminConn := range h.adminConnections {
		if adminConn.UserID == adminUserID {
			targetAdmin = adminConn
			break
		}
	}

	if targetAdmin == nil {
		return fmt.Errorf("admin %s not connected", adminUserID)
	}

	// Crear mensaje de frame de pantalla
	frameMessage := dto.WebSocketMessage{
		Type: dto.MessageTypeScreenFrame,
		Data: screenFrame,
	}

	// Enviar frame al administrador
	err := targetAdmin.Conn.WriteJSON(frameMessage)
	if err != nil {
		return fmt.Errorf("error sending frame to admin: %w", err)
	}

	return nil
}

// SendInputCommandToClientByAdmin permite que un administrador envíe comandos de input (método alternativo)
func (h *AdminWebSocketHandler) SendInputCommandToClientByAdmin(adminUserID, sessionID string, inputCommand dto.InputCommand) error {
	// Validar permisos del administrador
	err := h.sessionService.ValidateInputCommandPermission(sessionID, adminUserID)
	if err != nil {
		return fmt.Errorf("invalid permission: %w", err)
	}

	// Obtener el PC cliente objetivo
	clientPCID, err := h.sessionService.GetClientPCIDForActiveSession(sessionID)
	if err != nil {
		return fmt.Errorf("error getting client PC: %w", err)
	}

	// Reenviar comando al cliente
	if h.clientWSHandler != nil {
		return h.clientWSHandler.SendInputCommandToClient(clientPCID, inputCommand)
	}

	return fmt.Errorf("client WebSocket handler not available")
}

// NotifySessionAccepted notifica al administrador que una sesión fue aceptada
func (h *AdminWebSocketHandler) NotifySessionAccepted(sessionID string) error {
	// Obtener información de la sesión
	session, err := h.sessionService.GetSessionById(sessionID)
	if err != nil {
		return fmt.Errorf("error getting session: %w", err)
	}
	if session == nil {
		return fmt.Errorf("session not found")
	}

	// Buscar la conexión del administrador por UserID
	h.mutex.RLock()
	var adminConn *AdminConnection
	for _, conn := range h.adminConnections {
		if conn.UserID == session.AdminUserID() {
			adminConn = conn
			break
		}
	}
	h.mutex.RUnlock()

	if adminConn == nil {
		return fmt.Errorf("admin user %s not connected", session.AdminUserID())
	}

	// Crear mensaje de notificación
	notification := dto.WebSocketMessage{
		Type: "session_accepted",
		Data: map[string]interface{}{
			"session_id":   sessionID,
			"client_pc_id": session.ClientPCID(),
			"status":       string(session.Status()),
			"start_time":   session.StartTime(),
			"message":      "Client accepted remote control session",
			"timestamp":    time.Now().Unix(),
		},
	}

	// Enviar notificación
	err = adminConn.Conn.WriteJSON(notification)
	if err != nil {
		return fmt.Errorf("error sending notification to admin: %w", err)
	}

	log.Printf("✅ ADMIN NOTIFICATION: Session %s acceptance sent to admin %s", sessionID, session.AdminUserID())
	return nil
}

// NotifySessionEnded notifica al administrador que una sesión terminó
func (h *AdminWebSocketHandler) NotifySessionEnded(sessionID, clientPCID, adminUserID string) error {
	// Buscar la conexión del administrador por UserID
	h.mutex.RLock()
	var adminConn *AdminConnection
	for _, conn := range h.adminConnections {
		if conn.UserID == adminUserID {
			adminConn = conn
			break
		}
	}
	h.mutex.RUnlock()

	if adminConn == nil {
		log.Printf("⚠️ Admin user %s not connected when trying to notify session %s ended", adminUserID, sessionID)
		return nil // No es un error crítico si el admin no está conectado
	}

	// Crear mensaje de notificación
	notification := dto.WebSocketMessage{
		Type: "session_ended",
		Data: map[string]interface{}{
			"session_id":    sessionID,
			"client_pc_id":  clientPCID,
			"admin_user_id": adminUserID,
			"reason":        "Client disconnected",
			"message":       "Remote control session ended",
			"timestamp":     time.Now().Unix(),
		},
	}

	// Enviar notificación
	err := adminConn.Conn.WriteJSON(notification)
	if err != nil {
		return fmt.Errorf("error sending session ended notification to admin: %w", err)
	}

	log.Printf("✅ ADMIN NOTIFICATION: Session %s ended notification sent to admin %s", sessionID, adminUserID)
	return nil
}
