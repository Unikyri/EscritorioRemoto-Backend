package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/unikyri/escritorio-remoto-backend/internal/application/pcservice"
	"github.com/unikyri/escritorio-remoto-backend/internal/application/remotesessionservice"
	"github.com/unikyri/escritorio-remoto-backend/internal/application/userservice"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/clientpc"
	"github.com/unikyri/escritorio-remoto-backend/internal/presentation/dto"
)

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin for now
		// In production, you should validate the origin
		return true
	},
}

// ClientConnection represents an active WebSocket connection
type ClientConnection struct {
	Conn       *websocket.Conn
	UserID     string
	Username   string
	Role       string
	PCID       string
	IsAuth     bool
	LastSeen   time.Time
	RemoteAddr string
}

// WebSocketHandler manages WebSocket connections for client PCs
type WebSocketHandler struct {
	authService    *userservice.AuthService
	pcService      pcservice.IPCService
	sessionService *remotesessionservice.RemoteSessionService
	adminWSHandler *AdminWebSocketHandler
	connections    map[string]*ClientConnection // map[connectionID]*ClientConnection
	pcConnections  map[string]*ClientConnection // map[pcID]*ClientConnection
	mutex          sync.RWMutex
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(
	authService *userservice.AuthService,
	pcService pcservice.IPCService,
	sessionService *remotesessionservice.RemoteSessionService,
	adminWSHandler *AdminWebSocketHandler,
) *WebSocketHandler {
	return &WebSocketHandler{
		authService:    authService,
		pcService:      pcService,
		sessionService: sessionService,
		adminWSHandler: adminWSHandler,
		connections:    make(map[string]*ClientConnection),
		pcConnections:  make(map[string]*ClientConnection),
		mutex:          sync.RWMutex{},
	}
}

// HandleWebSocket handles WebSocket connections
func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	// Get client IP
	clientIP := getClientIP(c.Request)

	// Create connection object
	connectionID := generateConnectionID()
	clientConn := &ClientConnection{
		Conn:       conn,
		IsAuth:     false,
		LastSeen:   time.Now(),
		RemoteAddr: clientIP,
	}

	// Add to connections map
	h.mutex.Lock()
	h.connections[connectionID] = clientConn
	h.mutex.Unlock()

	// Clean up on exit
	defer func() {
		h.mutex.Lock()
		delete(h.connections, connectionID)
		if clientConn.PCID != "" {
			delete(h.pcConnections, clientConn.PCID)

			// 🔄 Intentar finalizar/rechazar sesiones activas/pendientes para este PC
			log.Printf("⚡ Calling HandleClientPCDisconnect for PCID: %s", clientConn.PCID)
			if err := h.sessionService.HandleClientPCDisconnect(clientConn.PCID); err != nil {
				// Loguear el error, pero no hacer que la desconexión falle por esto.
				// El servicio HandleClientPCDisconnect ya loguea sus propios errores críticos.
				log.Printf("⚠️ Error calling HandleClientPCDisconnect for PC %s: %v", clientConn.PCID, err)
			}

			// Marcar PC como offline cuando se cierra la conexión
			if clientConn.IsAuth {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				// Obtener información del PC antes de marcarlo como offline
				currentPC, err := h.pcService.GetPCByID(ctx, clientConn.PCID)
				var pcIdentifier string
				if err == nil && currentPC != nil {
					pcIdentifier = currentPC.Identifier
				}

				// Obtener estado anterior del PC para notificación
				oldStatus := "ONLINE"

				err = h.pcService.UpdatePCConnectionStatus(ctx, clientConn.PCID, clientpc.PCConnectionStatusOffline)
				if err != nil {
					log.Printf("Error updating PC status to offline: %v", err)
				} else {
					log.Printf("PC marked as offline: %s (%s)", clientConn.PCID, clientConn.Username)

					// Notificar a administradores sobre la desconexión del PC
					if h.adminWSHandler != nil {
						h.adminWSHandler.BroadcastPCDisconnected(clientConn.PCID, pcIdentifier, clientConn.UserID)
						// Notificar cambio de estado específico
						h.adminWSHandler.BroadcastPCStatusChanged(clientConn.PCID, pcIdentifier, oldStatus, "OFFLINE")
						// Notificar actualización general de la lista
						h.adminWSHandler.BroadcastPCListUpdate()
					}
				}
			}
		}
		h.mutex.Unlock()
		log.Printf("Client disconnected: %s (Username: %s, PCID: %s)", connectionID, clientConn.Username, clientConn.PCID)
	}()

	log.Printf("New WebSocket connection: %s from %s", connectionID, clientIP)

	// Handle messages
	for {
		var message dto.WebSocketMessage
		err := conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Update last seen
		clientConn.LastSeen = time.Now()

		// Handle message based on type
		switch message.Type {
		case dto.MessageTypeClientAuth:
			h.handleClientAuth(conn, clientConn, message.Data)
		case dto.MessageTypePCRegistration:
			h.handlePCRegistration(conn, clientConn, message.Data, clientIP)
		case dto.MessageTypeHeartbeat:
			h.handleHeartbeat(conn, clientConn, message.Data)
		case dto.MessageTypeScreenFrame:
			h.handleScreenFrame(conn, clientConn, message.Data)
		case "session_accepted":
			h.handleSessionAccepted(conn, clientConn, message.Data)
		case "session_rejected":
			h.handleSessionRejected(conn, clientConn, message.Data)
		default:
			log.Printf("Unknown message type: %s", message.Type)
		}
	}
}

