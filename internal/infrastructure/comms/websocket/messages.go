package websocket

import "time"

// Tipos de mensajes WebSocket
const (
	MessageTypeRemoteControlRequest = "remote_control_request"
	MessageTypeSessionAccepted      = "session_accepted"
	MessageTypeSessionRejected      = "session_rejected"
	MessageTypeSessionStarted       = "session_started"
	MessageTypeSessionEnded         = "session_ended"
	MessageTypeSessionFailed        = "session_failed"
	MessageTypeHeartbeat           = "heartbeat"
	MessageTypeError               = "error"
)

// BaseMessage estructura base para todos los mensajes WebSocket
type BaseMessage struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
}

// RemoteControlRequestMessage mensaje para solicitar control remoto
type RemoteControlRequestMessage struct {
	BaseMessage
	SessionID    string `json:"session_id"`
	AdminUserID  string `json:"admin_user_id"`
	ClientPCID   string `json:"client_pc_id"`
	AdminUsername string `json:"admin_username,omitempty"`
}

// SessionAcceptedMessage mensaje cuando el cliente acepta la sesión
type SessionAcceptedMessage struct {
	BaseMessage
	SessionID string `json:"session_id"`
}

// SessionRejectedMessage mensaje cuando el cliente rechaza la sesión
type SessionRejectedMessage struct {
	BaseMessage
	SessionID string `json:"session_id"`
	Reason    string `json:"reason,omitempty"`
}

// SessionStartedMessage mensaje cuando la sesión inicia exitosamente
type SessionStartedMessage struct {
	BaseMessage
	SessionID   string    `json:"session_id"`
	AdminUserID string    `json:"admin_user_id"`
	ClientPCID  string    `json:"client_pc_id"`
	StartTime   time.Time `json:"start_time"`
}

// SessionEndedMessage mensaje cuando la sesión termina
type SessionEndedMessage struct {
	BaseMessage
	SessionID string `json:"session_id"`
	EndReason string `json:"end_reason"`
	Duration  string `json:"duration,omitempty"`
}

// SessionFailedMessage mensaje cuando la sesión falla
type SessionFailedMessage struct {
	BaseMessage
	SessionID string `json:"session_id"`
	Error     string `json:"error"`
}

// HeartbeatMessage mensaje de heartbeat
type HeartbeatMessage struct {
	BaseMessage
	ClientID string `json:"client_id"`
}

// ErrorMessage mensaje de error genérico
type ErrorMessage struct {
	BaseMessage
	Error   string `json:"error"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// NewRemoteControlRequestMessage crea un mensaje de solicitud de control remoto
func NewRemoteControlRequestMessage(sessionID, adminUserID, clientPCID, adminUsername string) *RemoteControlRequestMessage {
	return &RemoteControlRequestMessage{
		BaseMessage: BaseMessage{
			Type:      MessageTypeRemoteControlRequest,
			Timestamp: time.Now().UTC(),
		},
		SessionID:     sessionID,
		AdminUserID:   adminUserID,
		ClientPCID:    clientPCID,
		AdminUsername: adminUsername,
	}
}

// NewSessionAcceptedMessage crea un mensaje de sesión aceptada
func NewSessionAcceptedMessage(sessionID string) *SessionAcceptedMessage {
	return &SessionAcceptedMessage{
		BaseMessage: BaseMessage{
			Type:      MessageTypeSessionAccepted,
			Timestamp: time.Now().UTC(),
		},
		SessionID: sessionID,
	}
}

// NewSessionRejectedMessage crea un mensaje de sesión rechazada
func NewSessionRejectedMessage(sessionID, reason string) *SessionRejectedMessage {
	return &SessionRejectedMessage{
		BaseMessage: BaseMessage{
			Type:      MessageTypeSessionRejected,
			Timestamp: time.Now().UTC(),
		},
		SessionID: sessionID,
		Reason:    reason,
	}
}

// NewSessionStartedMessage crea un mensaje de sesión iniciada
func NewSessionStartedMessage(sessionID, adminUserID, clientPCID string, startTime time.Time) *SessionStartedMessage {
	return &SessionStartedMessage{
		BaseMessage: BaseMessage{
			Type:      MessageTypeSessionStarted,
			Timestamp: time.Now().UTC(),
		},
		SessionID:   sessionID,
		AdminUserID: adminUserID,
		ClientPCID:  clientPCID,
		StartTime:   startTime,
	}
}

// NewSessionEndedMessage crea un mensaje de sesión terminada
func NewSessionEndedMessage(sessionID, endReason, duration string) *SessionEndedMessage {
	return &SessionEndedMessage{
		BaseMessage: BaseMessage{
			Type:      MessageTypeSessionEnded,
			Timestamp: time.Now().UTC(),
		},
		SessionID: sessionID,
		EndReason: endReason,
		Duration:  duration,
	}
}

// NewSessionFailedMessage crea un mensaje de sesión fallida
func NewSessionFailedMessage(sessionID, error string) *SessionFailedMessage {
	return &SessionFailedMessage{
		BaseMessage: BaseMessage{
			Type:      MessageTypeSessionFailed,
			Timestamp: time.Now().UTC(),
		},
		SessionID: sessionID,
		Error:     error,
	}
}

// NewHeartbeatMessage crea un mensaje de heartbeat
func NewHeartbeatMessage(clientID string) *HeartbeatMessage {
	return &HeartbeatMessage{
		BaseMessage: BaseMessage{
			Type:      MessageTypeHeartbeat,
			Timestamp: time.Now().UTC(),
		},
		ClientID: clientID,
	}
}

// NewErrorMessage crea un mensaje de error
func NewErrorMessage(error, message, details string) *ErrorMessage {
	return &ErrorMessage{
		BaseMessage: BaseMessage{
			Type:      MessageTypeError,
			Timestamp: time.Now().UTC(),
		},
		Error:   error,
		Message: message,
		Details: details,
	}
} 