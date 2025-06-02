package websocket

import (
	"context"
	"encoding/json"
	"log"

	"github.com/unikyri/escritorio-remoto-backend/internal/application/remotesessionservice"
)

// SessionHandler maneja los mensajes WebSocket relacionados con sesiones
type SessionHandler struct {
	sessionService *remotesessionservice.RemoteSessionService
	hub            *Hub
}

// NewSessionHandler crea una nueva instancia del handler de sesiones
func NewSessionHandler(sessionService *remotesessionservice.RemoteSessionService, hub *Hub) *SessionHandler {
	return &SessionHandler{
		sessionService: sessionService,
		hub:            hub,
	}
}

// HandleMessage procesa mensajes WebSocket relacionados con sesiones
func (sh *SessionHandler) HandleMessage(clientID string, messageType string, data []byte) error {
	switch messageType {
	case MessageTypeSessionAccepted:
		return sh.handleSessionAccepted(clientID, data)
	case MessageTypeSessionRejected:
		return sh.handleSessionRejected(clientID, data)
	case MessageTypeHeartbeat:
		return sh.handleHeartbeat(clientID, data)
	default:
		log.Printf("Unknown message type from client %s: %s", clientID, messageType)
		return nil
	}
}

// handleSessionAccepted maneja cuando un cliente acepta una sesión
func (sh *SessionHandler) handleSessionAccepted(clientID string, data []byte) error {
	var message SessionAcceptedMessage
	if err := json.Unmarshal(data, &message); err != nil {
		log.Printf("Error unmarshaling SessionAccepted message: %v", err)
		return err
	}

	// Aceptar la sesión usando el servicio
	err := sh.sessionService.AcceptSession(message.SessionID)
	if err != nil {
		log.Printf("Error accepting session %s: %v", message.SessionID, err)

		// Enviar mensaje de error al cliente
		errorMsg := NewSessionFailedMessage(message.SessionID, err.Error())
		sh.hub.SendToClient(context.Background(), clientID, errorMsg)
		return err
	}

	// Obtener la sesión actualizada
	session, err := sh.sessionService.GetSessionById(message.SessionID)
	if err != nil {
		log.Printf("Error getting session %s: %v", message.SessionID, err)
		return err
	}

	// Notificar al administrador que la sesión ha iniciado
	sessionStartedMsg := NewSessionStartedMessage(
		session.SessionID(),
		session.AdminUserID(),
		session.ClientPCID(),
		*session.StartTime(),
	)

	// Enviar notificación al administrador (necesitaríamos un mapa de admin -> conexión)
	// Por ahora, broadcast a todos los administradores conectados
	sh.hub.BroadcastToAll(sessionStartedMsg)

	log.Printf("Session %s accepted and started", message.SessionID)
	return nil
}

// handleSessionRejected maneja cuando un cliente rechaza una sesión
func (sh *SessionHandler) handleSessionRejected(clientID string, data []byte) error {
	var message SessionRejectedMessage
	if err := json.Unmarshal(data, &message); err != nil {
		log.Printf("Error unmarshaling SessionRejected message: %v", err)
		return err
	}

	// Rechazar la sesión usando el servicio
	err := sh.sessionService.RejectSession(message.SessionID, message.Reason)
	if err != nil {
		log.Printf("Error rejecting session %s: %v", message.SessionID, err)
		return err
	}

	// Obtener la sesión actualizada
	session, err := sh.sessionService.GetSessionById(message.SessionID)
	if err != nil {
		log.Printf("Error getting session %s: %v", message.SessionID, err)
		return err
	}

	// Notificar al administrador que la sesión fue rechazada
	sessionRejectedMsg := NewSessionRejectedMessage(session.SessionID(), message.Reason)

	// Broadcast a todos los administradores conectados
	sh.hub.BroadcastToAll(sessionRejectedMsg)

	log.Printf("Session %s rejected by client %s", message.SessionID, clientID)
	return nil
}

// handleHeartbeat maneja mensajes de heartbeat
func (sh *SessionHandler) handleHeartbeat(clientID string, data []byte) error {
	var message HeartbeatMessage
	if err := json.Unmarshal(data, &message); err != nil {
		log.Printf("Error unmarshaling Heartbeat message: %v", err)
		return err
	}

	// Aquí podríamos actualizar el last_seen del PC en la base de datos
	log.Printf("Heartbeat received from client %s", clientID)
	return nil
}

// SendRemoteControlRequest envía una solicitud de control remoto a un cliente
func (sh *SessionHandler) SendRemoteControlRequest(sessionID, adminUserID, clientPCID, adminUsername string) error {
	message := NewRemoteControlRequestMessage(sessionID, adminUserID, clientPCID, adminUsername)
	return sh.hub.SendToClient(context.Background(), clientPCID, message)
}