// handleClientAuth handles client authentication
func (h *WebSocketHandler) handleClientAuth(conn *websocket.Conn, clientConn *ClientConnection, data interface{}) {
	// Parse authentication request
	authData, err := json.Marshal(data)
	if err != nil {
		h.sendAuthResponse(conn, false, "", "", "Invalid request format")
		return
	}

	var authReq dto.ClientAuthRequest
	if err := json.Unmarshal(authData, &authReq); err != nil {
		h.sendAuthResponse(conn, false, "", "", "Invalid request format")
		return
	}

	// Authenticate user
	token, user, err := h.authService.AuthenticateClient(authReq.Username, authReq.Password)
	if err != nil {
		h.sendAuthResponse(conn, false, "", "", "Authentication failed")
		return
	}

	// Update connection with user info
	clientConn.UserID = user.UserID()
	clientConn.Username = user.Username()
	clientConn.Role = string(user.Role())
	clientConn.IsAuth = true

	// Send success response
	h.sendAuthResponse(conn, true, token, user.UserID(), "")
	log.Printf("Client authenticated: %s (%s)", user.Username(), user.UserID())
}

// handlePCRegistration handles PC registration
func (h *WebSocketHandler) handlePCRegistration(conn *websocket.Conn, clientConn *ClientConnection, data interface{}, clientIP string) {
	// Check if client is authenticated
	if !clientConn.IsAuth {
		h.sendPCRegistrationResponse(conn, false, "", "Authentication required")
		return
	}

	// Parse registration request
	regData, err := json.Marshal(data)
	if err != nil {
		h.sendPCRegistrationResponse(conn, false, "", "Invalid request format")
		return
	}

	var regReq dto.PCRegistrationRequest
	if err := json.Unmarshal(regData, &regReq); err != nil {
		h.sendPCRegistrationResponse(conn, false, "", "Invalid request format")
		return
	}

	// Use provided IP or detect from connection
	ip := regReq.IP
	if ip == "" {
		ip = clientIP
	}

	// Register PC
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pc, err := h.pcService.RegisterPC(ctx, clientConn.UserID, regReq.PCIdentifier, ip)
	if err != nil {
		h.sendPCRegistrationResponse(conn, false, "", err.Error())
		return
	}

	// Update connection with PC info
	clientConn.PCID = pc.PCID

	// Add to PC connections map
	h.mutex.Lock()
	h.pcConnections[pc.PCID] = clientConn
	h.mutex.Unlock()

	// Notificar a administradores sobre el registro del PC
	if h.adminWSHandler != nil {
		h.adminWSHandler.BroadcastPCRegistered(pc.PCID, pc.Identifier, pc.OwnerUserID, pc.IP)

		// También notificar que el PC está ahora ONLINE ya que se acaba de conectar
		log.Printf("PC registered and online: %s (%s) for user %s", pc.Identifier, pc.PCID, clientConn.Username)
		h.adminWSHandler.BroadcastPCConnected(pc.PCID, pc.Identifier, pc.OwnerUserID, pc.IP)
		h.adminWSHandler.BroadcastPCStatusChanged(pc.PCID, pc.Identifier, "OFFLINE", "ONLINE")

		// Notificar actualización general de la lista
		h.adminWSHandler.BroadcastPCListUpdate()
	}

	// Send success response
	h.sendPCRegistrationResponse(conn, true, pc.PCID, "")
	log.Printf("PC registered: %s (%s) for user %s", regReq.PCIdentifier, pc.PCID, clientConn.Username)
}

