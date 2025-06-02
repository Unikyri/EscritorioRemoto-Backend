package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/unikyri/escritorio-remoto-backend/internal/application/remotesessionservice"
	"github.com/unikyri/escritorio-remoto-backend/internal/infrastructure/comms/websocket"
	"github.com/unikyri/escritorio-remoto-backend/internal/presentation/http/dto"
	"github.com/unikyri/escritorio-remoto-backend/internal/presentation/middleware"
)

// RemoteControlHandler maneja las operaciones de control remoto
type RemoteControlHandler struct {
	sessionService  *remotesessionservice.RemoteSessionService
	websocketHub    *websocket.Hub
}

// NewRemoteControlHandler crea una nueva instancia del handler
func NewRemoteControlHandler(
	sessionService *remotesessionservice.RemoteSessionService,
	websocketHub *websocket.Hub,
) *RemoteControlHandler {
	return &RemoteControlHandler{
		sessionService: sessionService,
		websocketHub:   websocketHub,
	}
}

// InitiateSession maneja POST /api/admin/sessions/initiate
func (rch *RemoteControlHandler) InitiateSession(c *gin.Context) {
	// Obtener ID del usuario desde JWT (middleware de autenticación)
	adminUserID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	// Parsear request body
	var req dto.InitiateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	// Validar request
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		})
		return
	}

	// Iniciar sesión usando el servicio
	session, err := rch.sessionService.InitiateSession(
		adminUserID.(string),
		req.ClientPCID,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "session_initiation_failed",
			Message: err.Error(),
		})
		return
	}

	// Enviar notificación WebSocket al cliente objetivo
	err = rch.sendRemoteControlRequestToClient(session.SessionID(), session.ClientPCID(), adminUserID.(string))
	if err != nil {
		// Log error pero no fallar la request - la sesión ya se creó
		// En una implementación real usaríamos logging estructurado
		// log.Error("Failed to send WebSocket notification", "error", err)
	}

	// Responder con la sesión creada
	response := dto.InitiateSessionResponse{
		Success:   true,
		SessionID: session.SessionID(),
		Status:    string(session.Status()),
		Message:   "Remote control request sent to client",
	}

	c.JSON(http.StatusOK, response)
}

// GetSessionStatus maneja GET /api/admin/sessions/:sessionId/status
func (rch *RemoteControlHandler) GetSessionStatus(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_session_id",
			Message: "Session ID is required",
		})
		return
	}

	// Obtener sesión
	session, err := rch.sessionService.GetSessionById(sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "session_retrieval_failed",
			Message: err.Error(),
		})
		return
	}

	if session == nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "session_not_found",
			Message: "Session not found",
		})
		return
	}

	// Crear response DTO
	response := dto.SessionStatusResponse{
		SessionID:    session.SessionID(),
		AdminUserID:  session.AdminUserID(),
		ClientPCID:   session.ClientPCID(),
		Status:       string(session.Status()),
		StartTime:    session.StartTime(),
		EndTime:      session.EndTime(),
		CreatedAt:    session.CreatedAt(),
		UpdatedAt:    session.UpdatedAt(),
	}

	if session.GetDuration() > 0 {
		duration := session.GetDuration()
		response.Duration = &duration
	}

	c.JSON(http.StatusOK, response)
}

// GetActiveSessions maneja GET /api/admin/sessions/active
func (rch *RemoteControlHandler) GetActiveSessions(c *gin.Context) {
	// Obtener sesiones activas
	sessions, err := rch.sessionService.GetActiveSessions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "sessions_retrieval_failed",
			Message: err.Error(),
		})
		return
	}

	// Convertir a DTOs
	var sessionDTOs []dto.SessionSummaryDTO
	for _, session := range sessions {
		sessionDTOs = append(sessionDTOs, dto.SessionSummaryDTO{
			SessionID:   session.SessionID(),
			AdminUserID: session.AdminUserID(),
			ClientPCID:  session.ClientPCID(),
			Status:      string(session.Status()),
			StartTime:   session.StartTime(),
			CreatedAt:   session.CreatedAt(),
		})
	}

	response := dto.ActiveSessionsResponse{
		Sessions: sessionDTOs,
		Count:    len(sessionDTOs),
	}

	c.JSON(http.StatusOK, response)
}

// GetUserSessions maneja GET /api/admin/sessions/my
func (rch *RemoteControlHandler) GetUserSessions(c *gin.Context) {
	// Obtener ID del usuario desde JWT
	adminUserID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	// Obtener sesiones del usuario
	sessions, err := rch.sessionService.GetSessionsByUser(adminUserID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "sessions_retrieval_failed",
			Message: err.Error(),
		})
		return
	}

	// Convertir a DTOs
	var sessionDTOs []dto.SessionSummaryDTO
	for _, session := range sessions {
		sessionDTOs = append(sessionDTOs, dto.SessionSummaryDTO{
			SessionID:   session.SessionID(),
			AdminUserID: session.AdminUserID(),
			ClientPCID:  session.ClientPCID(),
			Status:      string(session.Status()),
			StartTime:   session.StartTime(),
			EndTime:     session.EndTime(),
			CreatedAt:   session.CreatedAt(),
		})
	}

	response := dto.UserSessionsResponse{
		Sessions: sessionDTOs,
		Count:    len(sessionDTOs),
	}

	c.JSON(http.StatusOK, response)
}

// sendRemoteControlRequestToClient envía una solicitud de control remoto al cliente vía WebSocket
func (rch *RemoteControlHandler) sendRemoteControlRequestToClient(sessionID, clientPCID, adminUserID string) error {
	// Crear mensaje WebSocket usando el factory method
	message := websocket.NewRemoteControlRequestMessage(sessionID, adminUserID, clientPCID, "")

	// Enviar a través del hub WebSocket
	return rch.websocketHub.SendToClient(context.Background(), clientPCID, message)
} 