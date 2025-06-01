package userservice

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/unikyri/escritorio-remoto-backend/internal/application/interfaces"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/user"
)

// AuthService maneja la autenticación de usuarios
type AuthService struct {
	userRepository interfaces.IUserRepository
	jwtSecret      []byte
	jwtExpiration  time.Duration
}

// NewAuthService crea una nueva instancia del servicio de autenticación
func NewAuthService(userRepository interfaces.IUserRepository, jwtSecret string) *AuthService {
	return &AuthService{
		userRepository: userRepository,
		jwtSecret:      []byte(jwtSecret),
		jwtExpiration:  24 * time.Hour, // 24 horas por defecto
	}
}

// AuthenticateAdmin autentica un administrador y retorna un token JWT
func (s *AuthService) AuthenticateAdmin(username, password string) (string, *user.User, error) {
	// Buscar usuario por nombre de usuario
	foundUser, err := s.userRepository.FindByUsername(username)
	if err != nil {
		return "", nil, errors.New("invalid credentials")
	}

	// Verificar que el usuario existe
	if foundUser == nil {
		return "", nil, errors.New("invalid credentials")
	}

	// Verificar que es un administrador
	if !foundUser.IsAdministrator() {
		return "", nil, errors.New("user is not an administrator")
	}

	// Verificar que está activo
	if !foundUser.IsActive() {
		return "", nil, errors.New("user account is not active")
	}

	// Validar contraseña
	if err := foundUser.ValidatePassword(password); err != nil {
		return "", nil, errors.New("invalid credentials")
	}

	// Generar token JWT
	token, err := s.generateJWT(foundUser)
	if err != nil {
		return "", nil, errors.New("failed to generate authentication token")
	}

	return token, foundUser, nil
}

// ValidateToken valida un token JWT y retorna los claims
func (s *AuthService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// generateJWT genera un token JWT para el usuario
func (s *AuthService) generateJWT(u *user.User) (string, error) {
	claims := &JWTClaims{
		UserID:   u.UserID(),
		Username: u.Username(),
		Role:     string(u.Role()),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.jwtExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "escritorio-remoto-backend",
			Subject:   u.UserID(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// JWTClaims define los claims personalizados para el JWT
type JWTClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}
