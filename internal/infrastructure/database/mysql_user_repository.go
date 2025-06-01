package database

import (
	"database/sql"
	"errors"
	"time"

	"github.com/unikyri/escritorio-remoto-backend/internal/application/interfaces"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/user"
)

// MySQLUserRepository implementa IUserRepository para MySQL
type MySQLUserRepository struct {
	db *sql.DB
}

// NewMySQLUserRepository crea una nueva instancia del repositorio MySQL
func NewMySQLUserRepository(db *sql.DB) interfaces.IUserRepository {
	return &MySQLUserRepository{
		db: db,
	}
}

// FindByUsername busca un usuario por su nombre de usuario
func (r *MySQLUserRepository) FindByUsername(username string) (*user.User, error) {
	query := `
		SELECT user_id, username, ip, hashed_password, role, is_active, created_at, updated_at
		FROM users 
		WHERE username = ? AND is_active = TRUE
	`

	var userID, dbUsername, ip, hashedPassword, roleStr string
	var isActive bool
	var createdAt, updatedAt time.Time

	err := r.db.QueryRow(query, username).Scan(
		&userID, &dbUsername, &ip, &hashedPassword, &roleStr, &isActive, &createdAt, &updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Usuario no encontrado
		}
		return nil, err
	}

	// Convertir string a Role
	var role user.Role
	switch roleStr {
	case "ADMINISTRATOR":
		role = user.RoleAdministrator
	case "CLIENT_USER":
		role = user.RoleClientUser
	default:
		return nil, errors.New("invalid user role")
	}

	// Crear usuario desde datos de BD
	foundUser := user.NewUser(userID, dbUsername, ip, hashedPassword, role)

	// Actualizar campos que pueden ser diferentes
	if !isActive {
		foundUser.Deactivate()
	}

	return foundUser, nil
}

// FindByID busca un usuario por su ID
func (r *MySQLUserRepository) FindByID(userID string) (*user.User, error) {
	query := `
		SELECT user_id, username, ip, hashed_password, role, is_active, created_at, updated_at
		FROM users 
		WHERE user_id = ?
	`

	var dbUserID, username, ip, hashedPassword, roleStr string
	var isActive bool
	var createdAt, updatedAt time.Time

	err := r.db.QueryRow(query, userID).Scan(
		&dbUserID, &username, &ip, &hashedPassword, &roleStr, &isActive, &createdAt, &updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Usuario no encontrado
		}
		return nil, err
	}

	// Convertir string a Role
	var role user.Role
	switch roleStr {
	case "ADMINISTRATOR":
		role = user.RoleAdministrator
	case "CLIENT_USER":
		role = user.RoleClientUser
	default:
		return nil, errors.New("invalid user role")
	}

	// Crear usuario desde datos de BD
	foundUser := user.NewUser(dbUserID, username, ip, hashedPassword, role)

	// Actualizar campos que pueden ser diferentes
	if !isActive {
		foundUser.Deactivate()
	}

	return foundUser, nil
}

// Save guarda o actualiza un usuario existente
func (r *MySQLUserRepository) Save(u *user.User) error {
	query := `
		UPDATE users 
		SET username = ?, ip = ?, hashed_password = ?, role = ?, is_active = ?, updated_at = ?
		WHERE user_id = ?
	`

	_, err := r.db.Exec(query,
		u.Username(),
		u.IP(),
		u.HashedPassword(),
		string(u.Role()),
		u.IsActive(),
		time.Now(),
		u.UserID(),
	)

	return err
}

// Create crea un nuevo usuario
func (r *MySQLUserRepository) Create(u *user.User) error {
	query := `
		INSERT INTO users (user_id, username, ip, hashed_password, role, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	_, err := r.db.Exec(query,
		u.UserID(),
		u.Username(),
		u.IP(),
		u.HashedPassword(),
		string(u.Role()),
		u.IsActive(),
		now,
		now,
	)

	return err
}
