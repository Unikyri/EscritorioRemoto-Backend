package dto

import (
	"errors"
	"time"
)

// InitiateSessionRequest representa la solicitud para iniciar una sesión remota
type InitiateSessionRequest struct {
	ClientPCID string `json:"client_pc_id" binding:"required"`
}

// Validate valida la solicitud de iniciación de sesión
func (req *InitiateSessionRequest) Validate() error {
	if req.ClientPCID == "" {
		return errors.New("client_pc_id is required")
	}
	return nil
}

// InitiateSessionResponse representa la respuesta de iniciación de sesión
type InitiateSessionResponse struct {
	Success   bool   `json:"success"`
	SessionID string `json:"session_id,omitempty"`
	Status    string `json:"status,omitempty"`
	Message   string `json:"message,omitempty"`
	Error     string `json:"error,omitempty"`
}

// SessionStatusResponse representa la respuesta de estado de sesión
type SessionStatusResponse struct {
	SessionID    string         `json:"session_id"`
	AdminUserID  string         `json:"admin_user_id"`
	ClientPCID   string         `json:"client_pc_id"`
	Status       string         `json:"status"`
	StartTime    *time.Time     `json:"start_time,omitempty"`
	EndTime      *time.Time     `json:"end_time,omitempty"`
	Duration     *time.Duration `json:"duration,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// SessionSummaryDTO representa un resumen de sesión
type SessionSummaryDTO struct {
	SessionID   string     `json:"session_id"`
	AdminUserID string     `json:"admin_user_id"`
	ClientPCID  string     `json:"client_pc_id"`
	Status      string     `json:"status"`
	StartTime   *time.Time `json:"start_time,omitempty"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// ActiveSessionsResponse representa la respuesta de sesiones activas
type ActiveSessionsResponse struct {
	Sessions []SessionSummaryDTO `json:"sessions"`
	Count    int                 `json:"count"`
}

// UserSessionsResponse representa la respuesta de sesiones de usuario
type UserSessionsResponse struct {
	Sessions []SessionSummaryDTO `json:"sessions"`
	Count    int                 `json:"count"`
}

// ErrorResponse representa una respuesta de error
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
} 