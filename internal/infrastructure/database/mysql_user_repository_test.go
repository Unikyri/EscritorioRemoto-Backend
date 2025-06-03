package database

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/user"
	"golang.org/x/crypto/bcrypt"
)

func setupTestDB(t *testing.T) *sql.DB {
	// Configuración para base de datos de prueba
	config := Config{
		Host:               "localhost",
		Port:               "3306",
		Database:           "escritorio_remoto_db",
		Username:           "app_user",
		Password:           "app_password",
		MaxConnections:     5,
		MaxIdleConnections: 2,
	}

	db, err := NewConnection(config)
	require.NoError(t, err, "Failed to connect to test database")

	return db
}

func TestMySQLUserRepository_FindByUsername_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMySQLUserRepository(db)

	// Act - buscar el usuario admin que ya existe en la BD
	foundUser, err := repo.FindByUsername("admin")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, foundUser)
	assert.Equal(t, "admin", foundUser.Username())
	assert.True(t, foundUser.IsAdministrator())
	assert.True(t, foundUser.IsActive())
}

func TestMySQLUserRepository_FindByUsername_NotFound(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMySQLUserRepository(db)

	// Act
	foundUser, err := repo.FindByUsername("nonexistent")

	// Assert
	assert.NoError(t, err)
	assert.Nil(t, foundUser)
}

func TestMySQLUserRepository_FindByID_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMySQLUserRepository(db)

	// Act - buscar el usuario admin por ID
	foundUser, err := repo.FindByID("admin-000-000-000-000000000001")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, foundUser)
	assert.Equal(t, "admin-000-000-000-000000000001", foundUser.UserID())
	assert.Equal(t, "admin", foundUser.Username())
	assert.True(t, foundUser.IsAdministrator())
}

func TestMySQLUserRepository_Create_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMySQLUserRepository(db)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("testpassword"), bcrypt.DefaultCost)
	testUser := user.NewUser(
		"test-user-id-123",
		"testuser",
		"192.168.1.100",
		string(hashedPassword),
		user.RoleClientUser,
	)

	// Act
	err := repo.Create(testUser)

	// Assert
	assert.NoError(t, err)

	// Verificar que se creó correctamente
	foundUser, err := repo.FindByUsername("testuser")
	assert.NoError(t, err)
	assert.NotNil(t, foundUser)
	assert.Equal(t, "testuser", foundUser.Username())
	assert.Equal(t, "test-user-id-123", foundUser.UserID())

	// Cleanup - eliminar usuario de prueba
	_, err = db.Exec("DELETE FROM users WHERE user_id = ?", "test-user-id-123")
	assert.NoError(t, err)
}

func TestMySQLUserRepository_Save_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMySQLUserRepository(db)

	// Crear usuario de prueba primero
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("testpassword"), bcrypt.DefaultCost)
	testUser := user.NewUser(
		"test-user-save-123",
		"testuserSave",
		"192.168.1.100",
		string(hashedPassword),
		user.RoleClientUser,
	)

	err := repo.Create(testUser)
	require.NoError(t, err)

	// Modificar usuario
	testUser.Deactivate()

	// Act
	err = repo.Save(testUser)

	// Assert
	assert.NoError(t, err)

	// Verificar que se actualizó correctamente
	foundUser, err := repo.FindByID("test-user-save-123")
	assert.NoError(t, err)
	assert.NotNil(t, foundUser)
	assert.False(t, foundUser.IsActive()) // Debe estar desactivado

	// Cleanup
	_, err = db.Exec("DELETE FROM users WHERE user_id = ?", "test-user-save-123")
	assert.NoError(t, err)
}

func TestMySQLUserRepository_PasswordValidation(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMySQLUserRepository(db)

	// Act - buscar admin y validar contraseña
	foundUser, err := repo.FindByUsername("admin")
	require.NoError(t, err)
	require.NotNil(t, foundUser)

	// Assert - validar contraseña correcta
	err = foundUser.ValidatePassword("password")
	assert.NoError(t, err)

	// Assert - validar contraseña incorrecta
	err = foundUser.ValidatePassword("wrongpassword")
	assert.Error(t, err)
}