// handleHeartbeat handles heartbeat messages
func (h *WebSocketHandler) handleHeartbeat(conn *websocket.Conn, clientConn *ClientConnection, data interface{}) {
	// Parse heartbeat request
	hbData, err := json.Marshal(data)
	if err != nil {
		return
	}

	var hbReq dto.HeartbeatRequest
	if err := json.Unmarshal(hbData, &hbReq); err != nil {
		return
	}

	// Update last seen and ensure PC is online if PC is registered
	if clientConn.PCID != "" && clientConn.IsAuth {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Primero verificar el estado actual del PC para detectar cambios
		currentPC, err := h.pcService.GetPCByID(ctx, clientConn.PCID)
		var wasOffline bool
		if err == nil && currentPC != nil {
			wasOffline = currentPC.ConnectionStatus == clientpc.PCConnectionStatusOffline
		}

		// Actualizar último visto
		if err := h.pcService.UpdatePCLastSeen(ctx, clientConn.PCID); err != nil {
			log.Printf("Error updating PC last seen: %v", err)
		}

		// Asegurar que el PC esté marcado como online
		if err := h.pcService.UpdatePCConnectionStatus(ctx, clientConn.PCID, clientpc.PCConnectionStatusOnline); err != nil {
			log.Printf("Error updating PC status to online: %v", err)
		} else {
			// Si el PC estaba offline y ahora está online, notificar el cambio
			if wasOffline && h.adminWSHandler != nil {
				log.Printf("PC reconnected: %s (%s)", clientConn.PCID, clientConn.Username)

				// Obtener información actualizada del PC para las notificaciones
				updatedPC, err := h.pcService.GetPCByID(ctx, clientConn.PCID)
				if err == nil && updatedPC != nil {
					// Notificar reconexión del PC con información completa
					h.adminWSHandler.BroadcastPCConnected(updatedPC.PCID, updatedPC.Identifier, updatedPC.OwnerUserID, updatedPC.IP)
					// Notificar cambio de estado específico
					h.adminWSHandler.BroadcastPCStatusChanged(updatedPC.PCID, updatedPC.Identifier, "OFFLINE", "ONLINE")
					// Notificar actualización general de la lista
					h.adminWSHandler.BroadcastPCListUpdate()
				}
			}
		}
	}

	// Send heartbeat response
	response := dto.WebSocketMessage{
		Type: dto.MessageTypeHeartbeatResp,
		Data: dto.HeartbeatResponse{
			Timestamp: time.Now().Unix(),
			Status:    "OK",
		},
	}

	conn.WriteJSON(response)
}

// handleScreenFrame maneja frames de pantalla recibidos de clientes
func (h *WebSocketHandler) handleScreenFrame(conn *websocket.Conn, clientConn *ClientConnection, data interface{}) {
	if !clientConn.IsAuth {
		log.Printf("❌ SCREEN FRAME: Unauthorized client attempted to send frame")
		return
	}

	if clientConn.PCID == "" {
		log.Printf("❌ SCREEN FRAME: Client not registered, cannot process frame")
		return
	}

	// Parse screen frame data
	frameData, err := json.Marshal(data)
	if err != nil {
		log.Printf("❌ SCREEN FRAME: Error marshalling frame data: %v", err)
		return
	}

	var screenFrame dto.ScreenFrame
	if err := json.Unmarshal(frameData, &screenFrame); err != nil {
		log.Printf("❌ SCREEN FRAME: Error unmarshalling screen frame: %v", err)
		return
	}

	log.Printf("📹 SCREEN FRAME: Received frame %d from PC %s (session: %s, size: %dx%d)",
		screenFrame.SequenceNum, clientConn.PCID, screenFrame.SessionID, screenFrame.Width, screenFrame.Height)

	// Validar que la sesión está activa y el PC tiene permisos
	err = h.sessionService.ValidateStreamingPermission(screenFrame.SessionID, clientConn.PCID)
	if err != nil {
		log.Printf("❌ SCREEN FRAME: Invalid streaming permission: %v", err)
		return
	}

	// Obtener el administrador que está controlando esta sesión
	adminUserID, err := h.sessionService.GetAdminUserIDForActiveSession(screenFrame.SessionID)
	if err != nil {
		log.Printf("❌ SCREEN FRAME: Error getting admin for session: %v", err)
		return
	}

	// Reenviar frame al administrador a través del AdminWebSocketHandler
	if h.adminWSHandler != nil {
		err := h.adminWSHandler.ForwardScreenFrameToAdmin(adminUserID, screenFrame)
		if err != nil {
			log.Printf("❌ SCREEN FRAME: Error forwarding frame to admin %s: %v", adminUserID, err)
		} else {
			log.Printf("✅ SCREEN FRAME: Frame %d forwarded to admin %s", screenFrame.SequenceNum, adminUserID)
		}
	} else {
		log.Printf("⚠️ SCREEN FRAME: No admin WebSocket handler available")
	}
}

