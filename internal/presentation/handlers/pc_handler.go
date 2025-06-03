package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/unikyri/escritorio-remoto-backend/internal/application/pcservice"
	"github.com/unikyri/escritorio-remoto-backend/internal/application/userservice"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/user"
	"github.com/unikyri/escritorio-remoto-backend/internal/presentation/dto"
)

// PCHandler manages PC-related endpoints for administrators
type PCHandler struct {
	pcService   pcservice.IPCService
	authService *userservice.AuthService
}

// NewPCHandler creates a new PC handler
func NewPCHandler(pcService pcservice.IPCService, authService *userservice.AuthService) *PCHandler {
	return &PCHandler{
		pcService:   pcService,
		authService: authService,
	}
}

// GetAllClientPCs handles GET /api/admin/pcs - retrieves all client PCs
func (h *PCHandler) GetAllClientPCs(c *gin.Context) {
	// Verificar autenticación y autorización de administrador
	userInfo, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponseDTO{
			Error:   "AUTHENTICATION_REQUIRED",
			Message: "Authentication required",
			Code:    http.StatusUnauthorized,
		})
		return
	}

	userClaims, ok := userInfo.(*userservice.JWTClaims)
	if !ok || userClaims.Role != string(user.RoleAdministrator) {
		c.JSON(http.StatusForbidden, dto.ErrorResponseDTO{
			Error:   "ADMIN_PRIVILEGES_REQUIRED",
			Message: "Admin privileges required",
			Code:    http.StatusForbidden,
		})
		return
	}

	// Obtener todos los PCs cliente
	pcs, err := h.pcService.GetAllClientPCs(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponseDTO{
			Error:   "RETRIEVAL_FAILED",
			Message: "Failed to retrieve client PCs",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	// Convertir a DTOs
	pcDTOs := make([]dto.ClientPCDTO, len(pcs))
	for i, pc := range pcs {
		pcDTOs[i] = dto.ClientPCDTO{
			PCID:             pc.PCID,
			Identifier:       pc.Identifier,
			ConnectionStatus: string(pc.ConnectionStatus),
			OwnerUsername:    pc.OwnerUserID, // Nota: En esta fase usamos UserID, en fase posterior incluiremos lookup de username
			IP:               pc.IP,
			RegisteredAt:     pc.RegisteredAt,
			LastSeenAt:       pc.LastSeenAt,
			UpdatedAt:        pc.UpdatedAt,
		}
	}

	// Responder con la lista de PCs
	c.JSON(http.StatusOK, dto.ClientPCListResponse{
		Success: true,
		Data:    pcDTOs,
		Count:   len(pcDTOs),
		Message: "Client PCs retrieved successfully",
	})
}

// GetOnlineClientPCs handles GET /api/admin/pcs/online - retrieves only online client PCs
func (h *PCHandler) GetOnlineClientPCs(c *gin.Context) {
	// Verificar autenticación y autorización de administrador
	userInfo, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponseDTO{
			Error:   "AUTHENTICATION_REQUIRED",
			Message: "Authentication required",
			Code:    http.StatusUnauthorized,
		})
		return
	}

	userClaims, ok := userInfo.(*userservice.JWTClaims)
	if !ok || userClaims.Role != string(user.RoleAdministrator) {
		c.JSON(http.StatusForbidden, dto.ErrorResponseDTO{
			Error:   "ADMIN_PRIVILEGES_REQUIRED",
			Message: "Admin privileges required",
			Code:    http.StatusForbidden,
		})
		return
	}

	// Obtener solo los PCs cliente online
	pcs, err := h.pcService.GetOnlineClientPCs(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponseDTO{
			Error:   "RETRIEVAL_FAILED",
			Message: "Failed to retrieve online client PCs",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	// Convertir a DTOs
	pcDTOs := make([]dto.ClientPCDTO, len(pcs))
	for i, pc := range pcs {
		pcDTOs[i] = dto.ClientPCDTO{
			PCID:             pc.PCID,
			Identifier:       pc.Identifier,
			ConnectionStatus: string(pc.ConnectionStatus),
			OwnerUsername:    pc.OwnerUserID, // Nota: En esta fase usamos UserID, en fase posterior incluiremos lookup de username
			IP:               pc.IP,
			RegisteredAt:     pc.RegisteredAt,
			LastSeenAt:       pc.LastSeenAt,
			UpdatedAt:        pc.UpdatedAt,
		}
	}

	// Responder con la lista de PCs online
	c.JSON(http.StatusOK, dto.OnlineClientPCListResponse{
		Success: true,
		Data:    pcDTOs,
		Count:   len(pcDTOs),
		Message: "Online client PCs retrieved successfully",
	})
}
