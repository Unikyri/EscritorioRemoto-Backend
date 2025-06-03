package events

import (
	"time"

	"github.com/unikyri/escritorio-remoto-backend/internal/domain/shared/events"
)

// Constantes para tipos de eventos de sesión remota
const (
	RemoteSessionInitiated = "remote_session.initiated"
	RemoteSessionAccepted  = "remote_session.accepted"
	RemoteSessionRejected  = "remote_session.rejected"
	RemoteSessionEnded     = "remote_session.ended"
	RemoteSessionFailed    = "remote_session.failed"
)

// RemoteSessionInitiatedEventData datos para evento de sesión iniciada
type RemoteSessionInitiatedEventData struct {
	SessionID      string `json:"session_id"`
	AdminUserID    string `json:"admin_user_id"`
	ClientPCID     string `json:"client_pc_id"`
	AdminUsername  string `json:"admin_username"`
	ClientPCName   string `json:"client_pc_name"`
}

// RemoteSessionAcceptedEventData datos para evento de sesión aceptada
type RemoteSessionAcceptedEventData struct {
	SessionID   string    `json:"session_id"`
	AdminUserID string    `json:"admin_user_id"`
	ClientPCID  string    `json:"client_pc_id"`
	StartTime   time.Time `json:"start_time"`
}

// RemoteSessionRejectedEventData datos para evento de sesión rechazada
type RemoteSessionRejectedEventData struct {
	SessionID   string `json:"session_id"`
	AdminUserID string `json:"admin_user_id"`
	ClientPCID  string `json:"client_pc_id"`
}

// RemoteSessionEndedEventData datos para evento de sesión finalizada
type RemoteSessionEndedEventData struct {
	SessionID   string        `json:"session_id"`
	AdminUserID string        `json:"admin_user_id"`
	ClientPCID  string        `json:"client_pc_id"`
	EndTime     time.Time     `json:"end_time"`
	EndReason   string        `json:"end_reason"`
	Duration    time.Duration `json:"duration"`
}

// RemoteSessionFailedEventData datos para evento de sesión fallida
type RemoteSessionFailedEventData struct {
	SessionID   string `json:"session_id"`
	AdminUserID string `json:"admin_user_id"`
	ClientPCID  string `json:"client_pc_id"`
	ErrorMsg    string `json:"error_message"`
}

// NewRemoteSessionInitiatedEvent crea un evento de sesión iniciada
func NewRemoteSessionInitiatedEvent(sessionID, adminUserID, clientPCID, adminUsername, clientPCName string) events.DomainEvent {
	data := RemoteSessionInitiatedEventData{
		SessionID:     sessionID,
		AdminUserID:   adminUserID,
		ClientPCID:    clientPCID,
		AdminUsername: adminUsername,
		ClientPCName:  clientPCName,
	}

	return events.NewBaseDomainEvent(
		RemoteSessionInitiated,
		sessionID,
		data,
	)
}

// NewRemoteSessionAcceptedEvent crea un evento de sesión aceptada
func NewRemoteSessionAcceptedEvent(sessionID, adminUserID, clientPCID string, startTime time.Time) events.DomainEvent {
	data := RemoteSessionAcceptedEventData{
		SessionID:   sessionID,
		AdminUserID: adminUserID,
		ClientPCID:  clientPCID,
		StartTime:   startTime,
	}

	return events.NewBaseDomainEvent(
		RemoteSessionAccepted,
		sessionID,
		data,
	)
}

// NewRemoteSessionRejectedEvent crea un evento de sesión rechazada
func NewRemoteSessionRejectedEvent(sessionID, adminUserID, clientPCID string) events.DomainEvent {
	data := RemoteSessionRejectedEventData{
		SessionID:   sessionID,
		AdminUserID: adminUserID,
		ClientPCID:  clientPCID,
	}

	return events.NewBaseDomainEvent(
		RemoteSessionRejected,
		sessionID,
		data,
	)
}

// NewRemoteSessionEndedEvent crea un evento de sesión finalizada
func NewRemoteSessionEndedEvent(sessionID, adminUserID, clientPCID string, endTime time.Time, endReason string, duration time.Duration) events.DomainEvent {
	data := RemoteSessionEndedEventData{
		SessionID:   sessionID,
		AdminUserID: adminUserID,
		ClientPCID:  clientPCID,
		EndTime:     endTime,
		EndReason:   endReason,
		Duration:    duration,
	}

	return events.NewBaseDomainEvent(
		RemoteSessionEnded,
		sessionID,
		data,
	)
}

// NewRemoteSessionFailedEvent crea un evento de sesión fallida
func NewRemoteSessionFailedEvent(sessionID, adminUserID, clientPCID, errorMsg string) events.DomainEvent {
	data := RemoteSessionFailedEventData{
		SessionID:   sessionID,
		AdminUserID: adminUserID,
		ClientPCID:  clientPCID,
		ErrorMsg:    errorMsg,
	}

	return events.NewBaseDomainEvent(
		RemoteSessionFailed,
		sessionID,
		data,
	)
} 