// handleSessionAccepted maneja cuando el cliente acepta una sesión de control remoto
func (h *WebSocketHandler) handleSessionAccepted(conn *websocket.Conn, clientConn *ClientConnection, data interface{}) {
	// Parse session accepted message
	sessionData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshalling session accepted data: %v", err)
		return
	}

	var acceptedMsg struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(sessionData, &acceptedMsg); err != nil {
		log.Printf("Error unmarshalling session accepted message: %v", err)
		return
	}

	log.Printf("🎉 Client %s accepted remote control session: %s", clientConn.PCID, acceptedMsg.SessionID)

	// Actualizar estado de sesión en base de datos a ACTIVE
	err = h.sessionService.AcceptSession(acceptedMsg.SessionID)
	if err != nil {
		log.Printf("❌ Error accepting session in service: %v", err)

		// Enviar error al cliente
		errorMsg := dto.WebSocketMessage{
			Type: "session_failed",
			Data: map[string]interface{}{
				"session_id": acceptedMsg.SessionID,
				"error":      "Failed to activate session",
				"message":    err.Error(),
			},
		}
		conn.WriteJSON(errorMsg)
		return
	}

	log.Printf("✅ Session %s successfully activated in database", acceptedMsg.SessionID)

	// Notificar al administrador que la sesión fue aceptada
	if h.adminWSHandler != nil {
		err = h.adminWSHandler.NotifySessionAccepted(acceptedMsg.SessionID)
		if err != nil {
			log.Printf("⚠️ Warning: Failed to notify admin of session acceptance: %v", err)
		} else {
			log.Printf("✅ Admin notified of session %s acceptance", acceptedMsg.SessionID)
		}
	}

	// Enviar confirmación de sesión iniciada al cliente
	sessionStartedMsg := dto.WebSocketMessage{
		Type: "session_started",
		Data: map[string]interface{}{
			"session_id": acceptedMsg.SessionID,
			"status":     "ACTIVE",
			"message":    "Remote control session started successfully",
			"timestamp":  time.Now().Unix(),
		},
	}

	err = conn.WriteJSON(sessionStartedMsg)
	if err != nil {
		log.Printf("Error sending session started message to client: %v", err)
	} else {
		log.Printf("✅ Session started confirmation sent to client %s", clientConn.PCID)
	}
}

// handleSessionRejected maneja cuando el cliente rechaza una sesión de control remoto
func (h *WebSocketHandler) handleSessionRejected(conn *websocket.Conn, clientConn *ClientConnection, data interface{}) {
	// Parse session rejected message
	sessionData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshalling session rejected data: %v", err)
		return
	}

	var rejectedMsg struct {
		SessionID string `json:"session_id"`
		Reason    string `json:"reason"`
	}
	if err := json.Unmarshal(sessionData, &rejectedMsg); err != nil {
		log.Printf("Error unmarshalling session rejected message: %v", err)
		return
	}

	log.Printf("❌ Client %s rejected remote control session: %s (reason: %s)",
		clientConn.PCID, rejectedMsg.SessionID, rejectedMsg.Reason)

	// TODO: Actualizar estado de sesión en base de datos a ENDED_BY_CLIENT
	// TODO: Notificar al administrador que la sesión fue rechazada
}

// Helper methods for sending responses

func (h *WebSocketHandler) sendAuthResponse(conn *websocket.Conn, success bool, token, userID, errorMsg string) {
	response := dto.WebSocketMessage{
		Type: dto.MessageTypeClientAuthResp,
		Data: dto.ClientAuthResponse{
			Success: success,
			Token:   token,
			UserID:  userID,
			Error:   errorMsg,
		},
	}
	conn.WriteJSON(response)
}

func (h *WebSocketHandler) sendPCRegistrationResponse(conn *websocket.Conn, success bool, pcID, errorMsg string) {
	response := dto.WebSocketMessage{
		Type: dto.MessageTypePCRegistrationResp,
		Data: dto.PCRegistrationResponse{
			Success: success,
			PCID:    pcID,
			Error:   errorMsg,
		},
	}
	conn.WriteJSON(response)
}

// GetConnectedPCs returns a list of currently connected PCs
func (h *WebSocketHandler) GetConnectedPCs() map[string]*ClientConnection {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	result := make(map[string]*ClientConnection)
	for pcID, conn := range h.pcConnections {
		result[pcID] = conn
	}
	return result
}

// Utility functions

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func generateConnectionID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

