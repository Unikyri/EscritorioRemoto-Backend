package dto

import "time"

// AuthRequestDTO representa la petición de autenticación
type AuthRequestDTO struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// AuthResponseDTO representa la respuesta de autenticación exitosa
type AuthResponseDTO struct {
	Token string      `json:"token"`
	User  UserInfoDTO `json:"user"`
}

// UserInfoDTO representa la información del usuario
type UserInfoDTO struct {
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ErrorResponseDTO representa una respuesta de error
type ErrorResponseDTO struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}
