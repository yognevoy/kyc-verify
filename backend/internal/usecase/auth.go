package usecase

import (
	"context"
	"errors"
	"fmt"
	"net/mail"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"kyc-verify/internal/domain"
)

var (
	ErrInvalidEmail       = errors.New("invalid email")
	ErrPasswordTooShort   = errors.New("password must be at least 8 characters")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

const minPasswordLength = 8

type TokenIssuer interface {
	Issue(userID uuid.UUID, role string) (string, error)
}

type AuthUsecase struct {
	users  domain.UserRepository
	tokens TokenIssuer
}

func NewAuthUsecase(users domain.UserRepository, tokens TokenIssuer) *AuthUsecase {
	return &AuthUsecase{users: users, tokens: tokens}
}

func (a *AuthUsecase) Register(ctx context.Context, email, password string) (string, error) {
	if _, err := mail.ParseAddress(email); err != nil {
		return "", ErrInvalidEmail
	}
	if len(password) < minPasswordLength {
		return "", ErrPasswordTooShort
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}

	user := &domain.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hash),
		Role:         domain.RoleApplicant,
	}

	if err := a.users.Create(ctx, user); err != nil {
		return "", err
	}

	return a.tokens.Issue(user.ID, string(user.Role))
}

func (a *AuthUsecase) Login(ctx context.Context, email, password string) (string, error) {
	user, err := a.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return "", ErrInvalidCredentials
		}
		return "", fmt.Errorf("get user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	return a.tokens.Issue(user.ID, string(user.Role))
}
