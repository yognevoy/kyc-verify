package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleApplicant Role = "applicant"
	RoleReviewer  Role = "reviewer"
)

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Role         Role
	CreatedAt    time.Time
}

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
)
