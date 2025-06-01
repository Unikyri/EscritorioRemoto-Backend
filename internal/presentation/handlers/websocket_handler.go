package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/unikyri/escritorio-remoto-backend/internal/application/pcservice"
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
	authService   *userservice.AuthService
	pcService     pcservice.IPCService
	connections   map[string]*ClientConnection // map[connectionID]*ClientConnection
	pcConnections map[string]*ClientConnection // map[pcID]*ClientConnection
	mutex         sync.RWMutex
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(authService *userservice.AuthService, pcService pcservice.IPCService) *WebSocketHandler {
	return &WebSocketHandler{
		authService:   authService,
		pcService:     pcService,
		connections:   make(map[string]*ClientConnection),
		pcConnections: make(map[string]*ClientConnection),
		mutex:         sync.RWMutex{},
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
			// Mark PC as offline
			if clientConn.IsAuth {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				h.pcService.UpdatePCConnectionStatus(ctx, clientConn.PCID, clientpc.PCConnectionStatusOffline)
			}
		}
		h.mutex.Unlock()
		log.Printf("Client disconnected: %s", connectionID)
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

	// Update last seen if PC is registered
	if clientConn.PCID != "" && clientConn.IsAuth {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		h.pcService.UpdatePCLastSeen(ctx, clientConn.PCID)
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
