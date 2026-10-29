package domain

import (
	"errors"
	"time"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
)

type UserRole string

const (
	UserRoleOwner UserRole = "owner"
)

type User struct {
	ID             int64
	OrganizationID int64
	Email          string
	PasswordHash   string
	Role           UserRole
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
