package interfaces

import (
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/user"
)

// IUserRepository define la interfaz para el repositorio de usuarios
type IUserRepository interface {
	// FindByUsername busca un usuario por su nombre de usuario
	FindByUsername(username string) (*user.User, error)

	// FindByID busca un usuario por su ID
	FindByID(userID string) (*user.User, error)

	// Save guarda o actualiza un usuario
	Save(user *user.User) error

	// Create crea un nuevo usuario
	Create(user *user.User) error
}
