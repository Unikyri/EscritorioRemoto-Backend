package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/unikyri/escritorio-remoto-backend/internal/application/userservice"
	"github.com/unikyri/escritorio-remoto-backend/internal/presentation/dto"
)

// AuthHandler maneja las peticiones de autenticación
type AuthHandler struct {
	authService *userservice.AuthService
}

// NewAuthHandler crea una nueva instancia del manejador de autenticación
func NewAuthHandler(authService *userservice.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Login maneja el endpoint POST /api/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var request dto.AuthRequestDTO

	// Validar la estructura JSON
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseDTO{
			Error:   "invalid_request",
			Message: "Invalid request format or missing required fields",
			Code:    400,
		})
		return
	}

	// Autenticar al administrador
	token, user, err := h.authService.AuthenticateAdmin(request.Username, request.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponseDTO{
			Error:   "authentication_failed",
			Message: "Invalid credentials or user is not an administrator",
			Code:    401,
		})
		return
	}

	// Convertir usuario a DTO
	userSnapshot := user.ToSnapshot()
	userDTO := dto.UserInfoDTO{
		UserID:    userSnapshot.UserID,
		Username:  userSnapshot.Username,
		Role:      userSnapshot.Role,
		IsActive:  userSnapshot.IsActive,
		CreatedAt: userSnapshot.CreatedAt,
		UpdatedAt: userSnapshot.UpdatedAt,
	}

	// Respuesta exitosa
	response := dto.AuthResponseDTO{
		Token: token,
		User:  userDTO,
	}

	c.JSON(http.StatusOK, response)
}

// RegisterRoutes registra las rutas del AuthHandler
func (h *AuthHandler) RegisterRoutes(router *gin.RouterGroup) {
	auth := router.Group("/auth")
	{
		auth.POST("/login", h.Login)
	}
}
