package userservice

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/user"
	"golang.org/x/crypto/bcrypt"
)

// MockUserRepository es un mock del repositorio de usuarios
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) FindByUsername(username string) (*user.User, error) {
	args := m.Called(username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) FindByID(userID string) (*user.User, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) Save(user *user.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) Create(user *user.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func TestAuthService_AuthenticateAdmin_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	authService := NewAuthService(mockRepo, "test-secret")

	// Crear usuario admin de prueba
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	adminUser := user.NewUser(
		"admin-id",
		"admin",
		"127.0.0.1",
		string(hashedPassword),
		user.RoleAdministrator,
	)

	mockRepo.On("FindByUsername", "admin").Return(adminUser, nil)

	// Act
	token, returnedUser, err := authService.AuthenticateAdmin("admin", "password")

	// Assert
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Equal(t, adminUser.UserID(), returnedUser.UserID())
	assert.Equal(t, adminUser.Username(), returnedUser.Username())
	assert.True(t, returnedUser.IsAdministrator())
	mockRepo.AssertExpectations(t)
}

func TestAuthService_AuthenticateAdmin_UserNotFound(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	authService := NewAuthService(mockRepo, "test-secret")

	mockRepo.On("FindByUsername", "nonexistent").Return(nil, nil)

	// Act
	token, returnedUser, err := authService.AuthenticateAdmin("nonexistent", "password")

	// Assert
	assert.Error(t, err)
	assert.Empty(t, token)
	assert.Nil(t, returnedUser)
	assert.Contains(t, err.Error(), "invalid credentials")
	mockRepo.AssertExpectations(t)
}

func TestAuthService_AuthenticateAdmin_WrongPassword(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	authService := NewAuthService(mockRepo, "test-secret")

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	adminUser := user.NewUser(
		"admin-id",
		"admin",
		"127.0.0.1",
		string(hashedPassword),
		user.RoleAdministrator,
	)

	mockRepo.On("FindByUsername", "admin").Return(adminUser, nil)

	// Act
	token, returnedUser, err := authService.AuthenticateAdmin("admin", "wrong-password")

	// Assert
	assert.Error(t, err)
	assert.Empty(t, token)
	assert.Nil(t, returnedUser)
	assert.Contains(t, err.Error(), "invalid credentials")
	mockRepo.AssertExpectations(t)
}

func TestAuthService_AuthenticateAdmin_NotAdministrator(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	authService := NewAuthService(mockRepo, "test-secret")

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	clientUser := user.NewUser(
		"client-id",
		"client",
		"127.0.0.1",
		string(hashedPassword),
		user.RoleClientUser,
	)

	mockRepo.On("FindByUsername", "client").Return(clientUser, nil)

	// Act
	token, returnedUser, err := authService.AuthenticateAdmin("client", "password")

	// Assert
	assert.Error(t, err)
	assert.Empty(t, token)
	assert.Nil(t, returnedUser)
	assert.Contains(t, err.Error(), "not an administrator")
	mockRepo.AssertExpectations(t)
}

func TestAuthService_AuthenticateAdmin_InactiveUser(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	authService := NewAuthService(mockRepo, "test-secret")

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	adminUser := user.NewUser(
		"admin-id",
		"admin",
		"127.0.0.1",
		string(hashedPassword),
		user.RoleAdministrator,
	)
	adminUser.Deactivate() // Desactivar usuario

	mockRepo.On("FindByUsername", "admin").Return(adminUser, nil)

	// Act
	token, returnedUser, err := authService.AuthenticateAdmin("admin", "password")

	// Assert
	assert.Error(t, err)
	assert.Empty(t, token)
	assert.Nil(t, returnedUser)
	assert.Contains(t, err.Error(), "not active")
	mockRepo.AssertExpectations(t)
}

func TestAuthService_AuthenticateAdmin_RepositoryError(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	authService := NewAuthService(mockRepo, "test-secret")

	mockRepo.On("FindByUsername", "admin").Return(nil, errors.New("database error"))

	// Act
	token, returnedUser, err := authService.AuthenticateAdmin("admin", "password")

	// Assert
	assert.Error(t, err)
	assert.Empty(t, token)
	assert.Nil(t, returnedUser)
	assert.Contains(t, err.Error(), "invalid credentials")
	mockRepo.AssertExpectations(t)
}

func TestAuthService_ValidateToken_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	authService := NewAuthService(mockRepo, "test-secret")

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	adminUser := user.NewUser(
		"admin-id",
		"admin",
		"127.0.0.1",
		string(hashedPassword),
		user.RoleAdministrator,
	)

	mockRepo.On("FindByUsername", "admin").Return(adminUser, nil)

	// Generar token válido
	token, _, _ := authService.AuthenticateAdmin("admin", "password")

	// Act
	claims, err := authService.ValidateToken(token)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, "admin-id", claims.UserID)
	assert.Equal(t, "admin", claims.Username)
	assert.Equal(t, "ADMINISTRATOR", claims.Role)
	mockRepo.AssertExpectations(t)
}

func TestAuthService_ValidateToken_InvalidToken(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	authService := NewAuthService(mockRepo, "test-secret")

	// Act
	claims, err := authService.ValidateToken("invalid-token")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestAuthService_ValidateToken_ExpiredToken(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	authService := NewAuthService(mockRepo, "test-secret")

	// Configurar expiración muy corta para la prueba
	authService.jwtExpiration = time.Millisecond

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	adminUser := user.NewUser(
		"admin-id",
		"admin",
		"127.0.0.1",
		string(hashedPassword),
		user.RoleAdministrator,
	)

	mockRepo.On("FindByUsername", "admin").Return(adminUser, nil)

	// Generar token que expirará inmediatamente
	token, _, _ := authService.AuthenticateAdmin("admin", "password")

	// Esperar a que expire
	time.Sleep(time.Millisecond * 10)

	// Act
	claims, err := authService.ValidateToken(token)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, claims)
	mockRepo.AssertExpectations(t)
}
