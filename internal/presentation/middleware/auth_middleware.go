package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/unikyri/escritorio-remoto-backend/internal/application/userservice"
	"github.com/unikyri/escritorio-remoto-backend/internal/presentation/dto"
)

// AuthMiddleware creates a JWT authentication middleware
func AuthMiddleware(authService *userservice.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Obtener el token del header Authorization
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, dto.ErrorResponseDTO{
				Error:   "MISSING_TOKEN",
				Message: "Authorization token required",
				Code:    http.StatusUnauthorized,
			})
			c.Abort()
			return
		}

		// Verificar formato Bearer token
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, dto.ErrorResponseDTO{
				Error:   "INVALID_TOKEN_FORMAT",
				Message: "Invalid authorization token format",
				Code:    http.StatusUnauthorized,
			})
			c.Abort()
			return
		}

		tokenString := tokenParts[1]

		// Validar el token
		claims, err := authService.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, dto.ErrorResponseDTO{
				Error:   "INVALID_TOKEN",
				Message: "Invalid or expired token",
				Code:    http.StatusUnauthorized,
			})
			c.Abort()
			return
		}

		// Agregar los claims del usuario al contexto
		c.Set("user", claims)
		c.Next()
	}
}
