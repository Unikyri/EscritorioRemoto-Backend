package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/unikyri/escritorio-remoto-backend/internal/application/usecases/clientpc"
	"github.com/unikyri/escritorio-remoto-backend/internal/application/userservice"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/user"
	"github.com/unikyri/escritorio-remoto-backend/internal/presentation/dto"
)

// PCController maneja endpoints relacionados con PCs para administradores
type PCController struct {
	getAllPCsUseCase    clientpc.IGetAllPCsUseCase
	getOnlinePCsUseCase clientpc.IGetOnlinePCsUseCase
	authService         *userservice.AuthService
}

// NewPCController crea un nuevo controller de PC
func NewPCController(
	getAllPCsUseCase clientpc.IGetAllPCsUseCase,
	getOnlinePCsUseCase clientpc.IGetOnlinePCsUseCase,
	authService *userservice.AuthService,
) *PCController {
	return &PCController{
		getAllPCsUseCase:    getAllPCsUseCase,
		getOnlinePCsUseCase: getOnlinePCsUseCase,
		authService:         authService,
	}
}

// GetAllClientPCs handles GET /api/admin/pcs - retrieves all client PCs
func (ctrl *PCController) GetAllClientPCs(c *gin.Context) {
	// Verificar autenticación y autorización de administrador
	if !ctrl.isAdminAuthenticated(c) {
		return
	}

	// Ejecutar Use Case
	request := clientpc.GetAllPCsRequest{
		Limit:  50, // Valor por defecto
		Offset: 0,  // Valor por defecto
	}

	response, err := ctrl.getAllPCsUseCase.Execute(c.Request.Context(), request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponseDTO{
			Error:   "RETRIEVAL_FAILED",
			Message: "Failed to retrieve client PCs",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	// Convertir entidades a DTOs
	pcDTOs := make([]dto.ClientPCDTO, len(response.PCs))
	for i, pc := range response.PCs {
		pcDTOs[i] = dto.ClientPCDTO{
			PCID:             pc.ID().Value(),
			Identifier:       pc.Identifier(),
			ConnectionStatus: pc.ConnectionStatus().Value(),
			OwnerUsername:    pc.OwnerUserID(), // Nota: En esta fase usamos UserID, en fase posterior incluiremos lookup de username
			IP:               pc.IP(),
			RegisteredAt:     pc.RegisteredAt(),
			LastSeenAt:       pc.LastSeenAt(),
			UpdatedAt:        pc.UpdatedAt(),
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
func (ctrl *PCController) GetOnlineClientPCs(c *gin.Context) {
	// Verificar autenticación y autorización de administrador
	if !ctrl.isAdminAuthenticated(c) {
		return
	}

	// Ejecutar Use Case
	request := clientpc.GetOnlinePCsRequest{}

	response, err := ctrl.getOnlinePCsUseCase.Execute(c.Request.Context(), request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponseDTO{
			Error:   "RETRIEVAL_FAILED",
			Message: "Failed to retrieve online client PCs",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	// Convertir entidades a DTOs
	pcDTOs := make([]dto.ClientPCDTO, len(response.PCs))
	for i, pc := range response.PCs {
		pcDTOs[i] = dto.ClientPCDTO{
			PCID:             pc.ID().Value(),
			Identifier:       pc.Identifier(),
			ConnectionStatus: pc.ConnectionStatus().Value(),
			OwnerUsername:    pc.OwnerUserID(), // Nota: En esta fase usamos UserID, en fase posterior incluiremos lookup de username
			IP:               pc.IP(),
			RegisteredAt:     pc.RegisteredAt(),
			LastSeenAt:       pc.LastSeenAt(),
			UpdatedAt:        pc.UpdatedAt(),
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

// isAdminAuthenticated verifica autenticación y autorización de administrador
func (ctrl *PCController) isAdminAuthenticated(c *gin.Context) bool {
	userInfo, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponseDTO{
			Error:   "AUTHENTICATION_REQUIRED",
			Message: "Authentication required",
			Code:    http.StatusUnauthorized,
		})
		return false
	}

	userClaims, ok := userInfo.(*userservice.JWTClaims)
	if !ok || userClaims.Role != string(user.RoleAdministrator) {
		c.JSON(http.StatusForbidden, dto.ErrorResponseDTO{
			Error:   "ADMIN_PRIVILEGES_REQUIRED",
			Message: "Admin privileges required",
			Code:    http.StatusForbidden,
		})
		return false
	}

	return true
}
