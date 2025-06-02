package remotesession

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// SessionStatus define los estados posibles de una sesión remota
type SessionStatus string

const (
	StatusPendingApproval SessionStatus = "PENDING_APPROVAL"
	StatusActive          SessionStatus = "ACTIVE"
	StatusEnded           SessionStatus = "ENDED_SUCCESSFULLY"
	StatusEndedByAdmin    SessionStatus = "ENDED_BY_ADMIN"
	StatusEndedByClient   SessionStatus = "ENDED_BY_CLIENT"
	StatusRejected        SessionStatus = "REJECTED"
	StatusFailed          SessionStatus = "FAILED"
)

// RemoteSession representa la entidad de sesión de control remoto
type RemoteSession struct {
	sessionID    string
	adminUserID  string
	clientPCID   string
	startTime    *time.Time
	endTime      *time.Time
	status       SessionStatus
	sessionVideoID *string
	createdAt    time.Time
	updatedAt    time.Time
}

// NewRemoteSession crea una nueva sesión remota
func NewRemoteSession(adminUserID, clientPCID string) (*RemoteSession, error) {
	if adminUserID == "" {
		return nil, errors.New("admin user ID cannot be empty")
	}
	if clientPCID == "" {
		return nil, errors.New("client PC ID cannot be empty")
	}

	sessionID := uuid.New().String()
	now := time.Now().UTC()

	return &RemoteSession{
		sessionID:    sessionID,
		adminUserID:  adminUserID,
		clientPCID:   clientPCID,
		startTime:    nil,
		endTime:      nil,
		status:       StatusPendingApproval,
		sessionVideoID: nil,
		createdAt:    now,
		updatedAt:    now,
	}, nil
}

// NewRemoteSessionFromDB crea una sesión remota desde datos de base de datos
func NewRemoteSessionFromDB(
	sessionID, adminUserID, clientPCID string,
	startTime, endTime *time.Time,
	status SessionStatus,
	sessionVideoID *string,
	createdAt, updatedAt time.Time,
) *RemoteSession {
	return &RemoteSession{
		sessionID:      sessionID,
		adminUserID:    adminUserID,
		clientPCID:     clientPCID,
		startTime:      startTime,
		endTime:        endTime,
		status:         status,
		sessionVideoID: sessionVideoID,
		createdAt:      createdAt,
		updatedAt:      updatedAt,
	}
}

// Getters
func (rs *RemoteSession) SessionID() string {
	return rs.sessionID
}

func (rs *RemoteSession) AdminUserID() string {
	return rs.adminUserID
}

func (rs *RemoteSession) ClientPCID() string {
	return rs.clientPCID
}

func (rs *RemoteSession) StartTime() *time.Time {
	return rs.startTime
}

func (rs *RemoteSession) EndTime() *time.Time {
	return rs.endTime
}

func (rs *RemoteSession) Status() SessionStatus {
	return rs.status
}

func (rs *RemoteSession) SessionVideoID() *string {
	return rs.sessionVideoID
}

func (rs *RemoteSession) CreatedAt() time.Time {
	return rs.createdAt
}

func (rs *RemoteSession) UpdatedAt() time.Time {
	return rs.updatedAt
}

// Accept acepta la sesión y la marca como activa
func (rs *RemoteSession) Accept() error {
	if !rs.CanAccept() {
		return errors.New("session cannot be accepted in current state")
	}

	now := time.Now().UTC()
	rs.status = StatusActive
	rs.startTime = &now
	rs.updatedAt = now

	return nil
}

// Reject rechaza la sesión
func (rs *RemoteSession) Reject() error {
	if !rs.CanReject() {
		return errors.New("session cannot be rejected in current state")
	}

	rs.status = StatusRejected
	rs.updatedAt = time.Now().UTC()

	return nil
}

// End finaliza la sesión con el estado especificado
func (rs *RemoteSession) End(endStatus SessionStatus) error {
	if !rs.CanEnd() {
		return errors.New("session cannot be ended in current state")
	}

	if !isValidEndStatus(endStatus) {
		return errors.New("invalid end status")
	}

	now := time.Now().UTC()
	rs.status = endStatus
	rs.endTime = &now
	rs.updatedAt = now

	return nil
}

// SetSessionVideoID establece el ID del video de la sesión
func (rs *RemoteSession) SetSessionVideoID(videoID string) error {
	if videoID == "" {
		return errors.New("video ID cannot be empty")
	}

	rs.sessionVideoID = &videoID
	rs.updatedAt = time.Now().UTC()

	return nil
}

// Métodos de validación de estado
func (rs *RemoteSession) CanAccept() bool {
	return rs.status == StatusPendingApproval
}

func (rs *RemoteSession) CanReject() bool {
	return rs.status == StatusPendingApproval
}

func (rs *RemoteSession) CanEnd() bool {
	return rs.status == StatusActive
}

func (rs *RemoteSession) IsActive() bool {
	return rs.status == StatusActive
}

func (rs *RemoteSession) IsPending() bool {
	return rs.status == StatusPendingApproval
}

func (rs *RemoteSession) IsCompleted() bool {
	return rs.status == StatusEnded || 
		   rs.status == StatusEndedByAdmin || 
		   rs.status == StatusEndedByClient ||
		   rs.status == StatusRejected ||
		   rs.status == StatusFailed
}

// GetDuration retorna la duración de la sesión si está disponible
func (rs *RemoteSession) GetDuration() time.Duration {
	if rs.startTime == nil || rs.endTime == nil {
		return 0
	}
	return rs.endTime.Sub(*rs.startTime)
}

// UpdateStatus actualiza solo el estado (para casos especiales)
func (rs *RemoteSession) UpdateStatus(newStatus SessionStatus) error {
	if !isValidStatus(newStatus) {
		return errors.New("invalid session status")
	}

	if !rs.canTransitionTo(newStatus) {
		return errors.New("invalid status transition")
	}

	rs.status = newStatus
	rs.updatedAt = time.Now().UTC()

	return nil
}

// Validaciones privadas
func isValidStatus(status SessionStatus) bool {
	switch status {
	case StatusPendingApproval, StatusActive, StatusEnded, 
		 StatusEndedByAdmin, StatusEndedByClient, StatusRejected, StatusFailed:
		return true
	default:
		return false
	}
}

func isValidEndStatus(status SessionStatus) bool {
	switch status {
	case StatusEnded, StatusEndedByAdmin, StatusEndedByClient, StatusFailed:
		return true
	default:
		return false
	}
}

func (rs *RemoteSession) canTransitionTo(newStatus SessionStatus) bool {
	// Definir transiciones válidas según reglas de negocio
	validTransitions := map[SessionStatus][]SessionStatus{
		StatusPendingApproval: {StatusActive, StatusRejected, StatusFailed},
		StatusActive:          {StatusEnded, StatusEndedByAdmin, StatusEndedByClient, StatusFailed},
		StatusEnded:           {}, // No puede cambiar
		StatusEndedByAdmin:    {}, // No puede cambiar
		StatusEndedByClient:   {}, // No puede cambiar
		StatusRejected:        {}, // No puede cambiar
		StatusFailed:          {}, // No puede cambiar
	}

	allowedStates, exists := validTransitions[rs.status]
	if !exists {
		return false
	}

	for _, allowed := range allowedStates {
		if newStatus == allowed {
			return true
		}
	}

	return false
} 