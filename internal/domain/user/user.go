package user

import (
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Role string

const (
	RoleAdministrator Role = "ADMINISTRATOR"
	RoleClientUser    Role = "CLIENT_USER"
)

type User struct {
	userID         string
	username       string
	ip             string
	hashedPassword string
	role           Role
	isActive       bool
	createdAt      time.Time
	updatedAt      time.Time
}

func NewUser(userID, username, ip, hashedPassword string, role Role) *User {
	now := time.Now()
	return &User{
		userID:         userID,
		username:       username,
		ip:             ip,
		hashedPassword: hashedPassword,
		role:           role,
		isActive:       true,
		createdAt:      now,
		updatedAt:      now,
	}
}

func (u *User) UserID() string {
	return u.userID
}

func (u *User) Username() string {
	return u.username
}

func (u *User) IP() string {
	return u.ip
}

func (u *User) HashedPassword() string {
	return u.hashedPassword
}

func (u *User) Role() Role {
	return u.role
}

func (u *User) IsActive() bool {
	return u.isActive
}

func (u *User) IsAdministrator() bool {
	return u.role == RoleAdministrator
}

func (u *User) ValidatePassword(password string) error {
	if !u.isActive {
		return errors.New("user is not active")
	}

	return bcrypt.CompareHashAndPassword([]byte(u.hashedPassword), []byte(password))
}

func (u *User) Deactivate() {
	u.isActive = false
	u.updatedAt = time.Now()
}

type UserSnapshot struct {
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (u *User) ToSnapshot() UserSnapshot {
	return UserSnapshot{
		UserID:    u.userID,
		Username:  u.username,
		Role:      string(u.role),
		IsActive:  u.isActive,
		CreatedAt: u.createdAt,
		UpdatedAt: u.updatedAt,
	}
}