// SendRemoteControlRequestToClient envía una solicitud de control remoto a un cliente específico
func (h *WebSocketHandler) SendRemoteControlRequestToClient(sessionID, clientPCID, adminUserID, adminUsername string) error {
	log.Printf("🚀 REMOTE CONTROL: Attempting to send request to client PC: %s", clientPCID)

	h.mutex.RLock()
	clientConn, exists := h.pcConnections[clientPCID]
	h.mutex.RUnlock()

	if !exists {
		log.Printf("❌ REMOTE CONTROL: Client PC %s not found in connections map", clientPCID)
		log.Printf("📊 REMOTE CONTROL: Current connected PCs: %d", len(h.pcConnections))
		for pcID := range h.pcConnections {
			log.Printf("  - Connected PC: %s", pcID)
		}
		return fmt.Errorf("client PC %s not connected", clientPCID)
	}

	log.Printf("✅ REMOTE CONTROL: Found client connection for PC: %s", clientPCID)

	// Crear mensaje de solicitud de control remoto
	remoteControlMsg := dto.WebSocketMessage{
		Type: "remote_control_request",
		Data: map[string]interface{}{
			"session_id":     sessionID,
			"admin_user_id":  adminUserID,
			"admin_username": adminUsername,
			"client_pc_id":   clientPCID,
			"timestamp":      time.Now().Unix(),
		},
	}

	log.Printf("📡 REMOTE CONTROL: Sending message to client %s: %+v", clientPCID, remoteControlMsg)

	// Enviar mensaje al cliente
	err := clientConn.Conn.WriteJSON(remoteControlMsg)
	if err != nil {
		log.Printf("❌ REMOTE CONTROL: Error sending to client %s: %v", clientPCID, err)
		return err
	}

	log.Printf("✅ REMOTE CONTROL: Request sent successfully to client %s (session: %s)", clientPCID, sessionID)
	return nil
}

// SendInputCommandToClient envía un comando de input a un cliente específico
func (h *WebSocketHandler) SendInputCommandToClient(clientPCID string, inputCommand dto.InputCommand) error {
	log.Printf("🖱️ REMOTE CONTROL: Attempting to send input command to client PC: %s", clientPCID)

	h.mutex.RLock()
	clientConn, exists := h.pcConnections[clientPCID]
	h.mutex.RUnlock()

	if !exists {
		log.Printf("❌ INPUT COMMAND: Client PC %s not found in connections map", clientPCID)
		return fmt.Errorf("client PC %s not connected", clientPCID)
	}

	log.Printf("✅ INPUT COMMAND: Found client connection for PC: %s", clientPCID)

	// Crear mensaje de comando de input
	inputMsg := dto.WebSocketMessage{
		Type: "input_command",
		Data: inputCommand,
	}

	log.Printf("📡 INPUT COMMAND: Sending to client %s: %+v", clientPCID, inputMsg)

	// Enviar mensaje al cliente
	err := clientConn.Conn.WriteJSON(inputMsg)
	if err != nil {
		log.Printf("❌ INPUT COMMAND: Error sending to client %s: %v", clientPCID, err)
		return err
	}

	log.Printf("✅ INPUT COMMAND: Sent successfully to client %s", clientPCID)
	return nil
}

// SendSessionEndedToClient notifica al cliente que una sesión ha terminado
func (h *WebSocketHandler) SendSessionEndedToClient(sessionID, clientPCID string) error {
	log.Printf("🔚 SESSION END: Attempting to send session ended notification to client PC: %s", clientPCID)

	h.mutex.RLock()
	clientConn, exists := h.pcConnections[clientPCID]
	h.mutex.RUnlock()

	if !exists {
		log.Printf("⚠️ SESSION END: Client PC %s not found in connections map", clientPCID)
		return nil // No es un error crítico si el cliente no está conectado
	}

	log.Printf("✅ SESSION END: Found client connection for PC: %s", clientPCID)

	// Crear mensaje de sesión terminada
	sessionEndedMsg := dto.WebSocketMessage{
		Type: "control_session_ended",
		Data: map[string]interface{}{
			"session_id": sessionID,
			"reason":     "ended_by_admin",
			"message":    "Remote control session ended by administrator",
			"timestamp":  time.Now().Unix(),
		},
	}

	log.Printf("📡 SESSION END: Sending to client %s: %+v", clientPCID, sessionEndedMsg)

	// Enviar mensaje al cliente
	err := clientConn.Conn.WriteJSON(sessionEndedMsg)
	if err != nil {
		log.Printf("❌ SESSION END: Error sending to client %s: %v", clientPCID, err)
		return err
	}

	log.Printf("✅ SESSION END: Notification sent successfully to client %s", clientPCID)
	return nil
}
